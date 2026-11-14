package crm

// Financial records are separate from repair reports. Only explicit owner
// entries affect cash. Historical Report.ReceivedCents is never imported implicitly.
import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"regexp"
	"time"
)

type Share struct {
	Participant string `json:"participant"`
	Fixed       int64  `json:"fixed_cents"`
	BasisPoints int64  `json:"basis_points"`
}
type LedgerTerms struct {
	Final     int64   `json:"final_cents"`
	Completed string  `json:"completed_date"`
	Invoiced  string  `json:"invoice_date"`
	Due       string  `json:"due_date"`
	Shares    []Share `json:"shares"`
}
type LedgerEntry struct {
	ID          string       `json:"id"`
	Kind        string       `json:"kind"`
	Cents       int64        `json:"cents"`
	Date        string       `json:"date"`
	Method      string       `json:"method"`
	Participant string       `json:"participant"`
	Note        string       `json:"note"`
	Reverses    string       `json:"reverses,omitempty"`
	Terms       *LedgerTerms `json:"terms,omitempty"`
	Actor       string       `json:"actor"`
	Recorded    time.Time    `json:"recorded"`
}
type Ledger struct {
	JobID    string        `json:"job_id"`
	Brand    string        `json:"brand"`
	Mechanic string        `json:"mechanic"`
	Revision int64         `json:"revision"`
	Agreed   int64         `json:"agreed_cents"`
	Terms    LedgerTerms   `json:"terms"`
	Entries  []LedgerEntry `json:"entries"`
}
type LedgerRequest struct {
	Revision int64       `json:"revision"`
	Entry    LedgerEntry `json:"entry"`
}
type LedgerTotals struct {
	Received    int64            `json:"received_cents"`
	Refunded    int64            `json:"refunded_cents"`
	Outstanding int64            `json:"outstanding_cents"`
	Credit      int64            `json:"credit_cents"`
	Overdue     int64            `json:"overdue_cents"`
	Costs       int64            `json:"costs_cents"`
	Gross       int64            `json:"gross_cents"`
	CashGross   int64            `json:"cash_gross_cents"`
	Unallocated int64            `json:"unallocated_cents"`
	Earned      map[string]int64 `json:"earned"`
	Paid        map[string]int64 `json:"paid"`
	Payable     map[string]int64 `json:"payable"`
	Held        map[string]int64 `json:"held"`
	Reimburse   map[string]int64 `json:"reimburse"`
}

var ledgerKey = regexp.MustCompile(`^[A-Za-z0-9_-]{8,100}$`)

func validDate(s string) bool {
	t, e := time.Parse("2006-01-02", s)
	return e == nil && t.Year() >= 2000 && t.Year() <= 2100
}
func validTerms(t LedgerTerms) bool {
	if t.Final < 0 || t.Final > 100000000 || len(t.Shares) > 30 {
		return false
	}
	for _, d := range []string{t.Completed, t.Invoiced, t.Due} {
		if d != "" && !validDate(d) {
			return false
		}
	}
	if t.Due != "" && (t.Invoiced == "" || t.Due < t.Invoiced) {
		return false
	}
	var fixed, bps int64
	seen := map[string]bool{}
	for _, s := range t.Shares {
		if s.Participant == "" || len(s.Participant) > 100 || seen[s.Participant] || s.Fixed < 0 || s.Fixed > 100000000 || s.BasisPoints < 0 || s.BasisPoints > 10000 {
			return false
		}
		seen[s.Participant] = true
		fixed += s.Fixed
		bps += s.BasisPoints
		if fixed > t.Final || bps > 10000 {
			return false
		}
	}
	return true
}

// Fixed shares are allocated first, percentages apply to the positive margin
// remaining after costs and fixed allocations. Rounding stays unallocated.
func ledgerTotals(l Ledger, today string) LedgerTotals {
	t := LedgerTotals{Earned: map[string]int64{}, Paid: map[string]int64{}, Payable: map[string]int64{}, Held: map[string]int64{}, Reimburse: map[string]int64{}}
	entries := map[string]LedgerEntry{}
	for _, original := range l.Entries {
		e := original
		sign := int64(1)
		if e.Kind == "reverse" {
			e = entries[e.Reverses]
			sign = -1
		}
		n := e.Cents * sign
		switch e.Kind {
		case "payment", "legacy_payment":
			t.Received += n
			t.Held[e.Participant] += n
		case "refund":
			t.Refunded += n
			t.Held[e.Participant] -= n
		case "expense":
			t.Costs += n
			if e.Participant != "owner" {
				t.Reimburse[e.Participant] += n
			}
		case "remittance":
			t.Held[e.Participant] -= n
			t.Held["owner"] += n
		case "payout":
			t.Paid[e.Participant] += n
		case "reimbursement":
			t.Reimburse[e.Participant] -= n
		}
		entries[original.ID] = original
	}
	net := t.Received - t.Refunded
	t.Outstanding = max(0, l.Terms.Final-net)
	t.Credit = max(0, net-l.Terms.Final)
	if l.Terms.Due != "" && l.Terms.Due < today {
		t.Overdue = t.Outstanding
	}
	t.Gross = l.Terms.Final - t.Costs
	t.CashGross = net - t.Costs
	remainder := max(0, t.Gross)
	for _, s := range l.Terms.Shares {
		t.Earned[s.Participant] = s.Fixed
		remainder -= s.Fixed
	}
	remainder = max(0, remainder)
	var allocated int64
	for _, s := range l.Terms.Shares {
		t.Earned[s.Participant] += remainder * s.BasisPoints / 10000
		allocated += t.Earned[s.Participant]
	}
	t.Unallocated = t.Gross - allocated
	for p, v := range t.Earned {
		t.Payable[p] = v - t.Paid[p]
	}
	for p, v := range t.Paid {
		if _, ok := t.Earned[p]; !ok {
			t.Payable[p] = -v
		}
	}
	return t
}
func newLedger(j Job) Ledger {
	a := int64(0)
	if j.Agreed != nil {
		p := j.Agreed.Proposal
		a = p.VisitCents + p.WorkCents + p.PartsCents
	}
	if j.Agreed == nil {
		switch j.Status {
		case "client_confirmed", "scheduled", "in_progress", "review", "closed":
			p := j.Proposal
			a = p.VisitCents + p.WorkCents + p.PartsCents
		}
	}
	return Ledger{JobID: j.ID, Brand: j.Brand, Mechanic: j.Mechanic, Agreed: a, Terms: LedgerTerms{Final: a, Shares: []Share{}}, Entries: []LedgerEntry{}}
}
func applyLedger(l Ledger, req LedgerRequest, u User, j Job) (Ledger, error) {
	if !u.Manager() {
		return l, ErrForbidden
	}
	e := req.Entry
	if !ledgerKey.MatchString(e.ID) || len(e.Note) > 500 || e.Note == "" || len(e.Participant) > 100 || len(e.Method) > 30 || len(e.Reverses) > 100 {
		return l, errors.New("Укажите ключ операции и основание (до 500 символов)")
	}
	// Same key with different data is an error, never a silent successful write.
	for _, old := range l.Entries {
		if old.ID == e.ID {
			a, b := old, e
			a.Actor = ""
			a.Recorded = time.Time{}
			b.Actor = ""
			b.Recorded = time.Time{}
			aa, _ := json.Marshal(a)
			bb, _ := json.Marshal(b)
			if string(aa) != string(bb) {
				return l, ErrConflict
			}
			return l, nil
		}
	}
	if req.Revision != l.Revision {
		return l, ErrConflict
	}
	if len(l.Entries) >= 500 {
		return l, errors.New("Лимит 500 операций на заказ; требуется архивирование журнала")
	}
	if !validDate(e.Date) || e.Date > time.Now().In(Helsinki).Format("2006-01-02") {
		return l, errors.New("Укажите фактическую дату, не в будущем")
	}
	totals := ledgerTotals(l, e.Date)
	if e.Kind == "terms" {
		if e.Terms == nil || !validTerms(*e.Terms) || e.Cents != 0 || e.Reverses != "" {
			return l, errors.New("Проверьте сумму, даты и доли")
		}
		l.Terms = *e.Terms
	} else if e.Kind == "reverse" {
		if e.Cents != 0 || e.Terms != nil {
			return l, errors.New("Сторно не принимает сумму")
		}
		found := false
		for _, old := range l.Entries {
			if old.Reverses == e.Reverses && old.Kind == "reverse" {
				return l, ErrConflict
			}
			if old.ID == e.Reverses && old.Kind != "reverse" && old.Kind != "terms" {
				found = true
				if e.Date < old.Date {
					return l, errors.New("Сторно раньше исходной операции")
				}
			}
		}
		if !found {
			return l, errors.New("Исходная денежная операция не найдена")
		}
	} else {
		if e.Cents <= 0 || e.Cents > 100000000 || e.Participant == "" || len(e.Participant) > 100 || e.Terms != nil || e.Reverses != "" {
			return l, errors.New("Проверьте сумму и участника")
		}
		switch e.Method {
		case "cash", "card", "transfer", "other":
		default:
			return l, errors.New("Укажите способ платежа")
		}
		switch e.Kind {
		case "payment", "expense", "payout", "reimbursement":
		case "legacy_payment":
			if j.Report.ReceivedCents <= 0 || j.Report.ReceivedCents != e.Cents {
				return l, errors.New("Сумма не совпадает с историческим отчётом")
			}
			for _, old := range l.Entries {
				if old.Kind == "payment" || old.Kind == "legacy_payment" {
					return l, errors.New("В журнале уже есть платёж; выполните ручную сверку")
				}
			}
		case "refund":
			if e.Cents > totals.Received-totals.Refunded {
				return l, errors.New("Возврат превышает полученную сумму")
			}
		case "remittance":
			if e.Participant == "owner" || e.Cents > totals.Held[e.Participant] {
				return l, errors.New("Перевод превышает сумму у сотрудника")
			}
		default:
			return l, errors.New("Неизвестная финансовая операция")
		}
	}
	e.Actor = u.ID
	e.Recorded = time.Now().UTC()
	l.Entries = append(l.Entries, e)
	l.Revision++
	encoded, _ := json.Marshal(l)
	if len(encoded) > 600000 {
		return l, errors.New("Журнал достиг безопасного размера; требуется архивирование")
	}
	t := ledgerTotals(l, e.Date)
	if t.Received < 0 || t.Refunded < 0 || t.Refunded > t.Received || t.Costs < 0 {
		return l, errors.New("Сторно нарушает баланс; сначала отмените зависимые операции")
	}
	for p, n := range t.Held {
		if p != "owner" && n < 0 {
			return l, errors.New("Отрицательный остаток у сотрудника")
		}
	}
	return l, nil
}
func (s *Server) ledger(w http.ResponseWriter, r *http.Request, u User, j Job) {
	if !u.Manager() {
		fail(w, 403, "Финансовый журнал доступен владельцу")
		return
	}
	ref := s.Collection("ledgers").Doc(j.ID)
	l := newLedger(j)
	if r.Method == "GET" {
		d, e := ref.Get(r.Context())
		if e == nil {
			e = d.DataTo(&l)
		}
		if e != nil && status.Code(e) != codes.NotFound {
			fail(w, 503, "Ошибка журнала")
			return
		}
	} else if r.Method == "POST" {
		var req LedgerRequest
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 32000)).Decode(&req) != nil {
			fail(w, 400, "Некорректная операция")
			return
		}
		e := s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
			jd, e := tx.Get(s.Collection("jobs").Doc(j.ID))
			if e != nil {
				return e
			}
			var current Job
			if e = jd.DataTo(&current); e != nil {
				return e
			}
			l = newLedger(current)
			d, e := tx.Get(ref)
			if e == nil {
				if e = d.DataTo(&l); e != nil {
					return e
				}
			} else if status.Code(e) != codes.NotFound {
				return e
			}
			l, e = applyLedger(l, req, u, current)
			if e != nil {
				return e
			}
			return tx.Set(ref, l)
		})
		if e != nil {
			fail(w, 409, e.Error())
			return
		}
	} else {
		fail(w, 405, "Method not allowed")
		return
	}
	write(w, 200, map[string]any{"ledger": l, "totals": ledgerTotals(l, time.Now().In(Helsinki).Format("2006-01-02")), "legacy_report_cents": j.Report.ReceivedCents})
}
