package crm

import (
	"bytes"
	"context"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func emulator(t *testing.T) *Server {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("Firestore emulator required")
	}
	ctx := context.Background()
	s, e := Open(ctx, "demo-crm-"+randomID()[:12], "(default)", "", "https://crm.example")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.DB.Close() })
	return s
}
func putUser(t *testing.T, s *Server, u User) {
	t.Helper()
	if _, e := s.Collection("users").Doc(u.ID).Set(context.Background(), u); e != nil {
		t.Fatal(e)
	}
}
func TestEmulatorConcurrentBookingAndAudit(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	putUser(t, s, testMechanic)
	ids := []string{randomID(), randomID()}
	for _, id := range ids {
		j := fixture(t)
		j.ID = id
		j.Status = "client_confirmed"
		j.Mechanic = "m"
		j.Proposal = testProposal()
		j.Proposal.Start = time.Now().UTC().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)
		j.Proposal.End = j.Proposal.Start.Add(time.Hour)
		if _, e := s.Collection("jobs").Doc(id).Set(ctx, j); e != nil {
			t.Fatal(e)
		}
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, e := s.mutate(ctx, id, Action{Kind: "confirm", Revision: 1}, testOwner)
			results <- e
		}(id)
	}
	wg.Wait()
	close(results)
	ok, conflict := 0, 0
	for e := range results {
		if e == nil {
			ok++
		} else if strings.Contains(e.Error(), "уже есть выезд") {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("accepted=%d rejected=%d", ok, conflict)
	}
	ds, e := s.Collection("schedules").Doc("m").Get(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var sched schedule
	if e = ds.DataTo(&sched); e != nil || len(sched.Bookings) != 1 {
		t.Fatal(e, sched)
	}
	id := sched.Bookings[0].Job
	events, e := s.Collection("jobs").Doc(id).Collection("events").Documents(ctx).GetAll()
	if e != nil || len(events) != 1 {
		t.Fatal("audit missing", e)
	}
	var event Event
	if e = events[0].DataTo(&event); e != nil || event.Snapshot.Status != "scheduled" {
		t.Fatal(e)
	}
	if _, e = s.mutate(ctx, id, Action{Kind: "reschedule", Revision: 2, Text: "Client requested"}, testOwner); e != nil {
		t.Fatal(e)
	}
	ds, _ = s.Collection("schedules").Doc("m").Get(ctx)
	_ = ds.DataTo(&sched)
	if len(sched.Bookings) != 0 {
		t.Fatal("old slot not released")
	}
}
func TestEmulatorSessionRevocationAndScopedRead(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	u := testMechanic
	u.Email = "mechanic@example.invalid"
	u.ID = digest(u.Email)
	hash, _ := bcrypt.GenerateFromPassword([]byte("test-password-123"), bcrypt.MinCost)
	u.Hash = string(hash)
	putUser(t, s, u)
	r := httptest.NewRequest("POST", "/crm/api/login", bytes.NewBufferString(`{"email":"mechanic@example.invalid","password":"test-password-123"}`))
	r.Header.Set("X-CRM-Request", "1")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("no session")
	}
	request := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(cookies[0])
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	for _, assignee := range []string{u.ID, "other"} {
		j := fixture(t)
		j.ID = randomID()
		j.Mechanic = assignee
		if _, e := s.Collection("jobs").Doc(j.ID).Set(ctx, j); e != nil {
			t.Fatal(e)
		}
		w = request("/crm/api/jobs/" + j.ID)
		want := 200
		if assignee != "other" {
			want = 200
		} else {
			want = 404
		}
		if w.Code != want {
			t.Fatal(w.Code, want)
		}
	}
	w = request("/crm/api/jobs")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var list struct {
		Jobs []Job `json:"jobs"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list.Jobs) != 1 {
		t.Fatal("scope leak", w.Body.String())
	}
	u.Active = false
	putUser(t, s, u)
	if request("/crm/api/me").Code != http.StatusUnauthorized {
		t.Fatal("disabled session still valid")
	}
	u.Active = true
	u.Hash = "changed"
	putUser(t, s, u)
	if request("/crm/api/me").Code != 401 {
		t.Fatal("password change did not revoke")
	}
}
func TestEmulatorIntakeAtomicityAndIdempotency(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	b := s.DB.Batch()
	ref := s.DB.Collection("leads").Doc("test")
	b.Create(ref, map[string]string{"source": "test"})
	if e := AddLeadToBatch(s.DB, b, "test", "ABC123", "Test", "+358401234567", "", "Helsinki", "No start", "liikkuva", "fi", nil); e != nil {
		t.Fatal(e)
	}
	if _, e := b.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	jobs, e := s.Collection("jobs").Documents(ctx).GetAll()
	if e != nil || len(jobs) != 1 {
		t.Fatal(e, len(jobs))
	}
	t.Setenv("CRM_INTAKE_TOKEN", strings.Repeat("x", 32))
	body := `{"plate":"ABC123","name":"Test","phone":"+358401234567","address":"Helsinki","description":"Problem","brand":"autohelp","language":"fi"}`
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest("POST", "/crm/api/v1/leads", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
		r.Header.Set("Idempotency-Key", "unique-event-id-123456")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 201 && w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	jobs, _ = s.Collection("jobs").Documents(ctx).GetAll()
	if len(jobs) != 2 {
		t.Fatal("duplicate intake", len(jobs))
	}
}
