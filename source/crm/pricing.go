package crm

import (
	"cloud.google.com/go/firestore"
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
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

func DefaultPriceBook() PriceBook {
	return PriceBook{Items: []Tariff{{"liikkuva", "visit", "Выезд", 7000, "EUR", true, false}, {"liikkuva", "diagnostics", "Диагностика", 5000, "EUR", true, true}, {"autohelp", "visit", "Выезд", 5000, "EUR", true, false}, {"autohelp", "diagnostics", "Диагностика", 0, "EUR", false, false}}}
}
func loadPriceBook(ctx context.Context, db *firestore.Client) (PriceBook, error) {
	d, e := db.Collection("crm_settings").Doc("tariffs").Get(ctx)
	if status.Code(e) == codes.NotFound {
		return DefaultPriceBook(), nil
	}
	if e != nil {
		return PriceBook{}, e
	}
	var b PriceBook
	e = d.DataTo(&b)
	return b, e
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
	seen := map[string]bool{}
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
		b.Revision++
		return tx.Set(ref, b)
	})
	if e != nil {
		fail(w, 409, e.Error())
		return
	}
	write(w, 200, b)
}
