package crm

import (
	"context"
	"crypto/subtle"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
)

// whatsappWebhookHTTP connects the public Meta callback to the existing
// IngestWhatsApp adapter. The raw signed POST body is never re-encoded.
func whatsappWebhookHTTP(config WhatsAppConfig, verifyToken string, persist func(context.Context, WhatsAppConfig, []byte, string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		switch r.Method {
		case http.MethodGet:
			if len(verifyToken) < 32 {
				http.Error(w, "Webhook not configured", http.StatusServiceUnavailable)
				return
			}
			q := r.URL.Query()
			token := q.Get("hub.verify_token")
			if q.Get("hub.mode") != "subscribe" || token == "" ||
				subtle.ConstantTimeCompare([]byte(token), []byte(verifyToken)) != 1 {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			challenge := q.Get("hub.challenge")
			if challenge == "" || len(challenge) > 256 {
				http.Error(w, "Invalid challenge", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, challenge)
		case http.MethodPost:
			if len(config.AppSecret) < 16 || config.WABAID == "" || config.PhoneNumberID == "" ||
				normalizedWAPhone(config.BusinessNumber) == "" || persist == nil {
				http.Error(w, "Webhook not configured", http.StatusServiceUnavailable)
				return
			}
			// The current IngestWhatsApp persists the raw signed payload in one
			// Firestore document, and explicitly rejects payloads over 500 KB.
			// Do not acknowledge an oversized history event as processed.
			r.Body = http.MaxBytesReader(w, r.Body, 500000)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				var large *http.MaxBytesError
				if errors.As(err, &large) {
					http.Error(w, "Payload too large", http.StatusRequestEntityTooLarge)
					return
				}
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			signature := r.Header.Get("X-Hub-Signature-256")
			if !verifyWASignature(body, signature, config.AppSecret) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if err := persist(r.Context(), config, body, signature); err != nil {
				if errors.Is(err, ErrForbidden) {
					http.Error(w, "Forbidden", http.StatusForbidden)
				} else {
					// Non-2xx permits provider retry after storage failures.
					http.Error(w, "Temporary ingestion failure", http.StatusServiceUnavailable)
				}
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, "EVENT_RECEIVED")
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func (s *Server) whatsappWebhook(w http.ResponseWriter, r *http.Request) {
	config := WhatsAppConfig{
		AppSecret:      strings.TrimSpace(os.Getenv("WHATSAPP_APP_SECRET")),
		WABAID:         strings.TrimSpace(os.Getenv("WHATSAPP_WABA_ID")),
		PhoneNumberID:  strings.TrimSpace(os.Getenv("WHATSAPP_PHONE_NUMBER_ID")),
		BusinessNumber: strings.TrimSpace(os.Getenv("WHATSAPP_BUSINESS_NUMBER")),
	}
	whatsappWebhookHTTP(config, os.Getenv("WHATSAPP_VERIFY_TOKEN"), s.IngestWhatsApp).ServeHTTP(w, r)
}
