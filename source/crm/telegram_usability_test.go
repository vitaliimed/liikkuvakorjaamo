package crm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTelegramJobLabelsAndNewestFirst(t *testing.T) {
	now := time.Now()
	jobs := []Job{{ID: "old", Created: now.Add(-time.Hour)}, {ID: "new", Created: now, Name: "Иван", Description: "Не заводится"}}
	sortBotJobs(jobs)
	if jobs[0].ID != "new" {
		t.Fatal("newest job must be first")
	}
	label := botJobLabel(jobs[0], 1)
	for _, text := range []string{"Без номера", "Иван", "Не заводится"} {
		if !strings.Contains(label, text) {
			t.Fatal(label)
		}
	}
}
func TestTelegramFullCardUsesAgreement(t *testing.T) {
	j := Job{ID: "job", Name: "Customer", Phone: "123456789", Email: "client@example.test", Address: "Test street", Description: strings.Repeat("полное описание ", 400), Proposal: Proposal{VisitCents: 99900}, Agreed: &Agreement{Proposal: Proposal{VisitCents: 5000, WorkCents: 7000}}}
	card := botJobCard(j, "https://crm.example")
	for _, text := range []string{j.Name, j.Phone, j.Email, j.Address, j.Description, "50.00 €", "70.00 €", "Согласованная", "/crm/?job=job"} {
		if !strings.Contains(card, text) {
			t.Fatalf("missing %q", botShort(text, 40))
		}
	}
	if strings.Contains(card, "999.00") {
		t.Fatal("draft price shown as agreed")
	}
}
func TestDefaultOwnerShareFixed50(t *testing.T) {
	l := newLedger(Job{ID: "job"})
	if l.Terms.OwnerShare == nil || l.Terms.OwnerShare.Mode != "fixed" || l.Terms.OwnerShare.Value != 5000 || !validTerms(l.Terms) {
		t.Fatalf("%+v", l.Terms)
	}
}
func TestEmulatorStoredLedgerDoesNotInheritDefault(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	ref := s.Collection("ledgers").Doc("legacy")
	for _, terms := range []map[string]any{{"Final": int64(10000)}, {"Final": int64(10000), "OwnerShare": OwnerShare{Mode: "fixed", Value: 2000, Basis: "margin"}}} {
		if _, err := ref.Set(ctx, map[string]any{"Terms": terms}); err != nil {
			t.Fatal(err)
		}
		d, err := ref.Get(ctx)
		if err != nil {
			t.Fatal(err)
		}
		l := newLedger(Job{ID: "legacy"})
		if err = readStoredLedger(d, &l); err != nil {
			t.Fatal(err)
		}
		if _, ok := terms["OwnerShare"]; !ok {
			if l.Terms.OwnerShare != nil {
				t.Fatal("legacy terms changed")
			}
		} else if l.Terms.OwnerShare == nil || l.Terms.OwnerShare.Value != 2000 {
			t.Fatal("custom share overwritten")
		}
	}
}

type telegramCaptureTransport struct {
	texts     []string
	keyboards []bool
}

func (c *telegramCaptureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var payload struct {
		Text     string          `json:"text"`
		Keyboard json.RawMessage `json:"reply_markup"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, err
	}
	c.texts = append(c.texts, payload.Text)
	c.keyboards = append(c.keyboards, len(payload.Keyboard) > 0)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
}
func TestTelegramLongCardIsSentCompletely(t *testing.T) {
	t.Setenv("CRM_TELEGRAM_BOT_TOKEN", "test-only")
	capture := &telegramCaptureTransport{}
	previous := http.DefaultTransport
	http.DefaultTransport = capture
	t.Cleanup(func() { http.DefaultTransport = previous })
	text := strings.Repeat("🚘 полная заявка ", 500)
	if err := telegramKeyboard(context.Background(), "123", text, botMenu); err != nil {
		t.Fatal(err)
	}
	if strings.Join(capture.texts, "") != text {
		t.Fatal("message truncated")
	}
	for i, part := range capture.texts {
		if len([]rune(part)) > 2000 {
			t.Fatal("chunk too large")
		}
		if capture.keyboards[i] != (i == len(capture.texts)-1) {
			t.Fatal("keyboard must be on last chunk")
		}
	}
}
