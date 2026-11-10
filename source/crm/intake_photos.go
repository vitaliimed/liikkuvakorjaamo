package crm

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Only references already stored on the authorized job are accepted. No arbitrary URL fetch.
func intakeObject(ref, bucket, prefix string) (string, error) {
	u, e := url.Parse(ref)
	prefix = strings.Trim(prefix, "/") + "/"
	if e != nil || bucket == "" || u.Scheme != "gs" || u.Host != bucket || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", errors.New("Недопустимое хранилище")
	}
	object := strings.TrimPrefix(u.Path, "/")
	if !strings.HasPrefix(object, prefix) || strings.Contains(object, "/../") || strings.Contains(object, "/./") {
		return "", errors.New("Недопустимый объект")
	}
	return object, nil
}
func (s *Server) intakePhoto(w http.ResponseWriter, r *http.Request, j Job, index string) {
	if r.Method != "GET" {
		fail(w, 405, "Method not allowed")
		return
	}
	i, e := strconv.Atoi(index)
	if e != nil || i < 0 || i >= len(j.IntakePhotos) {
		fail(w, 404, "Фото не найдено")
		return
	}
	bucket := os.Getenv("GCS_BUCKET_NAME")
	prefix := os.Getenv("GCS_PHOTO_PREFIX")
	if prefix == "" {
		prefix = "lead-photos"
	}
	object, e := intakeObject(j.IntakePhotos[i], bucket, prefix)
	if e != nil {
		fail(w, 404, "Фото недоступно")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	client, e := storage.NewClient(ctx)
	if e != nil {
		fail(w, 503, "Хранилище недоступно")
		return
	}
	defer client.Close()
	rd, e := client.Bucket(bucket).Object(object).NewReader(ctx)
	if e != nil {
		fail(w, 404, "Фото не найдено")
		return
	}
	defer rd.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(rd, head)
	kind := http.DetectContentType(head[:n])
	if kind != "image/jpeg" && kind != "image/png" && kind != "image/webp" {
		fail(w, 415, "Неподдерживаемый формат фото")
		return
	}
	w.Header().Set("Content-Type", kind)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "inline")
	_, _ = w.Write(head[:n])
	_, _ = io.Copy(w, io.LimitReader(rd, 10<<20))
}
