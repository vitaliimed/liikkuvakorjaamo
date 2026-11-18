package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type unifiedPriceStore struct{ value SiteSettings }
func (s unifiedPriceStore) Load(context.Context) (SiteSettings, error) { return s.value, nil }
func (s unifiedPriceStore) Update(context.Context, func(SiteSettings) (SiteSettings, error)) error { return nil }

func TestAdminFeesAcrossLandingAndSEOAndThankYou(t *testing.T) {
	renderer, err := newTemplateRenderer()
	if err != nil { t.Fatal(err) }
	cfg := Config{
		SiteURL: "https://liikkuvakorjaamo.fi",
		DefaultLanguage: "fi",
		SettingsStore: unifiedPriceStore{SiteSettings{ProVisitEUR: 92, ProDiagnosticEUR: 83}},
	}
	h := liikkuvaPro(cfg, liikkuvaSecurity(cfg, newRouter(cfg, renderer)))
	for _, path := range []string{"/", "/en", "/hinnasto", "/vikadiagnostiikka", "/thank-you-fi", "/thank-you-en", "/auton-korjaus-kotona"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://liikkuvakorjaamo.fi"+path, nil))
			if w.Code != http.StatusOK { t.Fatalf("HTTP %d: %s", w.Code, w.Body.String()) }
			body := w.Body.String()
			if !strings.Contains(body, "92") || !strings.Contains(body, "83") {
				t.Fatalf("%s missing configured visit/diagnostics prices", path)
			}
			if strings.Contains(body, "{visit}") || strings.Contains(body, "{diagnostic}") || strings.Contains(body, "{total}") {
				t.Fatalf("%s contains raw pricing placeholder", path)
			}
		})
	}
}

func TestCustomEmailPriceTokensUseAdminWithoutRewritingQuotes(t *testing.T) {
	prices := SiteSettings{ProVisitEUR: 50, ProDiagnosticEUR: 65}
	input := "Käynti {visit} €, vianmääritys {diagnostic} €, yhdessä {total} €. Varaosat 75 €."
	got := liikkuvaPricePlaceholders(input, prices)
	for _, want := range []string{"Käynti 50 €", "vianmääritys 65 €", "yhdessä 115 €", "Varaosat 75 €."} {
		if !strings.Contains(got, want) { t.Fatalf("missing %q in %q", want, got) }
	}
}
