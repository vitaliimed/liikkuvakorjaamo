package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"liikkuvakorjaamo.fi/landing/crm"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProductionRegistrationRequired(t *testing.T) {
	for _, lang := range []string{"fi", "en"} {
		l := Lead{Name: "Release Test", Phone: "+358401234567", Language: lang}
		if validateLead(l)["car"] == "" {
			t.Fatal("empty plate accepted")
		}
		l.Car = "ABC-123"
		if len(validateLead(l)) != 0 {
			t.Fatal(validateLead(l))
		}
	}
}
func TestProductionPublicRoutesAndCanonicals(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	for _, tc := range []struct{ path, canonical string }{{"/", "/"}, {"/en", "/en"}, {"/privacy?lang=en", "/privacy?lang=en"}, {"/hinnasto", "/hinnasto"}, {"/palvelualue", "/palvelualue"}, {"/vikadiagnostiikka", "/vikadiagnostiikka"}} {
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, httptest.NewRequest("GET", f.cfg.SiteURL+tc.path, nil))
		if w.Code != 200 {
			t.Errorf("%s: %d", tc.path, w.Code)
		}
		if tc.path == "/" || tc.path == "/en" || strings.HasPrefix(tc.path, "/privacy") {
			if !strings.Contains(w.Body.String(), `rel="canonical" href="`+f.cfg.SiteURL+tc.canonical+`"`) {
				t.Errorf("canonical missing: %s", tc.path)
			}
		}
	}
}
func TestEmulatorLeadAndCRMAtomicSaveLeaseAndReplay(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("Firestore emulator required")
	}
	cfg := Config{FirestoreProjectID: fmt.Sprintf("demo-release-%d", time.Now().UnixNano()), FirestoreLeadsCollection: "liikkuva_release_tests"}
	ctx := context.Background()
	l := Lead{Name: "Synthetic Release Test", Phone: "+358401234567", Address: "Synthetic test address", Car: "TST-123", Language: "fi", Message: "Synthetic emulator-only test", PhotoURLs: []string{"gs://demo-liikkuva-files/test.png"}}
	id, err := saveLeadRecord(ctx, cfg, l)
	if err != nil {
		t.Fatal(err)
	}
	client, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	hash := sha256.Sum256([]byte("liikkuva:" + id))
	jobID := hex.EncodeToString(hash[:])
	doc, err := client.Collection("crm_jobs").Doc(jobID).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var job crm.Job
	if err = doc.DataTo(&job); err != nil {
		t.Fatal(err)
	}
	if len(job.IntakePhotos) != 1 || job.Plate != "TST123" {
		t.Fatal("CRM intake not preserved")
	}
	release, err := acquireLeadLease(ctx, cfg, id)
	if err != nil {
		t.Fatal(err)
	}
	if other, e := acquireLeadLease(ctx, cfg, id); e == nil {
		other()
		t.Fatal("concurrent worker admitted")
	}
	release()
	if err = markLeadDone(ctx, cfg, id); err != nil {
		t.Fatal(err)
	}
	// A completed no-email record needs no external configuration on replay.
	if err = processLeadFromRecord(ctx, cfg, id); err != nil {
		t.Fatal(err)
	}
	if err = processLeadFromRecord(ctx, cfg, id); err != nil {
		t.Fatal(err)
	}
	after, err := client.Collection("crm_jobs").Doc(jobID).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !after.UpdateTime.Equal(doc.UpdateTime) {
		t.Fatal("replay changed CRM job")
	}
	if _, err = client.Collection(cfg.FirestoreLeadsCollection).Doc(id).Get(ctx); err != nil {
		t.Fatal(err)
	}
}
