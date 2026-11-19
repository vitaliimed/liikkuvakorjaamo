package crm

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAppointmentRemindersIgnoreUnrelatedEdits(t *testing.T) {
	now := time.Now()
	j := Job{ID: "job", Mechanic: "m", Status: "scheduled", Proposal: Proposal{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour)}}
	n := Notice{Appointment: appointmentFingerprint(j), Expires: j.Proposal.Start}
	j.Revision++
	j.Report.Work = "edited"
	j.Proposal.Plan = "updated plan"
	if noticeObsolete(n, j, now) {
		t.Fatal("unrelated edits invalidate reminder")
	}
	for _, change := range []func(*Job){func(j *Job) { j.Status = "cancelled" }, func(j *Job) { j.Mechanic = "other" }, func(j *Job) { j.Proposal.Start = j.Proposal.Start.Add(time.Hour) }} {
		changed := j
		change(&changed)
		if !noticeObsolete(n, changed, now) {
			t.Fatal("stale reminder accepted")
		}
	}
	if !noticeObsolete(n, j, j.Proposal.Start) {
		t.Fatal("expired reminder accepted")
	}
}
func TestCalendarTwoAlarmsAndStableUID(t *testing.T) {
	j := Job{ID: "job", Status: "scheduled", Proposal: Proposal{Start: time.Now().Add(48 * time.Hour), End: time.Now().Add(49 * time.Hour)}}
	s := &Server{URL: "https://crm.example"}
	w := httptest.NewRecorder()
	s.calendar(w, httptest.NewRequest("GET", "/", nil), j)
	for _, want := range []string{"UID:job@liikkuva-crm", "TRIGGER:-P1D", "TRIGGER:-PT60M"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatal(want)
		}
	}
	text := appointmentText(j, s.URL, "Confirmed")
	if !strings.Contains(text, "/crm/api/jobs/job/calendar.ics") || !strings.Contains(text, "Хельсинки") {
		t.Fatal(text)
	}
}

func TestEmulatorConfirmedVisitQueuesOneCalendarNotice(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	putUser(t, s, testMechanic)
	j := fixture(t)
	j.ID = randomID()
	j.Status = "client_confirmed"
	j.Mechanic = "m"
	j.Proposal = testProposal()
	j.Proposal.Start = time.Now().UTC().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)
	j.Proposal.End = j.Proposal.Start.Add(time.Hour)
	if _, e := s.Collection("jobs").Doc(j.ID).Set(ctx, j); e != nil {
		t.Fatal(e)
	}
	action := Action{Kind: "confirm", Revision: j.Revision}
	if _, e := s.mutate(ctx, j.ID, action, testOwner); e != nil {
		t.Fatal(e)
	}
	if _, e := s.mutate(ctx, j.ID, action, testOwner); e == nil {
		t.Fatal("repeated confirmation accepted")
	}
	docs, e := s.Collection("outbox").Where("Target", "==", "m").Documents(ctx).GetAll()
	if e != nil || len(docs) != 1 {
		t.Fatalf("notices %d: %v", len(docs), e)
	}
	var n Notice
	if e = docs[0].DataTo(&n); e != nil || !strings.Contains(n.Text, "calendar.ics") {
		t.Fatal("calendar notice missing", e)
	}
	d, e := s.Collection("schedules").Doc("m").Get(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var schedule schedule
	if e = d.DataTo(&schedule); e != nil || len(schedule.Bookings) != 1 {
		t.Fatal("duplicate calendar entry", e)
	}
}

func TestEmulatorReminderTickDoesNotDuplicateAfterEdit(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	token := strings.Repeat("x", 32)
	t.Setenv("CRM_INTERNAL_TOKEN", token)
	t.Setenv("CRM_TELEGRAM_BOT_TOKEN", "")
	j := Job{ID: "reminder-job", Mechanic: "m", Status: "scheduled", Proposal: Proposal{Start: time.Now().Add(12 * time.Hour), End: time.Now().Add(13 * time.Hour)}}
	tick := func() {
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		s.tickHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	for i := 0; i < 2; i++ {
		j.Revision++
		if _, e := s.Collection("jobs").Doc(j.ID).Set(ctx, j); e != nil {
			t.Fatal(e)
		}
		tick()
	}
	docs, e := s.Collection("outbox").Documents(ctx).GetAll()
	if e != nil || len(docs) != 2 {
		t.Fatalf("expected owner and mechanic reminder once: %d %v", len(docs), e)
	}
	for _, d := range docs {
		var n Notice
		if e = d.DataTo(&n); e != nil || n.Expires.IsZero() || n.Appointment == "" {
			t.Fatal("reminder validity missing", e)
		}
	}
}
