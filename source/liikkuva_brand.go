package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	_ "time/tzdata"
)

const liikkuvaDomain = "liikkuvakorjaamo.fi"
const releaseVersion = "2026.10.30-admin-login"

//go:embed liikkuva-content.json
var liikkuvaContentJSON []byte

func liikkuvaContent(content LandingContent, language string) LandingContent {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(liikkuvaContentJSON, &data); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data[language], &content); err != nil {
		panic(err)
	}
	return content
}

func liikkuvaSubmissionID() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}
func liikkuvaTimestamp(t time.Time) string {
	location, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		panic(err)
	}
	return t.In(location).Format("2006-01-02 15:04:05")
}

func validateLiikkuva(cfg Config) error {
	if os.Getenv("LIIKKUVA_BOOTSTRAP") == "1" {
		return nil
	}
	if cfg.SiteURL != "https://"+liikkuvaDomain {
		return fmt.Errorf("SITE_URL must point to Liikkuva")
	}
	if cfg.FirestoreProjectID == "" || cfg.CloudTasksQueue != "liikkuva-lead-processing" {
		return fmt.Errorf("separate database project and queue required")
	}
	for _, collection := range []string{cfg.FirestoreLeadsCollection, cfg.FirestoreReviewsCollection, cfg.SiteSettingsCollection} {
		if !strings.HasPrefix(collection, "liikkuva_") || strings.Contains(collection, "/") {
			return fmt.Errorf("separate Liikkuva collections required")
		}
	}
	if !strings.HasSuffix(cfg.GCSBucketName, "-liikkuva-files") {
		return fmt.Errorf("separate Liikkuva photo bucket required")
	}
	if len(cfg.InternalTaskToken) < 32 || len(cfg.ReviewTokenSecret) < 32 {
		return fmt.Errorf("separate task and review secrets required")
	}
	if !strings.HasPrefix(cfg.InternalProcessLeadURL, "https://liikkuva-landing-") || !strings.HasSuffix(cfg.InternalProcessLeadURL, ".run.app/internal/process-lead") {
		return fmt.Errorf("worker must target the new Cloud Run service")
	}
	if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" || cfg.GoogleAppsScriptURL == "" || cfg.GoogleAppsScriptToken == "" {
		return fmt.Errorf("Telegram and NEW Google Sheet must be configured before accepting requests")
	}
	return nil
}

func liikkuvaSecurity(cfg Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			target := ""
			if r.URL.Path == "/fi" {
				target = "/"
			}
			for _, path := range []string{"/en", "/hinnasto", "/palvelualue", "/vikadiagnostiikka"} {
				if r.URL.Path == path+"/" {
					target = path
				}
			}
			if target != "" {
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusPermanentRedirect)
				return
			}
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if r.URL.Path == "/hinnasto" || r.URL.Path == "/palvelualue" || r.URL.Path == "/vikadiagnostiikka" {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				w.Header().Set("Allow", "GET, HEAD")
				http.Error(w, "Method not allowed", 405)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeFile(w, r, "static/seo"+r.URL.Path+".html")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/thank-you") || r.URL.Path == "/privacy" || r.URL.Path == "/review" {
			w.Header().Set("X-Robots-Tag", "noindex, follow")
		}
		// The serving frontend returns its own 404 for /healthz in production.
		// Keep that alias for local tools; deployment checks use /api/health.
		if r.URL.Path == "/api/health" || r.URL.Path == "/healthz" {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "site": liikkuvaDomain, "version": releaseVersion, "revision": os.Getenv("K_REVISION")})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/admin") {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", adminReferrerPolicy)
			w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		}
		if strings.HasPrefix(r.URL.Path, "/admin") && r.Method == http.MethodPost {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || !strings.EqualFold(u.Host, r.Host) && !strings.EqualFold(origin, strings.TrimRight(cfg.SiteURL, "/")) {
					http.Error(w, "Forbidden", 403)
					return
				}
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(w, "Forbidden", 403)
				return
			}
		}
		if r.URL.Path == "/review" && r.Method == http.MethodGet && r.URL.Query().Get("lead") == "" && r.URL.Query().Get("token") == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("X-Robots-Tag", "noindex, nofollow")
			title, text, button := "Feedback on our work", "After the service, you will receive a personal review link by email. Open that link to leave a star rating and a review.", "Homepage"
			if r.URL.Query().Get("lang") != "en" {
				title = "Palaute tehdystä työstä"
				text = "Saat henkilökohtaisen arvostelulinkin sähköpostitse palvelun jälkeen. Avaamalla linkin voit antaa tähtiarvion ja kirjoittaa arvostelun."
				button = "Etusivu"
			}
			fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><link rel="stylesheet" href="/static/css/style.css"><title>Liikkuva — review</title></head><body><header class="subpage-header"><div class="container"><a class="brand" href="/"><img src="/static/img/logo_liikkuva.svg" alt="" width="44" height="44"><span>Liikkuva<span class="brand-small">Auto Korjaamo</span></span></a></div></header><main class="subpage-main"><div class="container"><article class="subpage-card"><h1>%s</h1><p>%s</p><a class="btn btn-orange" href="/">%s</a></article></div></main></body></html>`, title, text, button)
			return
		}
		if os.Getenv("LIIKKUVA_BOOTSTRAP") == "1" && r.Method != "GET" && !strings.HasPrefix(r.URL.Path, "/admin") {
			http.Error(w, "Setup in progress", 503)
			return
		}
		if r.URL.Path == "/admin/photo" {
			liikkuvaAdminPhoto(cfg, w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

//go:embed seo-defaults.json
var liikkuvaSEOJSON []byte

func liikkuvaSettings(settings SiteSettings) SiteSettings {
	var defaults SiteSettings
	if err := json.Unmarshal(liikkuvaSEOJSON, &defaults); err != nil {
		panic(err)
	}
	if strings.TrimSpace(settings.LandingSEOTitleFI) == "" {
		settings.LandingSEOTitleFI = defaults.LandingSEOTitleFI
	}
	if strings.TrimSpace(settings.LandingSEOTitleEN) == "" {
		settings.LandingSEOTitleEN = defaults.LandingSEOTitleEN
	}
	if strings.TrimSpace(settings.LandingSEODescriptionFI) == "" {
		settings.LandingSEODescriptionFI = defaults.LandingSEODescriptionFI
	}
	if strings.TrimSpace(settings.LandingSEODescriptionEN) == "" {
		settings.LandingSEODescriptionEN = defaults.LandingSEODescriptionEN
	}
	if strings.TrimSpace(settings.GA4MeasurementID) == "" {
		settings.GA4MeasurementID = defaults.GA4MeasurementID
	}
	if strings.TrimSpace(settings.LandingRobotsMeta) == "" {
		settings.LandingRobotsMeta = defaults.LandingRobotsMeta
	}
	if strings.TrimSpace(settings.ThankYouRobotsMeta) == "" {
		settings.ThankYouRobotsMeta = defaults.ThankYouRobotsMeta
	}
	if strings.TrimSpace(settings.PrivacyRobotsMeta) == "" {
		settings.PrivacyRobotsMeta = defaults.PrivacyRobotsMeta
	}
	if strings.TrimSpace(settings.PrivacySEOTitleFI) == "" {
		settings.PrivacySEOTitleFI = defaults.PrivacySEOTitleFI
	}
	if strings.TrimSpace(settings.PrivacySEOTitleEN) == "" {
		settings.PrivacySEOTitleEN = defaults.PrivacySEOTitleEN
	}
	if strings.TrimSpace(settings.ThankYouSEOTitleFI) == "" {
		settings.ThankYouSEOTitleFI = defaults.ThankYouSEOTitleFI
	}
	if strings.TrimSpace(settings.ThankYouSEOTitleEN) == "" {
		settings.ThankYouSEOTitleEN = defaults.ThankYouSEOTitleEN
	}
	if strings.TrimSpace(settings.RobotsTXT) == "" {
		settings.RobotsTXT = defaults.RobotsTXT
	}
	return settings
}
