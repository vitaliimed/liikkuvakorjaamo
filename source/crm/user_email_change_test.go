package crm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEmulatorEmployeeEmailChangeKeepsIdentity(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	post := func(id, email string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"id": id, "email": email, "name": "Mechanic", "password": "test-password-123", "role": "mechanic", "active": true, "telegram_id": "123456"})
		w := httptest.NewRecorder()
		s.users(w, httptest.NewRequest("POST", "/crm/api/users", bytes.NewReader(body)), testOwner)
		return w
	}
	if w := post("", "old@example.test"); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	id := digest("old@example.test")
	if w := post(id, "new@example.test"); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	d, e := s.userByEmail(ctx, "new@example.test")
	if e != nil || d.Ref.ID != id {
		t.Fatal("identity changed", e)
	}
	if _, e = s.userByEmail(ctx, "old@example.test"); e == nil {
		t.Fatal("old email still logs in")
	}
	body := bytes.NewBufferString(`{"email":"new@example.test","password":"test-password-123"}`)
	w := httptest.NewRecorder()
	s.login(w, httptest.NewRequest("POST", "/", body))
	if w.Code != 200 {
		t.Fatal("new login failed", w.Body.String())
	}
	var u User
	d.DataTo(&u)
	token := randomID()
	s.Collection("password_resets").Doc(digest(token)).Set(ctx, passwordResetRecord{User: id, Email: u.Email, Auth: digest(u.Hash), Expires: time.Now().Add(time.Hour)})
	if _, _, e = s.validPasswordReset(ctx, token); e != nil {
		t.Fatal("reset after rename failed", e)
	}
	if w := post("", "old@example.test"); w.Code == 200 {
		t.Fatal("old hash identity overwritten")
	}
	notices, e := s.Collection("outbox").Documents(ctx).GetAll()
	if e != nil || len(notices) != 1 {
		t.Fatal("greeting duplicated", len(notices), e)
	}
}
