package crm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQuickCompletionFromEveryStatus(t *testing.T) {
	for _, state := range []string{"new", "assigned", "proposed", "client_confirmed", "scheduled", "in_progress", "review", "closed", "cancelled"} {
		j := fixture(t)
		j.Status = state
		j.Mechanic = "m"
		l := newLedger(j)
		q := CompletionRequest{ID: randomID(), Revision: j.Revision, Outcome: "success", Work: "Колесо закреплено"}
		n, got, e := applyCompletion(j, l, q, testOwner, time.Now())
		if e != nil || n.Status != "closed" || got.Revision != 0 {
			t.Fatalf("%s: %v", state, e)
		}
		if _, _, e = applyCompletion(j, l, q, testMechanic, time.Now()); e != ErrForbidden {
			t.Fatal("mechanic bypass", e)
		}
	}
}
func TestPaymentItemsAndOwnerMargin(t *testing.T) {
	j := fixture(t)
	j.Mechanic = "m"
	l := newLedger(j)
	now := time.Now()
	payment := ledgerReq("payment", 15000, "m").Entry
	payment.Note = ""
	payment.Items = []PaymentItem{{"Запчасти", 3000}, {"Замена колёс", 2000}}
	q := CompletionRequest{ID: randomID(), Revision: j.Revision, Outcome: "success", Work: "Выполнено", Terms: &LedgerTerms{Final: 15000, OwnerShare: &OwnerShare{Mode: "percent", Value: 4000, Basis: "margin"}}, Payment: &payment, Expenses: []LedgerEntry{ledgerReq("expense", 3000, "m").Entry}}
	_, l, e := applyCompletion(j, l, q, testOwner, now)
	if e != nil {
		t.Fatal(e)
	}
	totals := ledgerTotals(l, now.Format("2006-01-02"))
	if totals.Received != 15000 || totals.Costs != 3000 || totals.Earned["owner"] != 4800 || totals.Earned["m"] != 7200 {
		t.Fatalf("%+v", totals)
	}
	payment.Items = append(payment.Items, PaymentItem{"Лишнее", 10001})
	q.Payment = &payment
	if _, _, e = applyCompletion(j, newLedger(j), q, testOwner, now); e == nil {
		t.Fatal("overallocated payment")
	}
}
func TestMultiplePaymentsNoDuplicate(t *testing.T) {
	j := fixture(t)
	l := newLedger(j)
	for _, n := range []int64{5000, 7000, 3000} {
		r := ledgerReq("payment", n, "owner")
		r.Entry.Note = ""
		r.Revision = l.Revision
		var e error
		l, e = applyLedger(l, r, testOwner, j)
		if e != nil {
			t.Fatal(e)
		}
		l, e = applyLedger(l, r, testOwner, j)
		if e != nil {
			t.Fatal(e)
		}
	}
	if ledgerTotals(l, "2026-11-17").Received != 15000 {
		t.Fatal("duplicate")
	}
}
func TestTelegramReportWizard(t *testing.T) {
	b := BotSession{Step: "problem", Draft: MechanicReport{Actor: "m", Payment: LedgerEntry{Date: "2026-11-17", Participant: "m"}}}
	done := false
	for _, text := range []string{"Болты", "Заменены болты", "150", "Наличные", "Добавить позицию", "Запчасти", "30", "Добавить позицию", "Колёса", "20", "Далее к расходам", "Добавить расход", "Болты", "30", "Я оплатил", "К итогу", "Отправить отчёт"} {
		var e error
		b, _, _, done, e = advanceReport(b, text)
		if e != nil {
			t.Fatal(text, e)
		}
	}
	if !done || b.Draft.Payment.Cents != 15000 || len(b.Draft.Payment.Items) != 2 || b.Draft.Expenses[0].Participant != "m" {
		t.Fatalf("%+v", b)
	}
	for _, s := range []string{"-1", "1.001", "NaN", "1e3", "1 500", "9223372036854775807"} {
		if _, e := botMoney(s); e == nil {
			t.Fatal(s)
		}
	}
}
func TestEmulatorQuickCompletionAtomicAndIdempotent(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	j := fixture(t)
	j.ID = randomID()
	j.Mechanic = "m"
	j.Status = "assigned"
	if _, e := s.Collection("jobs").Doc(j.ID).Set(ctx, j); e != nil {
		t.Fatal(e)
	}
	payment := ledgerReq("payment", 15000, "m").Entry
	q := CompletionRequest{ID: randomID(), Revision: j.Revision, Outcome: "success", Work: "Сделано", Payment: &payment}
	call := func() int {
		raw, _ := json.Marshal(q)
		w := httptest.NewRecorder()
		s.complete(w, httptest.NewRequest("POST", "/", bytes.NewReader(raw)), testOwner, j)
		return w.Code
	}
	if call() != 200 || call() != 200 {
		t.Fatal("completion retries")
	}
	d, e := s.Collection("ledgers").Doc(j.ID).Get(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var l Ledger
	d.DataTo(&l)
	if len(l.Entries) != 1 {
		t.Fatal("duplicate payment")
	}
	q.Work = "Другой запрос"
	if call() != 409 {
		t.Fatal("key collision")
	}
}

func TestEmulatorTelegramReportsApprovalAndRetries(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	u := testMechanic
	u.TelegramID = "123456"
	putUser(t, s, u)
	j := fixture(t)
	j.ID = randomID()
	j.Mechanic = u.ID
	j.Status = "assigned"
	s.Collection("jobs").Doc(j.ID).Set(ctx, j)
	if e := s.handleBotReport(ctx, u, 100000, "Мои заявки"); e != nil {
		t.Fatal(e)
	}
	d, _ := s.Collection("telegram_sessions").Doc(u.ID).Get(ctx)
	var b BotSession
	d.DataTo(&b)
	choice := ""
	for label, id := range b.Choices {
		if id == j.ID {
			choice = label
		}
	}
	if choice == "" {
		t.Fatal("missing job")
	}
	inputs := []string{choice, "Оставить проблему", "Заменено колесо", "150", "Наличные", "Без детализации", "Нет расходов", "Отправить отчёт"}
	for i, input := range inputs {
		if e := s.handleBotReport(ctx, u, int64(100001+i), input); e != nil {
			t.Fatal(input, e)
		}
	}
	if e := s.handleBotReport(ctx, u, 100008, "Отправить отчёт"); e != nil {
		t.Fatal(e)
	}
	docs, e := s.Collection("mechanic_reports").Documents(ctx).GetAll()
	if e != nil || len(docs) != 1 {
		t.Fatal("duplicate report", e, len(docs))
	}
	if _, e = s.Collection("ledgers").Doc(j.ID).Get(ctx); e == nil {
		t.Fatal("unapproved report counted")
	}
	var report MechanicReport
	docs[0].DataTo(&report)
	q := ReviewReportRequest{ID: report.ID, Decision: "approve"}
	raw, _ := json.Marshal(q)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		s.mechanicReports(w, httptest.NewRequest("POST", "/", bytes.NewReader(raw)), testOwner, j)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	d, e = s.Collection("ledgers").Doc(j.ID).Get(ctx)
	var l Ledger
	if e != nil || d.DataTo(&l) != nil || len(l.Entries) != 1 || ledgerTotals(l, "2026-11-17").Received != 15000 {
		t.Fatal("approval duplicated or lost", e, l)
	}
}
