package crm

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEmulatorHTTPPrivateData(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	j := fixture(t)
	j.ID = randomID()
	j.Mechanic = "m"
	old := j
	old.ID = randomID()
	old.Mechanic = "other"
	old.Name = "PRIVATE_PREVIOUS_NAME"
	old.Phone = "PRIVATE_PREVIOUS_PHONE"
	old.Report.ReceivedCents = 987654
	old.Proposal.WorkCents = 876543
	other := old
	other.ID = randomID()
	other.Vehicle = "different"
	for _, x := range []Job{j, old, other} {
		if _, e := s.Collection("jobs").Doc(x.ID).Set(ctx, x); e != nil {
			t.Fatal(e)
		}
	}
	for _, kind := range []string{"message", "report", "internal_note"} {
		_, e := s.Collection("jobs").Doc(j.ID).Collection("events").NewDoc().Set(ctx, Event{Type: kind, Actor: "o", Text: "PRIVATE_MESSAGE", At: time.Now(), Snapshot: &old})
		if e != nil {
			t.Fatal(e)
		}
	}
	call := func(u User, method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/crm/api/jobs/"+path, strings.NewReader(body))
		s.job(w, r, u, path)
		return w
	}
	for _, path := range []string{j.ID, j.ID + "/events", j.ID + "/vehicle-history"} {
		w := call(testMechanic, "GET", path, "")
		if w.Code != 200 {
			t.Fatal(path, w.Code, w.Body.String())
		}
		for _, secret := range []string{"PRIVATE_", "987654", "876543", `"snapshot"`, `"draft"`} {
			if strings.Contains(w.Body.String(), secret) {
				t.Fatal(path, secret, w.Body.String())
			}
		}
	}
	for _, path := range []string{other.ID, other.ID + "/photos/x", j.ID + "/vehicle-photos/" + other.ID + "/x", j.ID + "/vehicle-photos/" + other.ID + "/x?original=1"} {
		if w := call(testMechanic, "GET", path, ""); w.Code != 404 {
			t.Fatal(path, w.Code)
		}
	}
	for _, path := range []string{j.ID + "/emails", j.ID + "/messages"} {
		if w := call(testMechanic, "GET", path, ""); w.Code != 403 {
			t.Fatal(path, w.Code)
		}
	}
	for _, direction := range []string{"incoming", "outgoing", "internal"} {
		w := call(testMechanic, "POST", j.ID+"/actions", `{"kind":"message","revision":1,"text":"pretend","channel":"email","direction":"`+direction+`"}`)
		if w.Code != 403 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	w := call(testOwner, "GET", j.ID+"/events", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "PRIVATE_MESSAGE") {
		t.Fatal("owner history lost", w.Body.String())
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "/crm/api/jobs/"+j.ID+"/photos/x", nil))
	if w.Code != 401 {
		t.Fatal("anonymous image access", w.Code)
	}
}
func TestEmulatorVehicleAliasesAndAdapter(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	j := fixture(t)
	j.ID = randomID()
	_, e := s.Collection("jobs").Doc(j.ID).Set(ctx, j)
	if e != nil {
		t.Fatal(e)
	}
	id, e := resolveVehicle(ctx, s.DB, j.Plate)
	if e != nil || id != j.Vehicle {
		t.Fatal(id, e)
	}
	w := httptest.NewRecorder()
	s.changePlate(w, httptest.NewRequest("POST", "/", strings.NewReader(`{"plate":"XYZ987","revision":1}`)), testOwner, j)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	next, e := resolveVehicle(ctx, s.DB, "XYZ987")
	if e != nil || next != id {
		t.Fatal(next, e)
	}
	for i := 0; i < 2; i++ {
		_, e = s.RecordInboundMessage(ctx, j.ID, "whatsapp", "event-1", "verified", "Hei", "fi", nil)
		if e != nil {
			t.Fatal(e)
		}
	}
	ds, e := s.Collection("jobs").Doc(j.ID).Collection("messages").Documents(ctx).GetAll()
	if e != nil || len(ds) != 1 {
		t.Fatal("duplicate messages", e)
	}
}
func TestPricesScheduleAndOriginal(t *testing.T) {
	j := fixture(t)
	if j.Proposal.VisitCents != 7000 || j.Proposal.WorkCents != 5000 {
		t.Fatal(j.Proposal)
	}
	book := DefaultPriceBook()
	applyPrices(&j, book)
	book.Items[0].Cents = 1
	if j.PricingSnapshot[0].Cents != 7000 {
		t.Fatal("snapshot changed")
	}
	u := testMechanic
	p := testProposal()
	if !CanServe(u, "Helsinki") || CanServe(u, "Espoo") {
		t.Fatal("zone")
	}
	u.DaysOff = []string{p.Start.In(Helsinki).Format("2006-01-02")}
	if WithinHours(u, p) {
		t.Fatal("holiday ignored")
	}
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2000, 1000)))
	original := append([]byte{}, b.Bytes()...)
	thumb, e := thumbnail(b.Bytes())
	if e != nil {
		t.Fatal(e)
	}
	cfg, _, e := image.DecodeConfig(bytes.NewReader(thumb))
	if e != nil || cfg.Width != 1800 || !bytes.Equal(original, b.Bytes()) {
		t.Fatal("thumbnail/original", cfg, e)
	}
}
