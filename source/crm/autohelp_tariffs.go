package crm

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var tariffServiceCode = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// Import uses the prices captured at submission, not the prices at sync time.
func applyAutohelpLeadPrices(j *Job, snapshot []Tariff, visitEUR int64) {
	if len(snapshot) > 0 && validateAutohelpBook(PriceBook{Items: snapshot}) == nil {
		applyPrices(j, PriceBook{Items: snapshot})
		return
	}
	// Old AutoHelp versions did not always persist prices. Do not invent a
	// historical agreement by applying today's tariff to an older submission.
	applyPrices(j, PriceBook{})
	if visitEUR > 0 && visitEUR <= 100000 {
		applyPrices(j, PriceBook{Items: []Tariff{{Brand: "autohelp", Service: "visit", Title: "Выезд", Cents: visitEUR * 100, Currency: "EUR", Active: true}}})
	} else {
		j.Description += "\nТариф на дату обращения AutoHelp не сохранён. Согласуйте стоимость перед отправкой предложения."
	}
}

// A dedicated credential grants access to AutoHelp prices only, never jobs,
// customers, finances, Liikkuva prices or the notification dispatcher.
func (s *Server) autohelpTariffs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	expected := os.Getenv("CRM_AUTOHELP_TARIFF_TOKEN")
	given := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if len(expected) < 32 || subtle.ConstantTimeCompare([]byte(expected), []byte(given)) != 1 {
		fail(w, 403, "Нет доступа к тарифам")
		return
	}
	if r.Method == http.MethodGet {
		book, err := loadPriceBook(r.Context(), s.DB)
		if err != nil {
			fail(w, 503, "Тарифы временно недоступны")
			return
		}
		write(w, 200, autohelpBook(book))
		return
	}
	if r.Method != http.MethodPost {
		fail(w, 405, "Method not allowed")
		return
	}
	var next PriceBook
	if err := decode(w, r, &next); err != nil {
		fail(w, 400, err.Error())
		return
	}
	if err := validateAutohelpBook(next); err != nil {
		fail(w, 400, err.Error())
		return
	}
	result, err := s.saveAutohelpBook(r.Context(), next)
	if errors.Is(err, ErrConflict) {
		fail(w, 409, "Тарифы уже изменены. Обновите страницу и повторите изменение.")
		return
	}
	if err != nil {
		fail(w, 503, "Не удалось сохранить тарифы")
		return
	}
	write(w, 200, result)
}

func autohelpBook(b PriceBook) PriceBook {
	out := PriceBook{Revision: b.Revision, Items: []Tariff{}}
	for _, item := range b.Items {
		if item.Brand == "autohelp" {
			out.Items = append(out.Items, item)
		}
	}
	return out
}

func validateAutohelpBook(b PriceBook) error {
	if b.Revision < 0 || len(b.Items) < 2 || len(b.Items) > 100 {
		return errors.New("Укажите тарифы AutoHelp: выезд и диагностику")
	}
	seen := map[string]bool{}
	for _, p := range b.Items {
		if p.Brand != "autohelp" || !tariffServiceCode.MatchString(p.Service) || strings.TrimSpace(p.Title) == "" || len(p.Title) > 200 || p.Cents < 0 || p.Cents > 10000000 || p.Currency != "EUR" || seen[p.Service] {
			return errors.New("Некорректный тариф AutoHelp")
		}
		seen[p.Service] = true
		if p.Service == "visit" && !p.Active {
			return errors.New("Тариф выезда должен быть активен")
		}
	}
	if !seen["visit"] || !seen["diagnostics"] {
		return errors.New("Сохраните отдельные тарифы выезда и диагностики")
	}
	return nil
}

func mergeAutohelpBook(old, next PriceBook) (PriceBook, error) {
	if err := validateAutohelpBook(next); err != nil {
		return PriceBook{}, err
	}
	if old.Revision != next.Revision {
		// Lost HTTP response: repeating the exact already-applied prices is safe.
		if reflect.DeepEqual(autohelpBook(old).Items, next.Items) {
			return old, nil
		}
		return PriceBook{}, ErrConflict
	}
	merged := PriceBook{Revision: old.Revision + 1}
	for _, item := range old.Items {
		if item.Brand != "autohelp" {
			merged.Items = append(merged.Items, item)
		}
	}
	merged.Items = append(merged.Items, next.Items...)
	return merged, nil
}

func (s *Server) saveAutohelpBook(ctx context.Context, next PriceBook) (PriceBook, error) {
	ref := s.Collection("settings").Doc("tariffs")
	var result PriceBook
	err := s.DB.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		old := DefaultPriceBook()
		d, err := tx.Get(ref)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if err == nil {
			if err = d.DataTo(&old); err != nil {
				return err
			}
		}
		merged, err := mergeAutohelpBook(old, next)
		if err != nil {
			return err
		}
		result = autohelpBook(merged)
		if merged.Revision == old.Revision {
			return nil
		}
		if err = tx.Set(ref, merged); err != nil {
			return err
		}
		return tx.Create(ref.Collection("history").NewDoc(), map[string]any{
			"Actor": "autohelp-admin", "At": time.Now().UTC(), "Before": autohelpBook(old), "After": result,
		})
	})
	return result, err
}
