package crm

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const testWAVerify = "test-token-012345678901234567890123456789"
const testWASecret = "test-app-secret-with-sufficient-length"

func testWAGatewayConfig() WhatsAppConfig {
	return WhatsAppConfig{
		AppSecret: testWASecret, WABAID: "test-waba",
		PhoneNumberID: "test-phone", BusinessNumber: "+15551366024",
	}
}

func TestWhatsAppWebhookGETVerification(t *testing.T) {
	h := whatsappWebhookHTTP(testWAGatewayConfig(), testWAVerify, nil)
	query := url.Values{"hub.mode": {"subscribe"}, "hub.verify_token": {testWAVerify}, "hub.challenge": {"123abc"}}
	req := httptest.NewRequest(http.MethodGet, "/crm/webhooks/whatsapp?"+query.Encode(), nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != "123abc" {
		t.Fatalf("GET: status %d body %s", w.Code, w.Body.String())
	}
	query.Set("hub.verify_token", "incorrect")
	w = httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/crm/webhooks/whatsapp?"+query.Encode(), nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("invalid verify token: got %d", w.Code)
	}
}

func TestWhatsAppWebhookSignedPOST(t *testing.T) {
	raw := []byte(`{"object":"whatsapp_business_account","entry":[]}`)
	sum := hmac.New(sha256.New, []byte(testWASecret))
	sum.Write(raw)
	signature := "sha256=" + hex.EncodeToString(sum.Sum(nil))
	calls := 0
	h := whatsappWebhookHTTP(testWAGatewayConfig(), testWAVerify,
		func(_ context.Context, c WhatsAppConfig, body []byte, s string) error {
			calls++
			if c.PhoneNumberID != "test-phone" || !bytes.Equal(body, raw) || s != signature {
				t.Fatal("signed raw input was modified")
			}
			return nil
		})
	req := httptest.NewRequest(http.MethodPost, "/crm/webhooks/whatsapp", bytes.NewReader(raw))
	req.Header.Set("X-Hub-Signature-256", signature)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "EVENT_RECEIVED" || calls != 1 {
		t.Fatalf("valid signed message: status %d calls %d", w.Code, calls)
	}
	req = httptest.NewRequest(http.MethodPost, "/crm/webhooks/whatsapp", bytes.NewReader(raw))
	req.Header.Set("X-Hub-Signature-256", "sha256="+strings.Repeat("0", 64))
	w = httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusUnauthorized || calls != 1 {
		t.Fatalf("invalid signature: status %d calls %d", w.Code, calls)
	}
}

func TestWhatsAppWebhookOversizedPOST(t *testing.T) {
	h := whatsappWebhookHTTP(testWAGatewayConfig(), testWAVerify,
		func(context.Context, WhatsAppConfig, []byte, string) error {
			t.Fatal("Oversized input must not be ingested")
			return nil
		})
	req := httptest.NewRequest(http.MethodPost, "/crm/webhooks/whatsapp", strings.NewReader(strings.Repeat("x", 500001)))
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("Oversized input status %d", w.Code)
	}
}
