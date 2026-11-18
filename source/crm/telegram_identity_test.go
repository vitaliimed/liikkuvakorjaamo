package crm

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
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
	if w := post("c@example.test", "-100123"); w.Code != 400 {
		t.Fatal("group ID accepted")
	}
	t.Setenv("CRM_OWNER_TELEGRAM_ID", "987654")
	if w := post("d@example.test", "987654"); w.Code != 400 {
		t.Fatal("reserved owner ID accepted")
	}
}
