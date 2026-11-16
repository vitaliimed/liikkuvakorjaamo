package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestThankYouInstagramURLIsSafe(t *testing.T) {
	for _, tc := range []struct {
		input, want string
	}{
		{"https://www.instagram.com/liikkuva.example/", "https://www.instagram.com/liikkuva.example/"},
		{" https://instagram.com/liikkuva.example ", "https://instagram.com/liikkuva.example"},
		{"", ""},
		{"http://instagram.com/liikkuva", ""},
		{"https://instagram.com/", ""},
		{"https://instagram.com.evil.example/liikkuva", ""},
		{"https://attacker.example/liikkuva", ""},
		{"javascript:alert(1)", ""},
		{"https://user:pass@instagram.com/liikkuva", ""},
		{"https://www.instagram.com:444/liikkuva", ""},
	} {
		if got := thankYouInstagramURL(tc.input); got != tc.want {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.want, got)
		}
	}
}

func TestThankYouContactEmailFallback(t *testing.T) {
	for _, tc := range []struct { input, want string }{
		{"", "service@liikkuvakorjaamo.fi"},
		{"service@liikkuvakorjaamo.fi", "service@liikkuvakorjaamo.fi"},
		{" support@example.fi ", "support@example.fi"},
		{"attacker@example.fi?subject=spam", "service@liikkuvakorjaamo.fi"},
		{"Name <support@example.fi>", "service@liikkuvakorjaamo.fi"},
	} {
		if got := thankYouContactEmail(tc.input); got != tc.want {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.want, got)
		}
	}
}

func TestThankYouContactCardsFIENAndNoForcedRedirect(t *testing.T) {
	renderer, err := newTemplateRenderer()
	if err != nil { t.Fatal(err) }
	cfg := Config{
		SiteURL: "https://liikkuvakorjaamo.fi",
		DefaultLanguage: "fi",
		SettingsStore: unifiedPriceStore{SiteSettings{
			ProVisitEUR: 50, ProDiagnosticEUR: 65,
			ServiceEmail: "service@liikkuvakorjaamo.fi",
			InstagramURL: "https://www.instagram.com/liikkuva.example/",
			ThankYouInstagramTitleFI: "Seuraa meitä — mukautettu",
			ThankYouInstagramTitleEN: "Follow our repairs — custom",
		}},
	}
	h := liikkuvaPro(cfg, liikkuvaSecurity(cfg, newRouter(cfg, renderer)))
	for _, tc := range []struct {
		path, title, follow string
	}{
		{"/thank-you-fi", "Seuraa meitä — mukautettu", "Avaa Instagram"},
		{"/thank-you-en", "Follow our repairs — custom", "Open Instagram"},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, cfg.SiteURL+tc.path, nil))
		if w.Code != http.StatusOK { t.Fatalf("%s: HTTP %d, %s", tc.path, w.Code, w.Body.String()) }
		body := w.Body.String()
		for _, marker := range []string{tc.title, tc.follow, "mailto:service@liikkuvakorjaamo.fi", "https://www.instagram.com/liikkuva.example/", "thankyou-connect-grid", "thankyou-contacts.css", "rel=\"noopener noreferrer\"", "50", "65"} {
			if !strings.Contains(body, marker) { t.Errorf("%s: missing %q", tc.path, marker) }
		}
		for _, unexpected := range []string{"http-equiv=\"refresh\"", "thankyouCountdown"} {
			if strings.Contains(body, unexpected) { t.Errorf("%s: unexpected redirect control %q", tc.path, unexpected) }
		}
	}
}

func TestThankYouHidesInvalidInstagramAndStillShowsEmail(t *testing.T) {
	renderer, err := newTemplateRenderer()
	if err != nil { t.Fatal(err) }
	cfg := Config{
		SiteURL: "https://liikkuvakorjaamo.fi",
		DefaultLanguage: "fi",
		SettingsStore: unifiedPriceStore{SiteSettings{
			ProVisitEUR: 50, ProDiagnosticEUR: 65,
			InstagramURL: "javascript:alert(1)",
		}},
	}
	h := liikkuvaPro(cfg, liikkuvaSecurity(cfg, newRouter(cfg, renderer)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, cfg.SiteURL+"/thank-you-fi", nil))
	if w.Code != http.StatusOK { t.Fatalf("HTTP %d", w.Code) }
	body := w.Body.String()
	if strings.Contains(body, "javascript:") || strings.Contains(body, "Avaa Instagram") {
		t.Fatal("unsafe or unconfigured social link rendered")
	}
	if !strings.Contains(body, "mailto:service@liikkuvakorjaamo.fi") {
		t.Fatal("canonical fallback contact email not rendered")
	}
}
