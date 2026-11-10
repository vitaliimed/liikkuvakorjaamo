package crm

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "errors"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func metaSignedRequest(body, secret string) *http.Request {
    req := httptest.NewRequest(http.MethodPost, "/crm/webhooks/whatsapp", strings.NewReader(body))
    mac := hmac.New(sha256.New, []byte(secret))
    _, _ = mac.Write([]byte(body))
    req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
    return req
}

func TestWhatsAppVerifyTokenHandshake(t *testing.T) {
    t.Setenv("WHATSAPP_VERIFY_TOKEN", "long-random-meta-verify-token")
    handler := whatsappWebhookHandler(nil)
    req := httptest.NewRequest("GET",
        "/crm/webhooks/whatsapp?hub.mode=subscribe&hub.challenge=123456&hub.verify_token=long-random-meta-verify-token", nil)
    result := httptest.NewRecorder()
    handler(result, req)
    if result.Code != 200 || result.Body.String() != "123456" {
        t.Fatalf("handshake got %d %q", result.Code, result.Body.String())
    }
    for _, query := range []string{
        "hub.mode=subscribe&hub.challenge=123456&hub.verify_token=wrong",
        "hub.mode=bad&hub.challenge=123456&hub.verify_token=long-random-meta-verify-token",
    } {
        rr := httptest.NewRecorder()
        handler(rr, httptest.NewRequest("GET", "/crm/webhooks/whatsapp?"+query, nil))
        if rr.Code != 403 {
            t.Fatalf("bad challenge returned %d", rr.Code)
        }
    }
}

func TestWhatsAppSignedMessageAndInvalidSignature(t *testing.T) {
    t.Setenv("WHATSAPP_APP_SECRET", "test-meta-app-secret")
    body := `{"object":"whatsapp_business_account","entry":[{"changes":[{"field":"messages","value":{"metadata":{"phone_number_id":"123"},"messages":[{"id":"wamid.TEST","from":"358401234567","type":"text","text":{"body":"Hei!"},"timestamp":"1794312000"}]}}]}]}`
    called := 0
    handler := whatsappWebhookHandler(func(_ context.Context, event waEnvelope) error {
        called++
        if len(event.Entry) != 1 || len(event.Entry[0].Changes) != 1 {
            t.Fatal("unexpected payload")
        }
        return nil
    })
    good := httptest.NewRecorder()
    handler(good, metaSignedRequest(body, "test-meta-app-secret"))
    if good.Code != 200 || called != 1 {
        t.Fatalf("valid webhook got %d, called %d", good.Code, called)
    }
    bad := httptest.NewRecorder()
    handler(bad, metaSignedRequest(body, "not-the-correct-secret"))
    if bad.Code != 401 || called != 1 {
        t.Fatalf("invalid HMAC got %d, called %d", bad.Code, called)
    }
    noSig := httptest.NewRecorder()
    handler(noSig, httptest.NewRequest("POST", "/crm/webhooks/whatsapp", strings.NewReader(body)))
    if noSig.Code != 401 || called != 1 {
        t.Fatalf("missing signature got %d, called %d", noSig.Code, called)
    }
}

func TestWhatsAppInvalidAndStorageFailure(t *testing.T) {
    t.Setenv("WHATSAPP_APP_SECRET", "test-meta-app-secret")
    badBody := `{"object":"not-whatsapp","entry":[]}`
    rr := httptest.NewRecorder()
    whatsappWebhookHandler(func(context.Context, waEnvelope) error { t.Fatal("store called"); return nil })(
        rr, metaSignedRequest(badBody, "test-meta-app-secret"))
    if rr.Code != 400 {
        t.Fatalf("bad payload got %d", rr.Code)
    }
    goodBody := `{"object":"whatsapp_business_account","entry":[]}`
    rr = httptest.NewRecorder()
    whatsappWebhookHandler(func(context.Context, waEnvelope) error { return errors.New("outage") })(
        rr, metaSignedRequest(goodBody, "test-meta-app-secret"))
    if rr.Code != 503 {
        t.Fatalf("retryable error got %d", rr.Code)
    }
}

func TestWhatsAppButtonParsing(t *testing.T) {
    var m waMessage
    if err := json.Unmarshal([]byte(`{"type":"interactive","interactive":{"type":"button_reply","button_reply":{"id":"approve:123","title":"Подтвердить"}}}`), &m); err != nil {
        t.Fatal(err)
    }
    text, media, button := metaMessageContent(m)
    if text != "Подтвердить" || media != "" || button != "approve:123" {
        t.Fatalf("bad interactive parsing: %s %s %s", text, media, button)
    }
}
