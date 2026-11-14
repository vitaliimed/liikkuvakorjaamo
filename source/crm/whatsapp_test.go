package crm

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func waSignature(b []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(b)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
func waFixture(field, value string) []byte {
	return []byte(`{"object":"whatsapp_business_account","entry":[{"id":"account","changes":[{"field":"` + field + `","value":{"metadata":{"phone_number_id":"number"},` + value + `}}]}]}`)
}
func TestWASignatureAndRoles(t *testing.T) {
	b := []byte("payload")
	secret := "testsecret123456789"
	if !verifyWASignature(b, waSignature(b, secret), secret) || verifyWASignature([]byte("tampered"), waSignature(b, secret), secret) {
		t.Fatal("signature")
	}
	for _, method := range []string{"GET", "POST"} {
		w := httptest.NewRecorder()
		(&Server{}).whatsAppInbox(w, httptest.NewRequest(method, "/", nil), testMechanic)
		if w.Code != 403 {
			t.Fatal(w.Code)
		}
	}
}
func TestEmulatorWAReplayEchoStatusAndBinding(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	c := WhatsAppConfig{AppSecret: "testsecret123456789", WABAID: "account", PhoneNumberID: "number", BusinessNumber: "+358413606032"}
	ingest := func(b []byte) {
		t.Helper()
		if e := s.IngestWhatsApp(ctx, c, b, waSignature(b, c.AppSecret)); e != nil {
			t.Fatal(e)
		}
	}
	// Delivery can arrive before content and read must not regress to sent.
	ingest(waFixture("messages", `"statuses":[{"id":"message1","status":"read","recipient_id":"358401234567","timestamp":"1700000000"}]`))
	echo := waFixture("smb_message_echoes", `"message_echoes":[{"id":"message1","from":"358413606032","to":"358401234567","timestamp":"1700000000","type":"text","text":{"body":"Hei!"}}]`)
	ingest(echo)
	ingest(echo)
	id := digest("number:message1")
	d, e := s.Collection("whatsapp_inbox").Doc(id).Get(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var rec WARecord
	d.DataTo(&rec)
	if rec.Status != "read" || rec.Direction != "outgoing_app" || rec.JobID != "" {
		t.Fatalf("%+v", rec)
	}
	j := fixture(t)
	j.ID = randomID()
	j.Phone = "+358 40 1234567"
	s.Collection("jobs").Doc(j.ID).Set(ctx, j)
	body, _ := json.Marshal(map[string]string{"id": id, "job_id": j.ID})
	w := httptest.NewRecorder()
	s.whatsAppInbox(w, httptest.NewRequest("POST", "/", bytes.NewReader(body)), testOwner)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	ingest(waFixture("messages", `"statuses":[{"id":"message1","status":"sent","recipient_id":"358401234567","timestamp":"1700000001"}]`))
	md, e := s.Collection("jobs").Doc(j.ID).Collection("messages").Doc(id).Get(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var msg Message
	md.DataTo(&msg)
	if msg.Original != "Hei!" || msg.Status != "read" {
		t.Fatalf("%+v", msg)
	}
	c.WABAID = "other"
	if e = s.IngestWhatsApp(ctx, c, echo, waSignature(echo, c.AppSecret)); e != ErrForbidden {
		t.Fatal("wrong account", e)
	}
}
func TestEmulatorWAHistoryAttachmentAndChangedStatus(t *testing.T) {
	s := emulator(t)
	ctx := context.Background()
	c := WhatsAppConfig{AppSecret: "testsecret123456789", WABAID: "account", PhoneNumberID: "number", BusinessNumber: "358413606032"}
	b := waFixture("history", `"history":[{"threads":[{"id":"358401234567","messages":[{"id":"history1","from":"358413606032","to":"358401234567","timestamp":"1700000000","type":"image","image":{"id":"media1","caption":"Before repair","mime_type":"image/jpeg"}}]}]}]`)
	if e := s.IngestWhatsApp(ctx, c, b, waSignature(b, c.AppSecret)); e != nil {
		t.Fatal(e)
	}
	id := digest("number:history1")
	d, _ := s.Collection("whatsapp_inbox").Doc(id).Get(ctx)
	var rec WARecord
	d.DataTo(&rec)
	if rec.Direction != "outgoing_app" || rec.Message.Image.ID != "media1" {
		t.Fatalf("%+v", rec)
	}
	j := fixture(t)
	j.ID = randomID()
	j.Phone = "358401234567"
	s.Collection("jobs").Doc(j.ID).Set(ctx, j)
	body, _ := json.Marshal(map[string]string{"id": id, "job_id": j.ID})
	w := httptest.NewRecorder()
	s.whatsAppInbox(w, httptest.NewRequest("POST", "/", bytes.NewReader(body)), testOwner)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	b = waFixture("messages", `"statuses":[{"id":"history1","status":"delivered","timestamp":"1700000001"}]`)
	if e := s.IngestWhatsApp(ctx, c, b, waSignature(b, c.AppSecret)); e != nil {
		t.Fatal(e)
	}
	d, _ = s.Collection("jobs").Doc(j.ID).Collection("messages").Doc(id).Get(ctx)
	var m Message
	d.DataTo(&m)
	if m.Status != "delivered" || len(m.Attachments) != 1 || m.Original != "Before repair" {
		t.Fatalf("%+v", m)
	}
	j.ID = randomID()
	j.Phone = "358409999999"
	s.Collection("jobs").Doc(j.ID).Set(ctx, j)
	body, _ = json.Marshal(map[string]string{"id": id, "job_id": j.ID})
	w = httptest.NewRecorder()
	s.whatsAppInbox(w, httptest.NewRequest("POST", "/", bytes.NewReader(body)), testOwner)
	if w.Code != 409 {
		t.Fatal("rebound conversation", w.Code)
	}
}
