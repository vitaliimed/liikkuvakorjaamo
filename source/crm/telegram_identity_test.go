package crm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmulatorTelegramIdentityCannotBeShared(t *testing.T) {
	s := emulator(t)
	owner := User{ID: "owner", Role: "owner", Active: true}
	post := func(email, id string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"email": email, "name": "Mechanic", "password": "test-password-123", "role": "mechanic", "active": true, "telegram_id": id})
		r := httptest.NewRequest("POST", "/crm/api/users", bytes.NewReader(body))
		w := httptest.NewRecorder()
		s.users(w, r, owner)
		return w
	}
	if w := post("a@example.test", "123456"); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := post("b@example.test", "123456"); w.Code != 400 {
		t.Fatal("shared ID accepted", w.Code)
	}
	if w := post("a@example.test", "123456"); w.Code != 200 {
		t.Fatal("same user edit rejected", w.Body.String())
	}
	notices, err := s.Collection("outbox").Documents(context.Background()).GetAll()
	if err != nil || len(notices) != 1 {
		t.Fatalf("welcome must be queued once: %d, %v", len(notices), err)
	}
	var welcome Notice
	if err = notices[0].DataTo(&welcome); err != nil || !strings.Contains(welcome.Text, "Здравствуйте") || welcome.Keyboard == "" {
		t.Fatal("missing onboarding", err)
	}
	if w := post("c@example.test", "-100123"); w.Code != 400 {
		t.Fatal("group ID accepted")
	}
	t.Setenv("CRM_OWNER_TELEGRAM_ID", "987654")
	if w := post("d@example.test", "987654"); w.Code != 400 {
		t.Fatal("reserved owner ID accepted")
	}
}
