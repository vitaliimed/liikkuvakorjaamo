package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

type ReviewHandler struct {
	cfg      Config
	renderer *TemplateRenderer
}

func NewReviewHandler(cfg Config, renderer *TemplateRenderer) *ReviewHandler {
	return &ReviewHandler{cfg: cfg, renderer: renderer}
}

func (h *ReviewHandler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.show(w, r)
	case http.MethodPost:
		h.submit(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ReviewHandler) show(w http.ResponseWriter, r *http.Request) {
	lang := sanitizeLanguage(r.URL.Query().Get("lang"), h.cfg.DefaultLanguage)
	if r.URL.Query().Get("submitted") == "1" {
		h.render(w, r, ReviewPageData{Lang: lang, Submitted: true}, http.StatusOK)
		return
	}

	leadID := strings.TrimSpace(r.URL.Query().Get("lead"))
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	lead, ok := h.authorizedLead(w, r, leadID, token)
	if !ok {
		return
	}
	lang = sanitizeLanguage(r.URL.Query().Get("lang"), lead.Language)

	exists, err := reviewExists(r.Context(), h.cfg, leadID)
	if err != nil {
		log.Printf("review existence check failed: %v", err)
		http.Error(w, "failed to load review", http.StatusInternalServerError)
		return
	}

	h.render(w, r, ReviewPageData{
		Lang:             lang,
		LeadID:           leadID,
		Token:            token,
		Name:             lead.Name,
		NameMode:         "real",
		Rating:           5,
		AlreadySubmitted: exists,
	}, http.StatusOK)
}

func (h *ReviewHandler) submit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	leadID := strings.TrimSpace(r.FormValue("lead_id"))
	token := strings.TrimSpace(r.FormValue("token"))
	lead, ok := h.authorizedLead(w, r, leadID, token)
	if !ok {
		return
	}

	lang := sanitizeLanguage(r.FormValue("language"), lead.Language)
	name := strings.TrimSpace(r.FormValue("name"))
	nameMode := normalizeReviewNameMode(r.FormValue("name_mode"))
	alias := strings.TrimSpace(r.FormValue("alias"))
	displayName := name
	if nameMode == "alias" {
		displayName = alias
	}
	message := strings.TrimSpace(r.FormValue("message"))
	rating, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("rating")))
	consent := strings.TrimSpace(r.FormValue("publish_consent")) == "1"
	errorsByField := validateReview(displayName, message, rating, consent, lang)
	if nameMode == "alias" {
		if nameError := errorsByField["name"]; nameError != "" {
			errorsByField["alias"] = nameError
			delete(errorsByField, "name")
		}
	}

	data := ReviewPageData{
		Lang:           lang,
		LeadID:         leadID,
		Token:          token,
		Name:           name,
		NameMode:       nameMode,
		Alias:          alias,
		Rating:         rating,
		Message:        message,
		PublishConsent: consent,
		Errors:         errorsByField,
	}
	if len(errorsByField) > 0 {
		h.render(w, r, data, http.StatusBadRequest)
		return
	}

	err := createReviewRecord(r.Context(), h.cfg, ReviewRecord{
		LeadID:         leadID,
		Name:           displayName,
		Rating:         rating,
		Message:        message,
		Language:       lang,
		PublishConsent: consent,
	})
	if errors.Is(err, errReviewAlreadySubmitted) {
		data.AlreadySubmitted = true
		h.render(w, r, data, http.StatusConflict)
		return
	}
	if err != nil {
		log.Printf("save review failed: %v", err)
		http.Error(w, "failed to save review", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/review?submitted=1&lang="+url.QueryEscape(lang), http.StatusSeeOther)
}

func normalizeReviewNameMode(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "alias") {
		return "alias"
	}
	return "real"
}

func (h *ReviewHandler) authorizedLead(w http.ResponseWriter, r *http.Request, leadID, token string) (Lead, bool) {
	if leadID == "" || token == "" {
		http.Error(w, "invalid review link", http.StatusBadRequest)
		return Lead{}, false
	}

	record, err := getLeadRecord(r.Context(), h.cfg, leadID)
	if err != nil {
		http.Error(w, "review link not found", http.StatusNotFound)
		return Lead{}, false
	}
	if !verifyReviewToken(h.cfg, leadID, record.Lead.Email, token) {
		http.Error(w, "invalid review link", http.StatusForbidden)
		return Lead{}, false
	}
	return record.Lead, true
}

func (h *ReviewHandler) render(w http.ResponseWriter, r *http.Request, data ReviewPageData, statusCode int) {
	settings, err := loadSiteSettings(r.Context(), h.cfg)
	if err != nil {
		log.Printf("loadSiteSettings for review warning: %v", err)
	}
	data.Lang = sanitizeLanguage(data.Lang, h.cfg.DefaultLanguage)
	data.CanonicalURL = strings.TrimRight(h.cfg.SiteURL, "/") + "/review"
	data.LogoURL = assetOrFallback(settings.LogoURL, "/static/img/logo_liikkuva.svg?v=20260327b")
	data.FaviconSVGURL, data.Favicon32URL, data.Favicon16URL, data.FaviconICOURL, data.AppleTouchIconURL = faviconURLs(settings)
	if data.Lang == "fi" {
		data.PageTitle = "Jätä arvostelu | Liikkuva"
		data.MetaDescription = "Jätä palautetta Liikkuvain liikkuvasta autokorjaamopalvelusta."
	} else {
		data.PageTitle = "Leave a review | Liikkuva"
		data.MetaDescription = "Share feedback about Liikkuva mobile car repair service."
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := h.renderer.renderReview(w, data); err != nil {
		log.Printf("render review failed: %v", err)
	}
}

func validateReview(name, message string, rating int, consent bool, lang string) map[string]string {
	errorsByField := map[string]string{}
	if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 80 {
		if lang == "fi" {
			errorsByField["name"] = "Kirjoita nimi (2–80 merkkiä)."
		} else {
			errorsByField["name"] = "Enter a name (2–80 characters)."
		}
	}
	if rating < 1 || rating > 5 {
		if lang == "fi" {
			errorsByField["rating"] = "Valitse 1–5 tähteä."
		} else {
			errorsByField["rating"] = "Select 1–5 stars."
		}
	}
	if utf8.RuneCountInString(message) < 10 || utf8.RuneCountInString(message) > 1200 {
		if lang == "fi" {
			errorsByField["message"] = "Kirjoita 10–1200 merkin arvostelu."
		} else {
			errorsByField["message"] = "Write a review of 10–1200 characters."
		}
	}
	if !consent {
		if lang == "fi" {
			errorsByField["publish_consent"] = "Julkaisulupa on vahvistettava."
		} else {
			errorsByField["publish_consent"] = "You must confirm permission to publish."
		}
	}
	return errorsByField
}

func reviewTokenSecret(cfg Config) string {
	if value := strings.TrimSpace(cfg.ReviewTokenSecret); value != "" {
		return value
	}
	if value := strings.TrimSpace(cfg.InternalTaskToken); value != "" {
		return value
	}
	return strings.TrimSpace(cfg.AdminPassword)
}

func buildReviewToken(cfg Config, leadID, email string) (string, error) {
	secret := reviewTokenSecret(cfg)
	if secret == "" {
		return "", fmt.Errorf("review token secret is not configured")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strings.TrimSpace(leadID) + "\n" + strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func verifyReviewToken(cfg Config, leadID, email, token string) bool {
	expected, err := buildReviewToken(cfg, leadID, email)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(token)))
}

func buildReviewURL(cfg Config, leadID string, lead Lead) (string, error) {
	return buildReviewURLWithBase(cfg, cfg.SiteURL, leadID, lead)
}

func buildReviewURLWithBase(cfg Config, baseURL, leadID string, lead Lead) (string, error) {
	token, err := buildReviewToken(cfg, leadID, lead.Email)
	if err != nil {
		return "", err
	}
	values := url.Values{}
	values.Set("lead", leadID)
	values.Set("token", token)
	values.Set("lang", sanitizeLanguage(lead.Language, "fi"))
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(cfg.SiteURL), "/")
	}
	return baseURL + "/review?" + values.Encode(), nil
}

func publicBaseURLForRequest(cfg Config, r *http.Request) string {
	fallback := strings.TrimRight(strings.TrimSpace(cfg.SiteURL), "/")
	if r == nil || strings.TrimSpace(r.Host) == "" {
		return fallback
	}

	host := strings.TrimSpace(r.Host)
	if !allowedPublicHost(fallback, host) {
		return fallback
	}

	proto := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]))
	if proto != "http" && proto != "https" {
		if r.TLS != nil || !strings.HasPrefix(host, "localhost") {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	return proto + "://" + host
}

func allowedPublicHost(configuredBaseURL, requestHost string) bool {
	hostname := strings.ToLower(strings.TrimSpace(strings.Split(requestHost, ":")[0]))
	if hostname == "localhost" || strings.HasSuffix(hostname, ".run.app") || hostname == "liikkuvakorjaamo.fi" || strings.HasSuffix(hostname, ".liikkuvakorjaamo.fi") {
		return true
	}
	parsed, err := url.Parse(configuredBaseURL)
	return err == nil && strings.EqualFold(parsed.Host, requestHost)
}
