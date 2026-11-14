package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

type seoSettingsStore struct{ settings SiteSettings }

func (s seoSettingsStore) Load(context.Context) (SiteSettings, error) { return s.settings, nil }
func (s seoSettingsStore) Update(context.Context, func(SiteSettings) (SiteSettings, error)) error {
	return nil
}

func TestSEOPublicRoutes(t *testing.T) {
	cfg := Config{SiteURL: "https://liikkuvakorjaamo.fi", SettingsStore: seoSettingsStore{SiteSettings{ProVisitEUR: 75, ProDiagnosticEUR: 60}}}
	h := liikkuvaPro(cfg, liikkuvaSecurity(cfg, http.NotFoundHandler()))
	raw, err := os.ReadFile("static/sitemap.xml")
	if err != nil {
		t.Fatal(err)
	}
	var sitemap struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err = xml.Unmarshal(raw, &sitemap); err != nil {
		t.Fatal(err)
	}
	if len(sitemap.URLs) != 8 {
		t.Fatalf("want 8 indexable URLs, got %d", len(sitemap.URLs))
	}
	titles := map[string]bool{}
	for _, entry := range sitemap.URLs {
		path := strings.TrimPrefix(entry.Loc, "https://liikkuvakorjaamo.fi")
		if path == "/" || path == "/en" {
			continue
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", entry.Loc, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d", path, w.Code)
		}
		body := w.Body.String()
		if strings.Contains(w.Header().Get("X-Robots-Tag"), "noindex") {
			t.Fatalf("production noindex %s", path)
		}
		if !strings.Contains(body, `rel="canonical" href="`+entry.Loc+`"`) {
			t.Fatalf("canonical %s", path)
		}
		if strings.Count(body, "<h1>") != 1 {
			t.Fatalf("h1 %s", path)
		}
		title := regexp.MustCompile(`<title>(.*?)</title>`).FindString(body)
		if title == "" || titles[title] {
			t.Fatalf("missing/duplicate title %s", path)
		}
		titles[title] = true
		for _, token := range []string{"{visit}", "{total}", "{diagnostic}"} {
			if strings.Contains(body, token) {
				t.Fatalf("unexpanded price %s", path)
			}
		}
		for _, m := range regexp.MustCompile(`<script type="application/ld\+json">(.*?)</script>`).FindAllStringSubmatch(body, -1) {
			if !json.Valid([]byte(m[1])) {
				t.Fatalf("invalid JSON-LD %s", path)
			}
		}
		if path == "/akun-vaihto" && (!strings.Contains(body, "75 €") || !strings.Contains(body, "60 €") || !strings.Contains(body, "135 €")) {
			t.Fatal("admin prices not applied")
		}
		for _, method := range []string{"HEAD", "POST"} {
			w = httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(method, entry.Loc, nil))
			if method == "HEAD" && (w.Code != 200 || w.Body.Len() != 0) {
				t.Fatalf("HEAD %s", path)
			}
			if method == "POST" && w.Code != 405 {
				t.Fatalf("POST %s: %d", path, w.Code)
			}
		}
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", entry.Loc+"/?utm_source=meta", nil))
		if w.Code != 308 || w.Header().Get("Location") != path+"?utm_source=meta" {
			t.Fatalf("redirect %s", path)
		}
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "https://preview.run.app"+path, nil))
		if !strings.Contains(w.Header().Get("X-Robots-Tag"), "noindex") {
			t.Fatalf("preview indexable %s", path)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "https://liikkuvakorjaamo.fi/not-a-service", nil))
	if w.Code != 404 {
		t.Fatal("unknown path should 404")
	}
}

func TestSEOLaunchRobotsAndLegacyDefaults(t *testing.T) {
	t.Setenv("LIIKKUVA_SEO_NOINDEX", "")
	for _, host := range []string{"liikkuvakorjaamo.fi", "www.liikkuvakorjaamo.fi", "tag---preview.run.app", "project.web.app"} {
		req := httptest.NewRequest("GET", "https://"+host+"/", nil)
		want := host == "liikkuvakorjaamo.fi" || host == "www.liikkuvakorjaamo.fi"
		if liikkuvaIndexable(req) != want {
			t.Fatal(host)
		}
	}
	t.Setenv("LIIKKUVA_SEO_NOINDEX", "1")
	if !strings.Contains(liikkuvaLandingRobots(httptest.NewRequest("GET", "https://liikkuvakorjaamo.fi/", nil)), "noindex") {
		t.Fatal("opt-out ignored")
	}
	s := liikkuvaSettings(SiteSettings{LandingSEOTitleFI: "Liikkuva autokorjaamo Uusimaa | Korjaus auton luona"})
	if !strings.Contains(s.LandingSEOTitleFI, "Helsinki") {
		t.Fatal("legacy title not migrated")
	}
	s = liikkuvaSettings(SiteSettings{LandingSEOTitleFI: "Custom title"})
	if s.LandingSEOTitleFI != "Custom title" {
		t.Fatal("custom title overwritten")
	}
	fallback := fallbackSitemapXML("https://preview.run.app")
	if strings.Contains(fallback, "thank-you") || strings.Contains(fallback, "privacy") || strings.Contains(fallback, "preview.run.app") {
		t.Fatal("bad sitemap fallback")
	}
}

func TestSEOLandingReplacesStoredNoindex(t *testing.T) {
	t.Setenv("LIIKKUVA_SEO_NOINDEX", "")
	renderer, err := newTemplateRenderer()
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{SiteURL: "https://liikkuvakorjaamo.fi", SettingsStore: seoSettingsStore{SiteSettings{LandingRobotsMeta: "noindex, follow", LandingSEOTitleFI: "Liikkuva autokorjaamo Uusimaa | Korjaus auton luona"}}}
	handler := NewPageHandler(cfg, renderer)
	for _, path := range []string{"/", "/en"} {
		for _, host := range []string{"liikkuvakorjaamo.fi", "preview.run.app"} {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "https://"+host+path+"?utm_source=test", nil)
			if path == "/" {
				handler.LandingFI(w, r)
			} else {
				handler.LandingEN(w, r)
			}
			body := w.Body.String()
			if w.Code != 200 {
				t.Fatal(w.Code)
			}
			want := `name="robots" content="index, follow, max-image-preview:large"`
			if host != "liikkuvakorjaamo.fi" {
				want = `name="robots" content="noindex, follow"`
			}
			if !strings.Contains(body, want) {
				t.Fatalf("robots %s%s", host, path)
			}
			if !strings.Contains(body, `rel="canonical" href="https://liikkuvakorjaamo.fi`+path+`"`) {
				t.Fatal("canonical")
			}
			if !strings.Contains(body, "Helsinki") {
				t.Fatal("new metadata missing")
			}
			if strings.Count(body, `id="leadForm"`) != 1 {
				t.Fatal("lead form regression")
			}
		}
	}
}
