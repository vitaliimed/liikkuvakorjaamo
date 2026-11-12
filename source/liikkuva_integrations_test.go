package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

type liikkuvaTelegramTransport func(*http.Request) (*http.Response, error)

func (f liikkuvaTelegramTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSharedBotBrandsBothMessageAndPhotoWithoutRealDelivery(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "synthetic-photo-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	file.WriteString("synthetic attachment")
	file.Close()
	var text, caption string
	transport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = transport })
	http.DefaultTransport = liikkuvaTelegramTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.telegram.org" {
			t.Fatal("unexpected network destination")
		}
		if strings.HasSuffix(r.URL.Path, "/sendMessage") {
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			text = values.Get("text")
			if values.Get("chat_id") != "-100123" {
				t.Fatal("wrong shared chat")
			}
		} else if strings.HasSuffix(r.URL.Path, "/sendDocument") {
			reader, err := r.MultipartReader()
			if err != nil {
				t.Fatal(err)
			}
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				data, _ := io.ReadAll(part)
				if part.FormName() == "caption" {
					caption = string(data)
				}
			}
		} else {
			t.Fatal("unexpected Telegram method")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})
	lead := Lead{Name: "<Client>", Phone: "+358401234567", Language: "fi", Car: "TEST-123", VisitPriceEUR: 85, PhotoPaths: []string{file.Name()}}
	if err := sendLeadToTelegram(Config{TelegramBotToken: "synthetic-token", TelegramChatID: "-100123"}, lead); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{text, caption} {
		if !strings.Contains(body, "#Liikkuva") || !strings.Contains(body, "liikkuvakorjaamo.fi") {
			t.Fatal("shared bot notification lost its source brand")
		}
	}
	if !strings.Contains(text, "85 €") || strings.Contains(text, "<Client>") || !strings.Contains(text, "&lt;Client&gt;") {
		t.Fatal("wrong price snapshot or unsafe customer text")
	}
	if !strings.Contains(caption, "TEST-123") {
		t.Fatal("photo cannot be matched to the vehicle")
	}
}

func TestThemeSettingsSaveAndRenderWithoutAttributeInjection(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	settings := &memorySettingsStore{}
	f.cfg.SettingsStore = settings
	renderer, _ := newTemplateRenderer()
	f.handler = newServer(f.cfg, renderer).Handler
	cookie := f.login(t, fixtureAdminPassword)
	values := url.Values{"access_csrf": {fixtureCSRF(cookie)}, "settings_revision": {"0"}, "active_tab": {"general"}, "phone": {"+358401234567"}, "ProTheme": {"night"}, "ProNightStart": {"22:30"}, "ProNightEnd": {"06:45"}}
	if w := f.request("POST", "/admin/save", values, cookie); w.Code != 303 {
		t.Fatalf("theme save failed: %d", w.Code)
	}
	saved, _ := settings.Load(context.Background())
	if saved.ProTheme != "night" || saved.ProNightStart != "22:30" || saved.ProNightEnd != "06:45" {
		t.Fatal("theme settings were not persisted")
	}
	w := f.request("GET", "/", nil, nil)
	for _, marker := range []string{`data-theme-policy="night"`, `data-night-start="22:30"`, `data-night-end="06:45"`, `/static/js/theme.js`, `data-theme-toggle`} {
		if !strings.Contains(w.Body.String(), marker) {
			t.Fatal("public theme missing: " + marker)
		}
	}
	unsafe := liikkuvaThemeHTML(`<html><head></head><body></body></html>`, SiteSettings{ProTheme: `"><script>bad</script>`, ProNightStart: `x" onload="bad`, ProNightEnd: "99:99"})
	if strings.Contains(unsafe, "bad") || !strings.Contains(unsafe, `data-theme-policy="light"`) || !strings.Contains(unsafe, `data-night-start="21:00"`) {
		t.Fatal("theme attributes were not normalized")
	}
}

func TestTelegramErrorsCannotExposeSharedBotCredentialsOrClaimSuccess(t *testing.T) {
	for _, sample := range []struct {
		status int
		body   string
	}{
		{401, `{"ok":false,"description":"secret-token-in-provider-response"}`},
		{200, `{"ok":false,"error_code":403}`},
		{200, `invalid-json`},
	} {
		err := checkTelegramResponse(&http.Response{StatusCode: sample.status, Body: io.NopCloser(strings.NewReader(sample.body))})
		if err == nil || strings.Contains(err.Error(), "secret-token") {
			t.Fatal("unsafe or unverified Telegram result")
		}
	}
}
