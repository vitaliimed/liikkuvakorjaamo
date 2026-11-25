package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type liikkuvaResponse struct {
	headers http.Header
	Body    bytes.Buffer
	Code    int
}

func (w *liikkuvaResponse) Header() http.Header { return w.headers }
func (w *liikkuvaResponse) WriteHeader(code int) {
	if w.Code == 0 {
		w.Code = code
	}
}
func (w *liikkuvaResponse) Write(b []byte) (int, error) {
	if w.Code == 0 {
		w.Code = 200
	}
	return w.Body.Write(b)
}
func liikkuvaInt(s string, d, min, max int) int {
	n, e := strconv.Atoi(s)
	if e != nil || n < min || n > max {
		return d
	}
	return n
}
func liikkuvaVisitFee(s SiteSettings) int {
	if s.ProVisitEUR < 1 || s.ProVisitEUR > 2000 {
		return visitPriceEUR
	}
	return s.ProVisitEUR
}
func liikkuvaLeadFee(l Lead) int {
	if l.VisitPriceEUR > 0 {
		return l.VisitPriceEUR
	}
	return visitPriceEUR
}
// The quoted starting prices are snapshotted on each lead so follow-up
// emails keep the values presented when the client submitted the request.
func liikkuvaLeadDiagnosticFee(l Lead) int {
	if l.DiagnosticFromEUR > 0 {
		return l.DiagnosticFromEUR
	}
	return 60
}

func liikkuvaProDefaults(s SiteSettings) SiteSettings {
	s.ProVisitEUR = liikkuvaVisitFee(s)
	if s.ProDiagnosticEUR < 1 || s.ProDiagnosticEUR > 2000 {
		s.ProDiagnosticEUR = 60
	}
	if s.ProFontSize < 16 || s.ProFontSize > 22 {
		s.ProFontSize = 18
	}
	// One approved brand accent. Legacy/custom database values cannot leak into public CSS.
	s.ProAccent = "#f97315"
	switch s.ProTheme {
	case "light", "dark", "system", "night":
	default:
		s.ProTheme = "light"
	}
	clock := regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
	if !clock.MatchString(s.ProNightStart) {
		s.ProNightStart = "21:00"
	}
	if !clock.MatchString(s.ProNightEnd) {
		s.ProNightEnd = "07:00"
	}
	return s
}
// Explicit price tokens are the only values expanded in user-authored copy.
// Automatic replacement of literal euro amounts could corrupt an independent
// price, such as a 50-euro battery service or a previously agreed parts quote.
func liikkuvaPricePlaceholders(s string, c SiteSettings) string {
	c = liikkuvaProDefaults(c)
	return strings.NewReplacer(
		"{total}", strconv.Itoa(c.ProVisitEUR+c.ProDiagnosticEUR),
		"{visit}", strconv.Itoa(c.ProVisitEUR),
		"{diagnostic}", strconv.Itoa(c.ProDiagnosticEUR),
	).Replace(s)
}

func liikkuvaPriceText(s string, c SiteSettings) string {
	return liikkuvaPricePlaceholders(s, c)
}

func liikkuvaPro(cfg Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tracking-config.js" {
			s, err := loadSiteSettings(r.Context(), cfg)
			if err != nil {
				http.Error(w, "Temporarily unavailable", 503)
				return
			}
			s = liikkuvaSettings(s)
			b, _ := json.Marshal(map[string]string{"ga4MeasurementId": s.GA4MeasurementID, "googleAdsTagId": s.GoogleAdsTagID, "googleAdsLeadConversionLabel": s.GoogleAdsLeadConversionLabel, "metaPixelId": s.MetaPixelID, "linkedinPartnerId": s.LinkedInPartnerID, "linkedinLeadConversionID": s.LinkedInLeadConversionID, "tiktokPixelId": s.TikTokPixelID})
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			fmt.Fprintf(w, "window.LIIKKUVA_TRACKING=%s;", b)
			return
		}
		public := r.URL.Path == "/" || r.URL.Path == "/en" || liikkuvaSEOPath(r.URL.Path) || extraServicePath.MatchString(r.URL.Path) || strings.HasPrefix(r.URL.Path, "/thank-you") || r.URL.Path == "/fi" || r.URL.Path == "/privacy" || r.URL.Path == "/review"
		if (r.Method != http.MethodGet && r.Method != http.MethodHead) || !public {
			next.ServeHTTP(w, r)
			return
		}
		settings, err := loadSiteSettings(r.Context(), cfg)
		if err != nil {
			http.Error(w, "Service temporarily unavailable", 503)
			return
		}
		settings = liikkuvaProDefaults(settings)
		rec := &liikkuvaResponse{headers: make(http.Header)}
		next.ServeHTTP(rec, r)
		if rec.Code == 0 {
			rec.Code = 200
		}
		if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			w.WriteHeader(rec.Code)
			w.Write(rec.Body.Bytes())
			return
		}
		body := liikkuvaPriceText(rec.Body.String(), settings)
		body = liikkuvaThemeHTML(body, settings)
		css := fmt.Sprintf(`<style>:root{--brand:%s;--blue:%s}html{font-size:%dpx}.btn-orange,.btn-primary{background:%s}</style>`, settings.ProAccent, settings.ProAccent, settings.ProFontSize, settings.ProAccent)
		body = strings.Replace(body, "</head>", css+"</head>", 1)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.Header().Del("Content-Length")
		w.Header().Del("Last-Modified")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(rec.Code)
		if r.Method != http.MethodHead {
			w.Write(bytes.NewBufferString(body).Bytes())
		}
	})
}
