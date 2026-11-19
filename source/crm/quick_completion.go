package crm

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strconv"
	"time"
)

type CompletionRequest struct {
	ID             string        `json:"id"`
	Revision       int64         `json:"revision"`
	LedgerRevision int64         `json:"ledger_revision"`
	Outcome        string        `json:"outcome"`
	Work           string        `json:"work"`
	Terms          *LedgerTerms  `json:"terms,omitempty"`
	Payment        *LedgerEntry  `json:"payment,omitempty"`
	Expenses       []LedgerEntry `json:"expenses,omitempty"`
}

func applyCompletion(j Job, l Ledger, q CompletionRequest, u User, now time.Time) (Job, Ledger, error) {
	if !ledgerKey.MatchString(q.ID) {
		return j, l, errors.New("Некорректный ключ завершения")
	}
	n, err := Apply(j, Action{Kind: "complete", Revision: q.Revision, Outcome: q.Outcome, Text: q.Work}, u, now)
	if err != nil {
		return j, l, err
	}
	if q.LedgerRevision != l.Revision {
		return j, l, ErrConflict
	}
	if q.Outcome != "success" && (q.Terms != nil || q.Payment != nil || len(q.Expenses) > 0) {
		return j, l, errors.New("Для этого результата финансовые поля должны быть пустыми")
	}
	date := now.In(Helsinki).Format("2006-01-02")
	entries := []LedgerEntry{}
	if q.Terms != nil {
		entries = append(entries, LedgerEntry{ID: q.ID + "-terms", Kind: "terms", Date: date, Note: "Условия при завершении", Terms: q.Terms})
	}
	if q.Payment != nil {
		e := *q.Payment
		e.ID = q.ID + "-payment"
		e.Kind = "payment"
		entries = append(entries, e)
	}
	if len(q.Expenses) > 50 {
		return j, l, errors.New("Слишком много расходов за одну операцию")
	}
	for i, e := range q.Expenses {
		e.ID = q.ID + "-expense-" + stringID(i)
		e.Kind = "expense"
		entries = append(entries, e)
	}
	for _, e := range entries {
		l, err = applyLedger(l, LedgerRequest{Revision: l.Revision, Entry: e}, u, j)
		if err != nil {
			return j, l, err
		}
	}
	return n, l, nil
}
func stringID(i int) string { return strconv.Itoa(i) }

func (s *Server) complete(w http.ResponseWriter, r *http.Request, u User, j Job) {
	if !u.Manager() {
		fail(w, 403, "Только владелец может завершить заявку")
		return
	}
	if r.Method != "POST" {
		fail(w, 405, "Method not allowed")
		return
	}
	var q CompletionRequest
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64000)).Decode(&q) != nil || !ledgerKey.MatchString(q.ID) {
		fail(w, 400, "Некорректное завершение")
		return
	}
	raw, _ := json.Marshal(q)
	fingerprint := digest(string(raw))
	jr := s.Collection("jobs").Doc(j.ID)
	lr := s.Collection("ledgers").Doc(j.ID)
	receipt := jr.Collection("completions").Doc(q.ID)
	err := s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
		old, e := tx.Get(receipt)
		if e == nil {
			var saved struct{ Fingerprint string }
			if old.DataTo(&saved) != nil || saved.Fingerprint != fingerprint {
				return ErrConflict
			}
			return nil
		}
		if status.Code(e) != codes.NotFound {
			return e
		}
		d, e := tx.Get(jr)
		if e != nil {
			return e
		}
		var current Job
		if e = d.DataTo(&current); e != nil {
			return e
		}
		l := newLedger(current)
		d, e = tx.Get(lr)
		if e == nil {
			if e = readStoredLedger(d, &l); e != nil {
				return e
			}
		} else if status.Code(e) != codes.NotFound {
			return e
		}
		var sr *firestore.DocumentRef
		var sched schedule
		if current.Mechanic != "" {
			sr = s.Collection("schedules").Doc(current.Mechanic)
			d, e = tx.Get(sr)
			if e == nil {
				if e = d.DataTo(&sched); e != nil {
					return e
				}
			} else if status.Code(e) != codes.NotFound {
				return e
			}
		}
		now := time.Now().UTC()
		next, l, e := applyCompletion(current, l, q, u, now)
		if e != nil {
			return e
		}
		if sr != nil {
			keep := []booking{}
			for _, b := range sched.Bookings {
				if b.Job != j.ID {
					keep = append(keep, b)
				}
			}
			sched.Bookings = keep
			if e = tx.Set(sr, sched); e != nil {
				return e
			}
		}
		if e = tx.Set(jr, next); e != nil {
			return e
		}
		if e = tx.Set(lr, l); e != nil {
			return e
		}
		if e = tx.Create(jr.Collection("events").Doc(q.ID), Event{Actor: u.ID, Type: "complete", Text: q.Work, At: now, Revision: next.Revision, Snapshot: &next}); e != nil {
			return e
		}
		return tx.Create(receipt, map[string]any{"Fingerprint": fingerprint, "Actor": u.ID, "At": now})
	})
	if err != nil {
		fail(w, 409, err.Error())
		return
	}
	write(w, 200, map[string]bool{"ok": true})
}
