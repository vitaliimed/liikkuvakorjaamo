package crm

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIntakeReferenceBoundaries(t *testing.T) {
	for _, ref := range []string{"https://evil.invalid/x", "gs://other/lead-photos/x", "gs://private/secret/x", "gs://private/lead-photos/../secret", "gs://private/lead-photos/x?alt=media"} {
		if _, e := intakeObject(ref, "private", "lead-photos"); e == nil {
			t.Fatal(ref)
		}
	}
	if obj, e := intakeObject("gs://private/lead-photos/x.jpg", "private", "lead-photos"); e != nil || obj != "lead-photos/x.jpg" {
		t.Fatal(obj, e)
	}
}
func TestFinanceOnlyOwner(t *testing.T) {
	w := httptest.NewRecorder()
	(&Server{}).finance(w, httptest.NewRequest("GET", "/", nil), testMechanic)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
}
func TestEmulatorFinanceAllPagesAndIntakeScope(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	for i := 0; i < 105; i++ {
		j := fixture(t)
		j.ID = randomID()
		j.Status = "closed"
		j.Report.Work = "done"
		j.Report.ReceivedCents = 100
		j.Report.Payment = "cash"
		if _, e := s.Collection("jobs").Doc(j.ID).Set(ctx, j); e != nil {
			t.Fatal(e)
		}
	}
	w := httptest.NewRecorder()
	s.finance(w, httptest.NewRequest("GET", "/", nil), testOwner)
	var f FinanceSummary
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &f) != nil || f.Total.Jobs != 105 || f.Total.AcceptedCents != 10500 {
		t.Fatal(w.Code, w.Body.String())
	}
	j := fixture(t)
	j.ID = randomID()
	j.Mechanic = "other"
	j.IntakePhotos = []string{"gs://private/lead-photos/x.jpg"}
	_, _ = s.Collection("jobs").Doc(j.ID).Set(ctx, j)
	w = httptest.NewRecorder()
	s.job(w, httptest.NewRequest("GET", "/", nil), testMechanic, j.ID+"/intake-photos/0")
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
}

func TestEmulatorCompleteRepairReview(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	putUser(t, s, testMechanic)
	j := fixture(t)
	j.ID = randomID()
	if _, e := s.Collection("jobs").Doc(j.ID).Set(ctx, j); e != nil {
		t.Fatal(e)
	}
	p := testProposal()
	p.Start = time.Now().UTC().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)
	p.End = p.Start.Add(time.Hour)
	steps := []struct {
		a Action
		u User
	}{
		{Action{Kind: "assign", Mechanic: "m"}, testOwner},
		{Action{Kind: "propose", Proposal: p}, testMechanic},
		{Action{Kind: "client_confirm", Text: "Test client agreed by phone"}, testOwner},
		{Action{Kind: "confirm"}, testOwner},
		{Action{Kind: "start"}, testMechanic},
		{Action{Kind: "report", Report: Report{Diagnosis: "Battery failed", Parts: "Battery", Work: "Battery replaced", Result: "partial", Remaining: "Check charging", Payment: "cash", ReceivedCents: 12000}}, testMechanic},
		{Action{Kind: "reopen", Text: "Please check charging"}, testOwner},
		{Action{Kind: "report", Report: Report{Diagnosis: "Charging verified", Parts: "Battery", Work: "Charging checked", Result: "repaired", Payment: "cash", ReceivedCents: 12000}}, testMechanic},
		{Action{Kind: "close"}, testOwner},
	}
	for _, step := range steps {
		step.a.Revision = j.Revision
		next, e := s.mutate(ctx, j.ID, step.a, step.u)
		if e != nil {
			t.Fatalf("%s: %v", step.a.Kind, e)
		}
		j = next
	}
	if j.Status != "closed" || j.Report.ReceivedCents != 12000 {
		t.Fatal(j)
	}
	ds, e := s.Collection("jobs").Doc(j.ID).Collection("events").Documents(ctx).GetAll()
	if e != nil || len(ds) != len(steps) {
		t.Fatal("Incomplete audit", len(ds), e)
	}
}

func TestAgreementSnapshotAndReschedule(t *testing.T) {
	j := fixture(t)
	j.Mechanic = "m"
	j.Status = "proposed"
	j.Proposal = testProposal()
	j = step(t, j, Action{Kind: "client_confirm", Text: "Agreed"}, testOwner)
	if j.Agreed == nil || j.Agreed.Proposal.WorkCents != 7000 {
		t.Fatal("agreement missing")
	}
	j.Proposal.WorkCents = 9000
	if j.Agreed.Proposal.WorkCents != 7000 {
		t.Fatal("snapshot changed")
	}
	j = step(t, j, Action{Kind: "reschedule", Text: "New time requested"}, testOwner)
	if j.Agreed != nil {
		t.Fatal("old consent reused")
	}
}

type fakeTranslator struct{}

func (fakeTranslator) Translate(_ context.Context, text, from, to string) (string, error) {
	return "Перевод: " + text, nil
}
func TestEmulatorTranslationAndCompleteCalendar(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	for i := 0; i < 102; i++ {
		j := fixture(t)
		j.ID = randomID()
		j.Mechanic = "m"
		if i == 101 {
			j.Mechanic = "other"
		}
		j.Status = "scheduled"
		j.Proposal = testProposal()
		if _, e := s.Collection("jobs").Doc(j.ID).Set(ctx, j); e != nil {
			t.Fatal(e)
		}
	}
	for _, tc := range []struct {
		u     User
		count int
	}{{testOwner, 102}, {testMechanic, 101}} {
		w := httptest.NewRecorder()
		s.calendarList(w, httptest.NewRequest("GET", "/", nil), tc.u)
		var response struct {
			Jobs []Job `json:"jobs"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &response) != nil || len(response.Jobs) != tc.count {
			t.Fatal(w.Code, len(response.Jobs), w.Body.String())
		}
	}
	j := fixture(t)
	j.ID = randomID()
	_, e := s.Collection("jobs").Doc(j.ID).Set(ctx, j)
	if e != nil {
		t.Fatal(e)
	}
	m, e := s.RecordInboundMessage(ctx, j.ID, "whatsapp", "translation-test", "client", "Hei", "fi", nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.TranslateMessage(ctx, testMechanic, j.ID, m.ID, fakeTranslator{}); e != ErrForbidden {
		t.Fatal("mechanic translated", e)
	}
	translated, e := s.TranslateMessage(ctx, testOwner, j.ID, m.ID, fakeTranslator{})
	if e != nil || translated.Original != "Hei" || translated.Translation != "Перевод: Hei" {
		t.Fatal(translated, e)
	}
}
