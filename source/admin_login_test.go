package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

func TestAdminFormPolicyAndRejectedLoginCanBeRetried(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	for _, path := range []string{"/admin", "/admin/login", "/admin/recovery", "/admin/reset-password?token=invalid"} {
		w := f.request("GET", path, nil, nil)
		if w.Header().Get("Referrer-Policy") != "strict-origin" {
			t.Errorf("%s suppresses native browser form Origin", path)
		}
	}
	for _, origin := range []string{"null", "https://attacker.example"} {
		r := httptest.NewRequest("POST", f.cfg.SiteURL+"/admin/login", strings.NewReader("password="+fixtureAdminPassword))
		r.Header.Set("Origin", origin)
		r.Header.Set("Sec-Fetch-Site", "same-origin")
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), `action="/admin/login"`) || w.Header().Get("Referrer-Policy") != "strict-origin" {
			t.Fatalf("rejected login cannot be retried: %d", w.Code)
		}
	}
	state, _ := f.store.Load(context.Background())
	if len(state.Sessions) != 0 || len(state.Limits) != 0 {
		t.Fatal("untrusted login reached the credential store")
	}
	cookie := f.login(t, fixtureAdminPassword)
	for _, path := range []string{"/admin", "/admin/access"} {
		w := f.request("GET", path, nil, cookie)
		if w.Header().Get("Referrer-Policy") != "strict-origin" {
			t.Errorf("authenticated form %s suppresses browser Origin", path)
		}
	}
}

// Opt-in real-browser check. Credentials, settings and SMTP are entirely local
// test doubles. The proxy reproduces Hosting's changed Host and cookie filter.
func TestAdminBrowserLoginFlow(t *testing.T) {
	if os.Getenv("LIIKKUVA_BROWSER_TEST") != "1" {
		t.Skip("set LIIKKUVA_BROWSER_TEST=1; requires Node, Playwright and Chromium")
	}
	f := newAccessFixture(t, fixtureAdminPassword)
	settings := &memorySettingsStore{state: SiteSettings{BusinessName: "Liikkuva", ServiceEmail: "service@example.test"}}
	f.cfg.SettingsStore = settings
	f.cfg.AssetStore = &memorySiteAssets{}
	var mu sync.Mutex
	var origins, referers []string
	ready := make(chan struct{})
	proxy := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-ready
		if r.URL.Path == "/__test/legacy-login" {
			rec := httptest.NewRecorder()
			renderAccess(rec, http.StatusOK, accessPageData{Mode: "login"})
			for key, values := range rec.Header() {
				w.Header()[key] = values
			}
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Write(rec.Body.Bytes())
			return
		}
		if r.URL.Path == "/__test/recovery-token" {
			f.mailMu.Lock()
			defer f.mailMu.Unlock()
			token := ""
			if len(f.mails) > 0 {
				match := regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`).FindStringSubmatch(f.mails[len(f.mails)-1].Body)
				if len(match) == 2 {
					token = match[1]
				}
			}
			json.NewEncoder(w).Encode(map[string]string{"token": token})
			return
		}
		if r.Method == http.MethodPost {
			mu.Lock()
			origins = append(origins, r.Header.Get("Origin"))
			referers = append(referers, r.Referer())
			mu.Unlock()
		}
		r = r.Clone(r.Context())
		r.Host = "liikkuva-preview.internal.run.app"
		cookie, _ := r.Cookie("__session")
		r.Header.Del("Cookie")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		f.handler.ServeHTTP(w, r)
	}))
	defer proxy.Close()
	f.cfg.SiteURL = proxy.URL
	renderer, err := newTemplateRenderer()
	if err != nil {
		t.Fatal(err)
	}
	f.handler = newServer(f.cfg, renderer).Handler
	close(ready)
	command := exec.Command("node", filepath.Join("..", "tests", "admin-login.cjs"), proxy.URL)
	output, err := command.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatalf("native browser login flow: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(origins) < 8 || origins[0] != "null" {
		t.Fatal("legacy no-referrer regression was not reproduced")
	}
	for i := 1; i < len(origins); i++ {
		if origins[i] != proxy.URL || referers[i] != proxy.URL+"/" {
			t.Fatalf("request %d lost its origin or leaked its URL path", i)
		}
	}
	saved, _ := settings.Load(context.Background())
	if saved.ProVisitEUR != 92 {
		t.Fatal("browser did not save the actual settings form")
	}
}
