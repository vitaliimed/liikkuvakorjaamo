package crm

import (
	"cloud.google.com/go/firestore"
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"os"
	"strings"
	"time"
)

type Tariff struct {
	Brand    string `json:"brand"`
	Service  string `json:"service"`
	Title    string `json:"title"`
	Cents    int64  `json:"cents"`
	Currency string `json:"currency"`
	Active   bool   `json:"active"`
	Minimum  bool   `json:"minimum"`
}
type PriceBook struct {
	Items    []Tariff `json:"items"`
	Revision int64    `json:"revision"`
}

// Liikkuva visit and diagnostic fees are owned by the landing admin panel.
// Other CRM tariffs (including AutoHelp) remain separately editable.
const defaultLiikkuvaVisitCents int64 = 7500
const defaultLiikkuvaDiagnosticCents int64 = 6000

func DefaultPriceBook() PriceBook {
	return PriceBook{Items: []Tariff{{"liikkuva", "visit", "Выезд", defaultLiikkuvaVisitCents, "EUR", true, false}, {"liikkuva", "diagnostics", "Диагностика", defaultLiikkuvaDiagnosticCents, "EUR", true, true}, {"autohelp", "visit", "Выезд", 5000, "EUR", true, false}, {"autohelp", "diagnostics", "Диагностика", 0, "EUR", false, false}}}
}

// The same Firestore document is read by the public site and the landing admin.
// Missing settings retain the same safe defaults as the Go landing app.
func loadSitePriceCents(ctx context.Context, db *firestore.Client) (int64, int64, error) {
	visit, diagnostics := defaultLiikkuvaVisitCents, defaultLiikkuvaDiagnosticCents
	collection := strings.TrimSpace(os.Getenv("SITE_SETTINGS_COLLECTION"))
	if collection == "" {
		collection = "site_settings"
	}
	d, err := db.Collection(collection).Doc("landing").Get(ctx)
	if status.Code(err) == codes.NotFound {
		return visit, diagnostics, nil
	}
	if err != nil {
		return 0, 0, err
	}
	var settings struct {
		ProVisitEUR      int
		ProDiagnosticEUR int
	}
	if err := d.DataTo(&settings); err != nil {
		return 0, 0, err
	}
	if settings.ProVisitEUR >= 1 && settings.ProVisitEUR <= 2000 {
		visit = int64(settings.ProVisitEUR) * 100
	}
	if settings.ProDiagnosticEUR >= 1 && settings.ProDiagnosticEUR <= 2000 {
		diagnostics = int64(settings.ProDiagnosticEUR) * 100
	}
	return visit, diagnostics, nil
}

func applySiteTariffs(b PriceBook, visit, diagnostic int64) PriceBook {
	foundVisit, foundDiagnostic := false, false
	for i := range b.Items {
		p := &b.Items[i]
		if p.Brand != "liikkuva" {
			continue
		}
		switch p.Service {
		case "visit":
			p.Cents, p.Currency, p.Active, p.Minimum = visit, "EUR", true, false
			foundVisit = true
		case "diagnostics":
			p.Cents, p.Currency, p.Active, p.Minimum = diagnostic, "EUR", true, true
			foundDiagnostic = true
		}
	}
	if !foundVisit {
		b.Items = append(b.Items, Tariff{Brand: "liikkuva", Service: "visit", Title: "Выезд", Cents: visit, Currency: "EUR", Active: true})
	}
	if !foundDiagnostic {
		b.Items = append(b.Items, Tariff{Brand: "liikkuva", Service: "diagnostics", Title: "Диагностика", Cents: diagnostic, Currency: "EUR", Active: true, Minimum: true})
	}
	return b
}

func loadPriceBook(ctx context.Context, db *firestore.Client) (PriceBook, error) {
	d, err := db.Collection("crm_settings").Doc("tariffs").Get(ctx)
	var b PriceBook
	if status.Code(err) == codes.NotFound {
		b = DefaultPriceBook()
	} else if err != nil {
		return PriceBook{}, err
	} else if err := d.DataTo(&b); err != nil {
		return PriceBook{}, err
	}
	visit, diagnostic, err := loadSitePriceCents(ctx, db)
	if err != nil {
		return PriceBook{}, err
	}
	return applySiteTariffs(b, visit, diagnostic), nil
}
func applyPrices(j *Job, b PriceBook) {
	j.PricingSnapshot = nil
	j.Proposal.VisitCents = 0
	j.Proposal.WorkCents = 0
	set := false
	for _, p := range b.Items {
		if p.Brand != j.Brand || !p.Active {
			continue
		}
		j.PricingSnapshot = append(j.PricingSnapshot, p)
		if p.Service == "visit" {
			j.Proposal.VisitCents = p.Cents
		}
		if p.Service == "diagnostics" {
			j.Proposal.WorkCents = p.Cents
			set = true
		}
	}
	j.Proposal.WorkPriceSet = &set
}
func (s *Server) tariffs(w http.ResponseWriter, r *http.Request, u User) {
	if !u.Manager() {
		fail(w, 403, "Только владелец")
		return
	}
	if r.Method == "GET" {
		b, e := loadPriceBook(r.Context(), s.DB)
		if e != nil {
			fail(w, 503, "Ошибка тарифов")
			return
		}
		write(w, 200, b)
		return
	}
	if r.Method != "POST" {
		fail(w, 405, "Method not allowed")
		return
	}
	var b PriceBook
	if e := decode(w, r, &b); e != nil {
		fail(w, 400, e.Error())
		return
	}
	// Reject attempts to override the two landing-controlled fees from CRM.
	visitFee, diagnosticFee, err := loadSitePriceCents(r.Context(), s.DB)
	if err != nil {
		fail(w, 503, "Не удалось прочитать цены сайта")
		return
	}
	for _, p := range b.Items {
		if p.Brand == "liikkuva" && ((p.Service == "visit" && p.Cents != visitFee) || (p.Service == "diagnostics" && p.Cents != diagnosticFee)) {
			fail(w, 409, "Цена выезда и диагностики Liikkuva изменяется только через /admin?tab=general. Обновите тарифы CRM.")
			return
		}
	}
	seen := map[string]bool{}
	if err := validateAutohelpBook(autohelpBook(b)); err != nil {
		fail(w, 400, err.Error())
		return
	}
	if len(b.Items) > 100 {
		fail(w, 400, "Максимум 100 услуг")
		return
	}
	for _, p := range b.Items {
		k := p.Brand + ":" + p.Service
		if (p.Brand != "liikkuva" && p.Brand != "autohelp") || p.Service == "" || p.Title == "" || p.Cents < 0 || p.Cents > 10000000 || p.Currency != "EUR" || seen[k] {
			fail(w, 400, "Некорректный тариф")
			return
		}
		seen[k] = true
	}
	ref := s.Collection("settings").Doc("tariffs")
	e := s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
		d, e := tx.Get(ref)
		var old PriceBook
		if e != nil && status.Code(e) != codes.NotFound {
			return e
		}
		if e == nil {
			if e = d.DataTo(&old); e != nil {
				return e
			}
		}
		if old.Revision != b.Revision {
			return ErrConflict
		}
		b.Revision = old.Revision + 1
		if err := tx.Set(ref, b); err != nil {
			return err
		}
		return tx.Create(ref.Collection("history").NewDoc(), map[string]any{
			"Actor": u.ID, "At": time.Now().UTC(), "Before": old, "After": b,
		})
	})
	if e != nil {
		fail(w, 409, e.Error())
		return
	}
	write(w, 200, b)
}
