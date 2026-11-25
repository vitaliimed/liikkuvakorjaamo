package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
)

// Store previous page content independently from the shared landing settings.
// The write is committed in the same Firestore transaction as the new settings.
func recordServiceChanges(tx *firestore.Transaction, ref *firestore.DocumentRef, current SiteSettings, next *SiteSettings) error {
	prev := make(map[string]ServicePage)
	for _, p := range serviceCatalog(current) {
		prev[p.Path] = p
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range next.ServicePages {
		page := &next.ServicePages[i]
		old, exists := prev[page.Path]
		page.Created, page.Updated = old.Created, old.Updated
		if exists && reflect.DeepEqual(old, *page) {
			continue
		}
		if page.Created == "" {
			page.Created = now
		}
		page.Updated = now
		if exists {
			key := fmt.Sprintf("%x", sha256.Sum256([]byte(page.Path)))
			history := ref.Collection("seo_history").Doc(key).Collection("versions")
			if err := tx.Set(history.Doc(fmt.Sprintf("%020d", next.Revision)), map[string]interface{}{"page": old, "saved_at": now}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *AdminHandler) ServiceHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	if !isAdminAuthenticated(h.cfg, r) {
		w.WriteHeader(403)
		return
	}
	path := r.URL.Query().Get("path")
	if !liikkuvaSEOPath(path) && !extraServicePath.MatchString(path) {
		http.NotFound(w, r)
		return
	}
	db, err := newFirestoreClient(r.Context(), h.cfg)
	if err != nil {
		http.Error(w, "Storage unavailable", 503)
		return
	}
	defer db.Close()
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(path)))
	docs, err := db.Collection(h.cfg.SiteSettingsCollection).Doc(siteSettingsDocID).Collection("seo_history").Doc(key).Collection("versions").OrderBy("saved_at", firestore.Desc).Limit(25).Documents(r.Context()).GetAll()
	if err != nil {
		http.Error(w, "History unavailable", 503)
		return
	}
	type version struct {
		Page    ServicePage `firestore:"page"`
		SavedAt string      `firestore:"saved_at"`
	}
	out := make([]version, 0, len(docs))
	for _, doc := range docs {
		var v version
		if err := doc.DataTo(&v); err != nil {
			http.Error(w, "Invalid history", 500)
			return
		}
		out = append(out, v)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *AdminHandler) PreviewService(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	if !isAdminAuthenticated(h.cfg, r) {
		w.WriteHeader(403)
		return
	}
	path := r.URL.Query().Get("path")
	if !liikkuvaSEOPath(path) && !extraServicePath.MatchString(path) {
		http.NotFound(w, r)
		return
	}
	lang := "fi"
	if strings.HasPrefix(path, "/en/") {
		lang = "en"
	}
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	// Keep the original request path for the authenticated preview distinction.
	NewPageHandler(h.cfg, h.renderer).serveLanding(w, r, lang)
}

// Both existing general settings and the newer page editor can update the same
// three pages. Compare with the previous revision to preserve either workflow.
func syncLegacyServiceFields(current SiteSettings, next *SiteSettings) {
	for i := range next.ServicePages {
		page := &next.ServicePages[i]
		var previous []string
		var legacy []*string
		switch page.Path {
		case "/palvelualue":
			previous = []string{current.ProAreasTitle, current.ProAreasDescription, current.ProAreasHeading, current.ProAreasText}
			legacy = []*string{&next.ProAreasTitle, &next.ProAreasDescription, &next.ProAreasHeading, &next.ProAreasText}
		case "/hinnasto":
			previous = []string{current.ProPricingTitle, current.ProPricingDescription, current.ProPricingHeading, current.ProPricingText}
			legacy = []*string{&next.ProPricingTitle, &next.ProPricingDescription, &next.ProPricingHeading, &next.ProPricingText}
		case "/vikadiagnostiikka":
			previous = []string{current.ProDiagnosticsTitle, current.ProDiagnosticsDescription, current.ProDiagnosticsHeading, current.ProDiagnosticsText}
			legacy = []*string{&next.ProDiagnosticsTitle, &next.ProDiagnosticsDescription, &next.ProDiagnosticsHeading, &next.ProDiagnosticsText}
		default:
			continue
		}
		fallback := servicePages[page.Path]
		defaults := []string{fallback.Title, fallback.Description, fallback.Heading, fallback.Intro}
		values := []*string{&page.Title, &page.Description, &page.Heading, &page.Intro}
		for index, field := range legacy {
			if *field != previous[index] {
				*values[index] = firstNonEmpty(*field, defaults[index])
			}
			if *values[index] != defaults[index] {
				*field = *values[index]
			} else {
				*field = ""
			}
		}
	}
}
