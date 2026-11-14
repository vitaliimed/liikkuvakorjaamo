package crm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func ledgerReq(kind string, n int64, p string) LedgerRequest {
	return LedgerRequest{Entry: LedgerEntry{ID: randomID(), Kind: kind, Cents: n, Participant: p, Method: "cash", Note: "verified receipt", Date: time.Now().In(Helsinki).Format("2006-01-02")}}
}
func TestLedgerMoneyAndDistribution(t *testing.T) {
	j := fixture(t)
	l := newLedger(j)
	l.Terms = LedgerTerms{Final: 20000, Due: "2020-01-02", Shares: []Share{{Participant: "owner", BasisPoints: 4000}, {Participant: "m", Fixed: 2000, BasisPoints: 6000}}}
	for _, r := range []LedgerRequest{ledgerReq("payment", 12000, "m"), ledgerReq("payment", 5000, "owner"), ledgerReq("expense", 5000, "m"), ledgerReq("remittance", 10000, "m"), ledgerReq("payout", 3000, "m")} {
		r.Revision = l.Revision
		next, e := applyLedger(l, r, testOwner, j)
		if e != nil {
			t.Fatal(e)
		}
		l = next
		repeat, e := applyLedger(l, r, testOwner, j)
		if e != nil || repeat.Revision != l.Revision {
			t.Fatal("not idempotent", e)
		}
	}
	x := ledgerTotals(l, "2026-11-10")
	if x.Received != 17000 || x.Outstanding != 3000 || x.Overdue != 3000 || x.Costs != 5000 || x.Gross != 15000 || x.Held["m"] != 2000 || x.Held["owner"] != 15000 || x.Earned["owner"] != 5200 || x.Earned["m"] != 9800 || x.Payable["m"] != 6800 || x.Reimburse["m"] != 5000 {
		t.Fatalf("%+v", x)
	}
	r := ledgerReq("refund", 1000, "owner")
	r.Revision = l.Revision
	l, e := applyLedger(l, r, testOwner, j)
	if e != nil {
		t.Fatal(e)
	}
	if ledgerTotals(l, "2026-11-10").Outstanding != 4000 {
		t.Fatal("refund balance")
	}
}
func TestLedgerReversalLegacyAndRole(t *testing.T) {
	j := fixture(t)
	j.Report.ReceivedCents = 10000
	l := newLedger(j)
	if ledgerTotals(l, "2026-11-10").Received != 0 {
		t.Fatal("implicit migration")
	}
	r := ledgerReq("legacy_payment", 10000, "m")
	if _, e := applyLedger(l, r, testMechanic, j); e != ErrForbidden {
		t.Fatal(e)
	}
	var e error
	l, e = applyLedger(l, r, testOwner, j)
	if e != nil {
		t.Fatal(e)
	}
	r = ledgerReq("legacy_payment", 10000, "m")
	r.Revision = l.Revision
	if _, e = applyLedger(l, r, testOwner, j); e == nil {
		t.Fatal("duplicate historical payment")
	}
	reversal := ledgerReq("reverse", 0, "")
	reversal.Entry.Reverses = l.Entries[0].ID
	reversal.Revision = l.Revision
	l, e = applyLedger(l, reversal, testOwner, j)
	if e != nil {
		t.Fatal(e)
	}
	if ledgerTotals(l, "2026-11-10").Received != 0 || len(l.Entries) != 2 {
		t.Fatal("reversal lost audit")
	}
	reversal.Entry.ID = randomID()
	reversal.Revision = l.Revision
	if _, e = applyLedger(l, reversal, testOwner, j); e == nil {
		t.Fatal("double reversal")
	}
	if validTerms(LedgerTerms{Final: 10, Shares: []Share{{Participant: "m", BasisPoints: 9223372036854775807}}}) {
		t.Fatal("overflow")
	}
}
func TestEmulatorLedgerAPIAndConcurrency(t *testing.T) {
	s := emulator(t)
	j := fixture(t)
	j.ID = randomID()
	j.Mechanic = testMechanic.ID
	j.Report.ReceivedCents = 15000
	_, e := s.Collection("jobs").Doc(j.ID).Set(context.Background(), j)
	if e != nil {
		t.Fatal(e)
	}
	for _, method := range []string{"GET", "POST"} {
		w := httptest.NewRecorder()
		s.job(w, httptest.NewRequest(method, "/", bytes.NewReader([]byte(`{}`))), testMechanic, j.ID+"/ledger")
		if w.Code != 403 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	var wg sync.WaitGroup
	results := make(chan int, 2)
	r := ledgerReq("payment", 1000, "owner")
	b, _ := json.Marshal(r)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			s.job(w, httptest.NewRequest("POST", "/", bytes.NewReader(b)), testOwner, j.ID+"/ledger")
			results <- w.Code
		}()
	}
	wg.Wait()
	close(results)
	for code := range results {
		if code != 200 {
			t.Fatal(code)
		}
	}
	w := httptest.NewRecorder()
	s.job(w, httptest.NewRequest("GET", "/", nil), testOwner, j.ID+"/ledger")
	var response struct {
		Ledger Ledger       `json:"ledger"`
		Totals LedgerTotals `json:"totals"`
	}
	if json.Unmarshal(w.Body.Bytes(), &response) != nil || response.Ledger.Revision != 1 || response.Totals.Received != 1000 {
		t.Fatal(w.Body.String())
	}
	r.Entry.Cents = 2000
	b, _ = json.Marshal(r)
	w = httptest.NewRecorder()
	s.job(w, httptest.NewRequest("POST", "/", bytes.NewReader(b)), testOwner, j.ID+"/ledger")
	if w.Code != 409 {
		t.Fatal("key reused with different amount", w.Code)
	}
	w = httptest.NewRecorder()
	s.ledgerReport(w, httptest.NewRequest("GET", "/", nil), testOwner)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
}
