package crm

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
var testOwner = User{ID: "o", Role: "owner", Active: true}
var testMechanic = User{ID: "m", Role: "mechanic", Active: true, MainCity: "Helsinki", Shifts: []Shift{{0, "00:00", "23:59"}, {1, "00:00", "23:59"}, {2, "00:00", "23:59"}, {3, "00:00", "23:59"}, {4, "00:00", "23:59"}, {5, "00:00", "23:59"}, {6, "00:00", "23:59"}}}

func fixture(t *testing.T) Job {
	t.Helper()
	j, e := NewJob(strings.Repeat("a", 64), "abc-123", "Client", "+358401234567", "Helsinki", "No start", "liikkuva", "fi")
	if e != nil {
		t.Fatal(e)
	}
	return j
}
func testProposal() Proposal {
	return Proposal{Start: testNow.Add(time.Hour), End: testNow.Add(2 * time.Hour), VisitCents: 5000, WorkCents: 7000, Payment: "cash", Plan: "Check battery"}
}
func step(t *testing.T, j Job, a Action, u User) Job {
	t.Helper()
	a.Revision = j.Revision
	n, e := Apply(j, a, u, testNow)
	if e != nil {
		t.Fatal(e)
	}
	return n
}
func TestFullWorkflow(t *testing.T) {
	j := fixture(t)
	j = step(t, j, Action{Kind: "assign", Mechanic: "m"}, testOwner)
	j = step(t, j, Action{Kind: "propose", Proposal: testProposal()}, testMechanic)
	j = step(t, j, Action{Kind: "client_confirm", Text: "Agreed by phone"}, testOwner)
	j = step(t, j, Action{Kind: "confirm"}, testOwner)
	j = step(t, j, Action{Kind: "start"}, testMechanic)
	j = step(t, j, Action{Kind: "report", Report: Report{Diagnosis: "Battery tested", Parts: "Battery", ReceivedCents: 12000, Work: "Battery replaced", Result: "repaired", Payment: "cash"}}, testMechanic)
	j = step(t, j, Action{Kind: "close"}, testOwner)
	if j.Status != "closed" || j.Revision != 8 || j.Report.ReceivedCents != 12000 {
		t.Fatal(j)
	}
}
func TestMechanicPermissions(t *testing.T) {
	j := fixture(t)
	j.Mechanic = "m"
	j.Status = "proposed"
	for _, u := range []User{{ID: "other", Role: "mechanic", Active: true}, {ID: "m", Role: "mechanic", Active: false}} {
		if u.CanRead(j) {
			t.Fatal("unauthorized read")
		}
	}
	for _, kind := range []string{"assign", "client_confirm", "confirm", "reschedule", "cancel", "close", "reopen"} {
		_, e := Apply(j, Action{Kind: kind, Revision: 1}, testMechanic, testNow)
		if !errors.Is(e, ErrForbidden) {
			t.Fatalf("%s: %v", kind, e)
		}
	}
}
func TestCannotSkipConfirmation(t *testing.T) {
	j := fixture(t)
	j.Mechanic = "m"
	j.Status = "proposed"
	j.Proposal = testProposal()
	for _, a := range []Action{{Kind: "confirm", Revision: 1}, {Kind: "client_confirm", Revision: 1}} {
		if _, e := Apply(j, a, testOwner, testNow); e == nil {
			t.Fatal("confirmation evidence skipped")
		}
	}
}
func TestStaleWrite(t *testing.T) {
	j := fixture(t)
	j.Revision = 4
	_, e := Apply(j, Action{Kind: "assign", Revision: 3, Mechanic: "m"}, testOwner, testNow)
	if !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
}
func TestInvalidProposals(t *testing.T) {
	j := fixture(t)
	j.Mechanic = "m"
	j.Status = "assigned"
	cases := []func(*Proposal){func(p *Proposal) { p.WorkCents = -1 }, func(p *Proposal) { p.Plan = " " }, func(p *Proposal) { p.Start = testNow.Add(-time.Hour) }, func(p *Proposal) { p.End = p.Start }, func(p *Proposal) { p.End = p.Start.Add(13 * time.Hour) }, func(p *Proposal) { p.Payment = "unknown" }}
	for i, f := range cases {
		p := testProposal()
		f(&p)
		if _, e := Apply(j, Action{Kind: "propose", Revision: 1, Proposal: p}, testMechanic, testNow); e == nil {
			t.Fatalf("invalid %d", i)
		}
	}
}
func TestIncompleteReport(t *testing.T) {
	j := fixture(t)
	j.Mechanic = "m"
	j.Status = "in_progress"
	r := Report{Diagnosis: "Inspected", Parts: "None", Result: "partial", Work: "Diagnosed", Payment: "cash"}
	if _, e := Apply(j, Action{Kind: "report", Revision: 1, Report: r}, testMechanic, testNow); e == nil {
		t.Fatal("missing next step accepted")
	}
	r.Remaining = "Order parts"
	j = step(t, j, Action{Kind: "report", Report: r}, testMechanic)
	if j.Status != "review" {
		t.Fatal(j.Status)
	}
}
func TestRescheduleClearsConsent(t *testing.T) {
	j := fixture(t)
	j.Status = "scheduled"
	j.Mechanic = "m"
	j.Proposal = testProposal()
	j = step(t, j, Action{Kind: "reschedule", Text: "Client requested"}, testOwner)
	if j.Status != "assigned" {
		t.Fatal(j.Status)
	}
	if _, e := Apply(j, Action{Kind: "confirm", Revision: j.Revision}, testOwner, testNow); e == nil {
		t.Fatal("old consent reused")
	}
}
func TestPlate(t *testing.T) {
	for _, s := range []string{"ABC-123", "abc 123", " ABC123 "} {
		p, e := Plate(s)
		if e != nil || p != "ABC123" {
			t.Fatal(p, e)
		}
	}
	for _, s := range []string{"", "<script>", "АВС123", "../../file"} {
		if _, e := Plate(s); e == nil {
			t.Fatal(s)
		}
	}
}
func TestOverlap(t *testing.T) {
	a := testProposal()
	b := a
	b.Start = a.End
	b.End = b.Start.Add(time.Hour)
	if Overlap(a, b) {
		t.Fatal("adjacent overlap")
	}
	b.Start = a.End.Add(-time.Minute)
	if !Overlap(a, b) {
		t.Fatal("overlap missed")
	}
}
func TestDraft(t *testing.T) {
	j := fixture(t)
	j.Proposal = testProposal()
	s := Draft(j)
	for _, v := range []string{"13:00", "50,00 €", "70,00 €", "120,00 €", "käteisellä"} {
		if !strings.Contains(s, v) {
			t.Fatalf("missing %s", v)
		}
	}
}
func TestCSRFAndAuth(t *testing.T) {
	s := &Server{URL: "https://example.com"}
	for _, tc := range []struct {
		path, method, header, origin string
		code                         int
	}{{"/crm/api/jobs", "POST", "", "", 403}, {"/crm/api/jobs", "POST", "1", "https://evil.invalid", 403}, {"/crm/api/jobs", "GET", "", "", 401}, {"/crm/api/v1/leads", "POST", "", "", 403}, {"/crm/internal/tick", "POST", "", "", 403}, {"/crm/bot/telegram", "POST", "", "", 403}} {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		r.Header.Set("X-CRM-Request", tc.header)
		r.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != tc.code {
			t.Errorf("%s: %d", tc.path, w.Code)
		}
	}
}
func TestCalendarInjection(t *testing.T) {
	j := fixture(t)
	j.Status = "scheduled"
	j.Proposal = testProposal()
	j.Name = "X\nEND:VEVENT\nBEGIN:VEVENT"
	w := httptest.NewRecorder()
	(&Server{URL: "https://example.com"}).calendar(w, httptest.NewRequest("GET", "/", nil), j)
	body := w.Body.String()
	if strings.Count(body, "\r\nBEGIN:VEVENT\r\n") != 1 || !strings.Contains(body, "DTSTART:20261108T110000Z") {
		t.Fatal(body)
	}
}
func TestUnconfirmedCalendar(t *testing.T) {
	w := httptest.NewRecorder()
	(&Server{}).calendar(w, httptest.NewRequest("GET", "/", nil), fixture(t))
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
func TestStaleNotification(t *testing.T) {
	j := fixture(t)
	hash := noticeFingerprint(j)
	j.Proposal.WorkCents++
	if hash == noticeFingerprint(j) {
		t.Fatal("price change missed")
	}
	j.Proposal.WorkCents--
	j.Status = "cancelled"
	if hash == noticeFingerprint(j) {
		t.Fatal("cancel missed")
	}
}
