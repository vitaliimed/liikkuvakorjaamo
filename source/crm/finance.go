package crm

import (
	"google.golang.org/api/iterator"
	"net/http"
)

type FinanceTotals struct {
	Jobs          int   `json:"jobs"`
	Closed        int   `json:"closed"`
	AgreedCents   int64 `json:"agreed_cents"`
	ReportedCents int64 `json:"reported_cents"`
	AcceptedCents int64 `json:"accepted_cents"`
}
type FinanceSummary struct {
	Total     FinanceTotals             `json:"total"`
	Brands    map[string]*FinanceTotals `json:"brands"`
	Mechanics map[string]*FinanceTotals `json:"mechanics"`
	Payments  map[string]int64          `json:"payments"`
}

func addFinance(t *FinanceTotals, j Job) {
	t.Jobs++
	switch j.Status {
	case "client_confirmed", "scheduled", "in_progress", "review", "closed":
		t.AgreedCents += j.Proposal.VisitCents + j.Proposal.WorkCents + j.Proposal.PartsCents
	}
	if j.Report.Work != "" {
		t.ReportedCents += j.Report.ReceivedCents
	}
	if j.Status == "closed" {
		t.Closed++
		t.AcceptedCents += j.Report.ReceivedCents
	}
}
func (s *Server) finance(w http.ResponseWriter, r *http.Request, u User) {
	if !u.Manager() {
		fail(w, 403, "Финансы доступны только владельцу")
		return
	}
	if r.Method != "GET" {
		fail(w, 405, "Method not allowed")
		return
	}
	result := FinanceSummary{Brands: map[string]*FinanceTotals{}, Mechanics: map[string]*FinanceTotals{}, Payments: map[string]int64{}}
	// Read all records, independent of the paginated UI. Never return partial totals.
	it := s.Collection("jobs").Documents(r.Context())
	defer it.Stop()
	for {
		d, e := it.Next()
		if e == iterator.Done {
			break
		}
		if e != nil {
			fail(w, 503, "Не удалось рассчитать полную сводку; повторите запрос")
			return
		}
		var j Job
		if d.DataTo(&j) != nil {
			fail(w, 503, "Некорректные данные финансового отчёта")
			return
		}
		addFinance(&result.Total, j)
		if result.Brands[j.Brand] == nil {
			result.Brands[j.Brand] = &FinanceTotals{}
		}
		addFinance(result.Brands[j.Brand], j)
		if result.Mechanics[j.Mechanic] == nil {
			result.Mechanics[j.Mechanic] = &FinanceTotals{}
		}
		addFinance(result.Mechanics[j.Mechanic], j)
		if j.Status == "closed" {
			result.Payments[j.Report.Payment] += j.Report.ReceivedCents
		}
	}
	write(w, 200, result)
}
