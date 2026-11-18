package crm

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAutohelpTariffsPreserveBrandAndRejectStaleWrites(t *testing.T) {
	old := DefaultPriceBook()
	next := autohelpBook(old)
	next.Items[0].Cents = 6500
	merged, err := mergeAutohelpBook(old, next)
	if err != nil || merged.Revision != 1 {
		t.Fatalf("merge: %+v %v", merged, err)
	}
	if merged.Items[0] != old.Items[0] || merged.Items[1] != old.Items[1] {
		t.Fatal("changed Liikkuva")
	}
	retry, err := mergeAutohelpBook(merged, next)
	if err != nil || retry.Revision != 1 {
		t.Fatal("retry created another revision")
	}
	next.Items[0].Cents = 7000
	if _, err := mergeAutohelpBook(merged, next); err != ErrConflict {
		t.Fatal("stale edit accepted", err)
	}
	next.Items[0].Brand = "liikkuva"
	if validateAutohelpBook(next) == nil {
		t.Fatal("cross-brand tariff accepted")
	}
}

func TestAutohelpImportKeepsSubmissionPrices(t *testing.T) {
	current := DefaultPriceBook()
	snapshot := autohelpBook(current)
	snapshot.Items[0].Cents = 4200
	snapshot.Items[1].Cents, snapshot.Items[1].Active = 5500, true
	j := Job{Brand: "autohelp"}
	applyAutohelpLeadPrices(&j, snapshot.Items, 0)
	if j.Proposal.VisitCents != 4200 || j.Proposal.WorkCents != 5500 {
		t.Fatal("lost frozen prices", j.Proposal)
	}
	applyAutohelpLeadPrices(&j, nil, 35)
	if j.Proposal.VisitCents != 3500 {
		t.Fatal("lost historical amount")
	}
	applyAutohelpLeadPrices(&j, nil, 0)
	if j.Proposal.VisitCents != 0 || j.Proposal.WorkCents != 0 || !strings.Contains(j.Description, "не сохранён") {
		t.Fatal("invented historical prices")
	}
}

func TestAutohelpTariffTokenIsSeparate(t *testing.T) {
	t.Setenv("CRM_AUTOHELP_TARIFF_TOKEN", strings.Repeat("a", 40))
	t.Setenv("CRM_INTERNAL_TOKEN", strings.Repeat("b", 40))
	s := &Server{}
	for _, token := range []string{"", strings.Repeat("b", 40)} {
		r := httptest.NewRequest("GET", "/crm/internal/autohelp-tariffs", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("wrong token accepted", w.Code)
		}
	}
}

func TestEmulatorAutohelpTariffSharedRevision(t *testing.T) {
	s := emulator(t)
	t.Setenv("CRM_AUTOHELP_TARIFF_TOKEN", strings.Repeat("a", 40))
	book := autohelpBook(DefaultPriceBook())
	book.Items[0].Cents = 6350
	post := func(book PriceBook) *httptest.ResponseRecorder {
		body, _ := json.Marshal(book)
		r := httptest.NewRequest("POST", "/crm/internal/autohelp-tariffs", bytes.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+strings.Repeat("a", 40))
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	if w := post(book); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := post(book); w.Code != 200 {
		t.Fatal("retry", w.Code, w.Body.String())
	}
	book.Items[0].Cents = 7900
	if w := post(book); w.Code != 409 {
		t.Fatal("stale", w.Code, w.Body.String())
	}
}
