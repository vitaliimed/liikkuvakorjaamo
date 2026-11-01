package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPineAppearanceAndIndexing(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	settings := &memorySettingsStore{state: SiteSettings{ProFontSize: 20, ProAccent: "#203832", LogoURL: "/static/img/logo-light.svg", LogoDarkURL: "/static/img/logo-dark.svg", LandingRobotsMeta: "index, follow"}}
	f.cfg.SettingsStore = settings
	renderer, err := newTemplateRenderer()
	if err != nil {
		t.Fatal(err)
	}
	f.handler = newServer(f.cfg, renderer).Handler
	for _, route := range []string{"/", "/en", "/hinnasto", "/palvelualue", "/vikadiagnostiikka", "/privacy", "/static/seo/hinnasto.html"} {
		w := f.request("GET", route, nil, nil)
		if w.Code != 200 {
			t.Fatalf("%s: %d", route, w.Code)
		}
		if w.Header().Get("X-Robots-Tag") != "noindex, follow" || !strings.Contains(w.Body.String(), `content="noindex, follow"`) {
			t.Fatalf("index protection missing: %s", route)
		}
	}
	robots := f.request("GET", "/robots.txt", nil, nil).Body.String()
	if strings.Contains(robots, "Disallow: /\n") {
		t.Fatal("Crawling blocked: noindex unreadable")
	}
	if strings.Contains(f.request("GET", "/sitemap.xml", nil, nil).Body.String(), "<loc>") {
		t.Fatal("Blocked site still advertises URLs")
	}
	body := f.request("GET", "/", nil, nil).Body.String()
	if !strings.Contains(body, "--font-scale:1.1111") || !strings.Contains(body, "--site-accent:#203832") {
		t.Fatal("Saved appearance not injected")
	}
	if strings.Contains(body, `<span>Liikkuva<span class="brand-small">`) {
		t.Fatal("Duplicate logo text")
	}
	settings.state.SearchIndexingEnabled = true
	w := f.request("GET", "/", nil, nil)
	if w.Header().Get("X-Robots-Tag") != "" || !strings.Contains(w.Body.String(), `content="index, follow"`) {
		t.Fatal("Indexing cannot be re-enabled")
	}
	if dir := os.Getenv("LIIKKUVA_QA_DIR"); dir != "" {
		os.MkdirAll(dir, 0700)
		for _, size := range []int{16, 18, 20, 22} {
			settings.state.ProFontSize = size
			for _, theme := range []string{"light", "dark"} {
				settings.state.ProTheme = theme
				w := f.request("GET", "/", nil, nil)
				if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("appearance-%d-%s.html", size, theme)), w.Body.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
