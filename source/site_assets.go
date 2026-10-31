package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

const maxSiteAssetBytes = 8 << 20
const siteAssetPrefix = "site-assets/"

var siteAssetName = regexp.MustCompile(`^[a-f0-9]{64}\.(png|jpg|webp|svg|ico)$`)

type siteAssetStore interface {
	Put(context.Context, string, string, []byte) error
	Get(context.Context, string) ([]byte, string, error)
}
type gcsSiteAssetStore struct{ bucket string }

func (s gcsSiteAssetStore) Put(ctx context.Context, name, kind string, data []byte) error {
	c, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	w := c.Bucket(s.bucket).Object(siteAssetPrefix + name).NewWriter(ctx)
	w.ContentType = kind
	w.CacheControl = "public, max-age=31536000, immutable"
	w.ChunkSize = 0
	if _, err = w.Write(data); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}
func (s gcsSiteAssetStore) Get(ctx context.Context, name string) ([]byte, string, error) {
	c, err := storage.NewClient(ctx)
	if err != nil {
		return nil, "", err
	}
	defer c.Close()
	r, err := c.Bucket(s.bucket).Object(siteAssetPrefix + name).NewReader(ctx)
	if err != nil {
		return nil, "", err
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, maxSiteAssetBytes+1))
	if err != nil || len(data) > maxSiteAssetBytes {
		return nil, "", fmt.Errorf("asset is too large")
	}
	return data, r.Attrs.ContentType, nil
}
func assetStorage(cfg Config) siteAssetStore {
	if cfg.AssetStore != nil {
		return cfg.AssetStore
	}
	if strings.TrimSpace(cfg.GCSBucketName) == "" {
		return nil
	}
	return gcsSiteAssetStore{cfg.GCSBucketName}
}

func validateSiteAsset(data []byte) (string, string, error) {
	if len(data) == 0 || len(data) > maxSiteAssetBytes {
		return "", "", fmt.Errorf("Выберите изображение размером до 8 МБ.")
	}
	kind := http.DetectContentType(data)
	switch kind {
	case "image/png", "image/jpeg":
		dimensions, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || dimensions.Width < 1 || dimensions.Height < 1 || dimensions.Width > 8000 || dimensions.Height > 8000 || dimensions.Width*dimensions.Height > 16000000 {
			return "", "", fmt.Errorf("Изображение повреждено или слишком большое: максимум 16 мегапикселей и 8000 пикселей по стороне.")
		}
		if kind == "image/png" {
			return "png", kind, nil
		}
		return "jpg", kind, nil
	case "image/webp":
		return "webp", kind, nil
	case "image/x-icon", "image/vnd.microsoft.icon":
		return "ico", "image/x-icon", nil
	}
	if len(data) <= 256<<10 && validateStaticSVG(data) == nil {
		return "svg", "image/svg+xml", nil
	}
	return "", "", fmt.Errorf("Поддерживаются PNG, JPG, WebP, ICO и SVG без скриптов, стилей и внешних ссылок.")
}

// Keep SVGs static: scripts, HTML, CSS, animation and external resources are rejected.
func validateStaticSVG(data []byte) error {
	elements := map[string]bool{}
	for _, name := range strings.Fields("svg g path circle rect line polyline polygon ellipse defs clipPath mask linearGradient radialGradient stop title desc use symbol") {
		elements[name] = true
	}
	attrs := map[string]bool{}
	for _, name := range strings.Fields("xmlns version width height viewBox fill fill-opacity fill-rule stroke stroke-width stroke-linecap stroke-linejoin stroke-miterlimit stroke-dasharray stroke-dashoffset stroke-opacity clip-rule clip-path mask opacity transform d x y x1 y1 x2 y2 cx cy r rx ry points id offset stop-color stop-opacity gradientUnits gradientTransform spreadMethod preserveAspectRatio href") {
		attrs[name] = true
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	depth, roots := 0, 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch token := token.(type) {
		case xml.StartElement:
			if !elements[token.Name.Local] || (token.Name.Space != "" && token.Name.Space != "http://www.w3.org/2000/svg") {
				return fmt.Errorf("unsupported SVG element")
			}
			if depth == 0 {
				roots++
				if token.Name.Local != "svg" || roots != 1 {
					return fmt.Errorf("expected one SVG root")
				}
			}
			depth++
			for _, attr := range token.Attr {
				if !attrs[attr.Name.Local] || strings.HasPrefix(strings.ToLower(attr.Name.Local), "on") {
					return fmt.Errorf("unsupported SVG attribute")
				}
				if attr.Name.Local == "href" && !strings.HasPrefix(attr.Value, "#") {
					return fmt.Errorf("external SVG reference")
				}
				v := strings.ToLower(strings.TrimSpace(attr.Value))
				if strings.Contains(v, "url(") && !regexp.MustCompile(`^url\(#[a-zA-Z0-9_-]+\)$`).MatchString(v) {
					return fmt.Errorf("external SVG resource")
				}
			}
		case xml.EndElement:
			depth--
		case xml.Directive:
			return fmt.Errorf("SVG directives are not allowed")
		case xml.ProcInst:
			if token.Target != "xml" {
				return fmt.Errorf("SVG instructions are not allowed")
			}
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(token)) != "" {
				return fmt.Errorf("text outside SVG")
			}
		}
	}
	if roots != 1 || depth != 0 {
		return fmt.Errorf("invalid SVG")
	}
	return nil
}

func (h *AdminHandler) UploadSiteAsset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !isAdminAuthenticated(h.cfg, r) {
		http.Error(w, "Войдите в кабинет.", 401)
		return
	}
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSiteAssetBytes+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "message": "Размер изображения — до 8 МБ."})
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	// adminAccessGuard validates the Origin and session-bound access_csrf before this handler.
	f, _, err := r.FormFile("asset")
	if err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "message": "Выберите файл."})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxSiteAssetBytes+1))
	if err != nil {
		http.Error(w, "Cannot read file", 400)
		return
	}
	ext, kind, err := validateSiteAsset(data)
	if err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	store := assetStorage(h.cfg)
	if store == nil {
		writeJSON(w, 503, map[string]any{"ok": false, "message": "Хранилище изображений ещё не подключено. Выполните update.py --configure-media из инструкции."})
		return
	}
	digest := sha256.Sum256(data)
	name := hex.EncodeToString(digest[:]) + "." + ext
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err = store.Put(ctx, name, kind, data); err != nil {
		writeJSON(w, 503, map[string]any{"ok": false, "message": "Не удалось сохранить изображение. Проверьте доступ сервиса к хранилищу."})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "url": "/media/site/" + name})
}
func serveSiteAsset(cfg Config, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != "GET" && r.Method != "HEAD" {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", 405)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/media/site/")
	store := assetStorage(cfg)
	if !siteAssetName.MatchString(name) || store == nil {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	data, _, err := store.Get(ctx, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ext, kind, err := validateSiteAsset(data)
	if err != nil || !strings.HasSuffix(name, "."+ext) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", kind)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}
