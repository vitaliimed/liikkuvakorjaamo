package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServiceLandingUsesSameLeadFlow(t *testing.T) {
	cfg := Config{SiteURL: "https://liikkuvakorjaamo.fi", SettingsStore: seoSettingsStore{SiteSettings{}}}
	handler := liikkuvaPro(cfg, liikkuvaSecurity(cfg, nil))
	for path := range servicePages {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", cfg.SiteURL+path, nil))
		html := w.Body.String()
		for _, part := range []string{`id="leadForm"`, `id="footerLeadFormMount"`, `rel="canonical" href="` + cfg.SiteURL + path + `"`, `class="container service-detail"`} {
			if w.Code != 200 || !strings.Contains(html, part) {
				t.Fatalf("%s missing %s (HTTP %d)", path, part, w.Code)
			}
		}
		if strings.Contains(html, `hreflang="en"`) {
			t.Fatalf("%s advertises an unrelated translation", path)
		}
	}
}
func TestReviewExamplesStayPrivate(t *testing.T) {
	settings := normalizeAdminSettingsForTemplate(SiteSettings{})
	if len(settings.Testimonials) != 6 || len(buildTestimonialsFromSettings(settings, "fi")) != 0 {
		t.Fatal("six private examples expected")
	}
}
func TestOfferEndsAtHelsinkiMidnight(t *testing.T) {
	if got := time.Unix(campaignDeadline("2026-12-31"), 0).UTC().Format(time.RFC3339); got != "2026-12-31T22:00:00Z" {
		t.Fatal(got)
	}
	if campaignDeadline("2026-02-31") != 0 {
		t.Fatal("invalid deadline accepted")
	}
}

// Write isolated, rendered fixtures only when the browser smoke test requests them.
func TestRenderBrowserFixtures(t *testing.T) {
	dir := os.Getenv("LANDING_FIXTURE_DIR")
	if dir == "" {
		t.Skip("browser fixture export disabled")
	}
	settings := SiteSettings{ExitEnabled: true, ExitDate: "2099-12-31", ExitTitleEN: "Request a visit", ExitTitleFI: "Pyydä käyntiä", GA4MeasurementID: "G-TEST"}
	for i := 0; i < 6; i++ {
		settings.Testimonials = append(settings.Testimonials, AdminTestimonialItem{Name: "Fixture", QuoteEN: "Test review", QuoteFI: "Testipalaute", Rating: 5})
	}
	cfg := Config{SiteURL: "https://liikkuvakorjaamo.fi", SettingsStore: seoSettingsStore{settings}}
	renderer, err := newTemplateRenderer()
	if err != nil {
		t.Fatal(err)
	}
	handler := newServer(cfg, renderer).Handler
	for _, item := range []struct{ path, file string }{{"/", "index.html"}, {"/en", "en.html"}, {"/palvelualue", "area.html"}} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", cfg.SiteURL+item.path, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d", item.path, w.Code)
		}
		if err := os.WriteFile(filepath.Join(dir, item.file), w.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCustomDraftCanOnlyRenderInAdmin(t *testing.T) {
	draft := ServicePage{Path: "/helsinki", Language: "fi", Heading: "Kaupungin sivu", Title: "Helsinki draft", Description: "Draft", Intro: "Esikatselu"}
	cfg := Config{SiteURL: "https://liikkuvakorjaamo.fi", SettingsStore: seoSettingsStore{SiteSettings{ServicePages: []ServicePage{draft}}}}
	renderer, err := newTemplateRenderer()
	if err != nil {
		t.Fatal(err)
	}
	h := newServer(cfg, renderer).Handler
	for _, path := range []string{"/helsinki", "/sitemap.xml", "/admin/seo/preview?path=/helsinki"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", cfg.SiteURL+path, nil))
		if path == "/helsinki" && w.Code != 404 {
			t.Fatal(w.Code)
		}
		if strings.Contains(w.Body.String(), draft.Heading) || (path == "/sitemap.xml" && strings.Contains(w.Body.String(), draft.Path)) {
			t.Fatal("Draft visible at " + path)
		}
	}
}

func TestEditorSectionOrderPreservesForms(t *testing.T) {
	var out strings.Builder
	page := `<main><form id="leadForm"></form><section data-service-block="areas">Map</section><section data-service-block="faq">Questions</section><section data-service-block="stats">Unused</section><section id="footer-lead">Form</section></main>`
	if err := renderServiceLayout(&out, strings.NewReader(page), "faq areas"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "Unused") || strings.Index(got, "Questions") > strings.Index(got, "Map") || !strings.Contains(got, "leadForm") || !strings.Contains(got, "footer-lead") {
		t.Fatal(got)
	}
}
