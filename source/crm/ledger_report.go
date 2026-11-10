package crm

import (
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"time"
)

// Date filters apply to cash movements, not to current balances. Booked earnings
// are separate from cash; a remittance is never counted as another customer payment.
type LedgerGroup struct {
	Jobs        int              `json:"jobs"`
	Agreed      int64            `json:"agreed_cents"`
	Final       int64            `json:"final_cents"`
	Received    int64            `json:"received_cents"`
	Refunds     int64            `json:"refunds_cents"`
	Outstanding int64            `json:"outstanding_cents"`
	Overdue     int64            `json:"overdue_cents"`
	Costs       int64            `json:"costs_cents"`
	Gross       int64            `json:"gross_cents"`
	Earned      map[string]int64 `json:"earned"`
	Paid        map[string]int64 `json:"paid"`
	Payable     map[string]int64 `json:"payable"`
}

func addLedgerGroup(g *LedgerGroup, l Ledger, t LedgerTotals) {
	g.Jobs++
	g.Agreed += l.Agreed
	g.Final += l.Terms.Final
	g.Received += t.Received
	g.Refunds += t.Refunded
	g.Outstanding += t.Outstanding
	g.Overdue += t.Overdue
	g.Costs += t.Costs
	g.Gross += t.Gross
	if g.Earned == nil {
		g.Earned = map[string]int64{}
		g.Paid = map[string]int64{}
		g.Payable = map[string]int64{}
	}
	for p, n := range t.Earned {
		g.Earned[p] += n
	}
	for p, n := range t.Paid {
		g.Paid[p] += n
	}
	for p, n := range t.Payable {
		g.Payable[p] += n
	}
}

type LedgerReport struct {
	Total     LedgerGroup                 `json:"total"`
	Brands    map[string]*LedgerGroup     `json:"brands"`
	Mechanics map[string]*LedgerGroup     `json:"mechanics"`
	Jobs      []map[string]any            `json:"jobs"`
	Months    map[string]map[string]int64 `json:"months"`
	Days      map[string]map[string]int64 `json:"days"`
}

func (s *Server) ledgerReport(w http.ResponseWriter, r *http.Request, u User) {
	if !u.Manager() {
		fail(w, 403, "Нет доступа")
		return
	}
	if r.Method != "GET" {
		fail(w, 405, "Method not allowed")
		return
	}
	result := LedgerReport{Brands: map[string]*LedgerGroup{}, Mechanics: map[string]*LedgerGroup{}, Jobs: []map[string]any{}, Months: map[string]map[string]int64{}, Days: map[string]map[string]int64{}}
	today := time.Now().In(Helsinki).Format("2006-01-02")
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if (from != "" && !validDate(from)) || (to != "" && !validDate(to)) || (from != "" && to != "" && from > to) {
		fail(w, 400, "Проверьте период")
		return
	}
	// Iterate jobs too: unpaid and historical jobs without a ledger must not disappear.
	it := s.Collection("jobs").Documents(r.Context())
	defer it.Stop()
	for {
		d, e := it.Next()
		if e == iterator.Done {
			break
		}
		if e != nil {
			fail(w, 503, "Ошибка отчёта")
			return
		}
		var j Job
		if d.DataTo(&j) != nil {
			fail(w, 503, "Ошибка заказа")
			return
		}
		if b := r.URL.Query().Get("brand"); b != "" && j.Brand != b {
			continue
		}
		if m := r.URL.Query().Get("mechanic"); m != "" && j.Mechanic != m {
			continue
		}
		l := newLedger(j)
		ld, e := s.Collection("ledgers").Doc(j.ID).Get(r.Context())
		if e == nil {
			if ld.DataTo(&l) != nil {
				fail(w, 503, "Ошибка финансовых данных")
				return
			}
		} else if status.Code(e) != codes.NotFound {
			fail(w, 503, "Ошибка журнала")
			return
		}
		totals := ledgerTotals(l, today)
		addLedgerGroup(&result.Total, l, totals)
		if result.Brands[j.Brand] == nil {
			result.Brands[j.Brand] = &LedgerGroup{}
		}
		if result.Mechanics[j.Mechanic] == nil {
			result.Mechanics[j.Mechanic] = &LedgerGroup{}
		}
		addLedgerGroup(result.Brands[j.Brand], l, totals)
		addLedgerGroup(result.Mechanics[j.Mechanic], l, totals)
		termsConfirmed := false
		for _, entry := range l.Entries {
			if entry.Kind == "terms" {
				termsConfirmed = true
			}
		}
		result.Jobs = append(result.Jobs, map[string]any{"job_id": j.ID, "plate": j.Plate, "brand": j.Brand, "mechanic": j.Mechanic, "agreed_cents": l.Agreed, "terms": l.Terms, "totals": ledgerTotals(l, today), "legacy_report_cents": j.Report.ReceivedCents, "review_required": !termsConfirmed})
		byID := map[string]LedgerEntry{}
		for _, original := range l.Entries {
			entry := original
			sign := int64(1)
			if entry.Kind == "reverse" {
				entry = byID[entry.Reverses]
				sign = -1
			}
			byID[original.ID] = original
			if (from != "" && original.Date < from) || (to != "" && original.Date > to) {
				continue
			}
			kind := entry.Kind
			if kind == "legacy_payment" {
				kind = "payment"
			}
			if kind == "terms" {
				continue
			}
			for _, group := range []struct {
				m map[string]map[string]int64
				k string
			}{{result.Days, original.Date}, {result.Months, original.Date[:7]}} {
				if group.m[group.k] == nil {
					group.m[group.k] = map[string]int64{}
				}
				group.m[group.k][kind] += entry.Cents * sign
			}
		}
	}
	write(w, 200, result)
}
