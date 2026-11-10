package crm

// The existing WhatsApp gateway must invoke this adapter after receiving the
// original Meta payload. It does not register a second public webhook or bot.
import (
	"cloud.google.com/go/firestore"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type WhatsAppConfig struct{ AppSecret, PhoneNumberID, WABAID, BusinessNumber string }
type WAMedia struct {
	ID       string `json:"id"`
	MIME     string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	Filename string `json:"filename"`
	Caption  string `json:"caption"`
}
type WAMessage struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      struct {
		Body string `json:"body"`
	} `json:"text"`
	Image    WAMedia `json:"image"`
	Video    WAMedia `json:"video"`
	Audio    WAMedia `json:"audio"`
	Document WAMedia `json:"document"`
	Sticker  WAMedia `json:"sticker"`
}
type WARecord struct {
	ID            string    `json:"id"`
	ExternalID    string    `json:"external_id"`
	PhoneNumberID string    `json:"phone_number_id"`
	Peer          string    `json:"peer"`
	Message       WAMessage `json:"message"`
	Direction     string    `json:"direction"`
	Status        string    `json:"status"`
	At            time.Time `json:"at"`
	JobID         string    `json:"job_id"`
	Translation   string    `json:"translation"`
	BoundBy       string    `json:"bound_by"`
	BoundAt       time.Time `json:"bound_at"`
}
type waEnvelope struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				Metadata struct {
					PhoneNumberID string `json:"phone_number_id"`
				} `json:"metadata"`
				History []struct {
					Threads []struct {
						ID       string      `json:"id"`
						Messages []WAMessage `json:"messages"`
					} `json:"threads"`
				} `json:"history"`
				Messages []WAMessage `json:"messages"`
				Echoes   []WAMessage `json:"message_echoes"`
				Statuses []struct {
					ID        string `json:"id"`
					Status    string `json:"status"`
					Recipient string `json:"recipient_id"`
					Timestamp string `json:"timestamp"`
				} `json:"statuses"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

var waPhone = regexp.MustCompile(`^[1-9][0-9]{6,14}$`)

func normalizedWAPhone(s string) string {
	r := strings.NewReplacer("+", "", " ", "", "-", "", "(", "", ")", "").Replace(s)
	if !waPhone.MatchString(r) {
		return ""
	}
	return r
}
func verifyWASignature(body []byte, signature, secret string) bool {
	if len(secret) < 16 || len(body) > 2<<20 || !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	supplied, e := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if e != nil {
		return false
	}
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return hmac.Equal(supplied, h.Sum(nil))
}
func waRank(s string) int {
	switch s {
	case "sent":
		return 1
	case "delivered":
		return 2
	case "read":
		return 3
	}
	return 0
}
func (s *Server) IngestWhatsApp(ctx context.Context, c WhatsAppConfig, body []byte, signature string) error {
	if !verifyWASignature(body, signature, c.AppSecret) {
		return ErrForbidden
	}
	if c.PhoneNumberID == "" || c.WABAID == "" || normalizedWAPhone(c.BusinessNumber) == "" {
		return errors.New("WhatsApp account not configured")
	}
	var payload waEnvelope
	if json.Unmarshal(body, &payload) != nil || payload.Object != "whatsapp_business_account" {
		return errors.New("Invalid WhatsApp envelope")
	}
	// Validate ALL accounts before any writes. Raw envelope retained privately for
	// unsupported message types, delivery errors, history and future reprocessing.
	for _, entry := range payload.Entry {
		if entry.ID != c.WABAID {
			return ErrForbidden
		}
		for _, change := range entry.Changes {
			if change.Value.Metadata.PhoneNumberID != c.PhoneNumberID {
				return ErrForbidden
			}
		}
	}
	// Raw payload can exceed Firestore's document size. Reject rather than truncate.
	if len(body) > 500000 {
		return errors.New("WhatsApp payload requires gateway chunking before ingestion")
	}
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			messages := change.Value.Messages
			if change.Field == "smb_message_echoes" {
				messages = change.Value.Echoes
			}
			if change.Field == "history" {
				for _, h := range change.Value.History {
					for _, thread := range h.Threads {
						messages = append(messages, thread.Messages...)
					}
				}
			}
			if change.Field == "messages" || change.Field == "smb_message_echoes" || change.Field == "history" {
				for _, m := range messages {
					peer := normalizedWAPhone(m.From)
					direction := "incoming"
					if change.Field == "smb_message_echoes" || (change.Field == "history" && normalizedWAPhone(m.From) == normalizedWAPhone(c.BusinessNumber)) {
						if normalizedWAPhone(m.From) != normalizedWAPhone(c.BusinessNumber) {
							return ErrForbidden
						}
						peer = normalizedWAPhone(m.To)
						direction = "outgoing_app"
					}
					if peer == "" || m.ID == "" || len(m.ID) > 512 {
						return errors.New("Invalid WhatsApp message")
					}
					stamp, e := strconv.ParseInt(m.Timestamp, 10, 64)
					if e != nil || stamp <= 0 {
						return errors.New("Invalid timestamp")
					}
					id := digest(c.PhoneNumberID + ":" + m.ID)
					ref := s.Collection("whatsapp_inbox").Doc(id)
					e = s.DB.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
						rec := WARecord{ID: id, ExternalID: m.ID, PhoneNumberID: c.PhoneNumberID, Peer: peer, Message: m, Direction: direction, At: time.Unix(stamp, 0).UTC(), Status: "received"}
						d, e := tx.Get(ref)
						if e == nil {
							var old WARecord
							if e = d.DataTo(&old); e != nil {
								return e
							}
							if old.Message.ID != "" {
								return nil
							}
							rec.Status = old.Status
						} else if status.Code(e) != codes.NotFound {
							return e
						}
						return tx.Set(ref, rec)
					})
					if e != nil {
						return e
					}
				}
			}
			for _, st := range change.Value.Statuses {
				if st.ID == "" || len(st.ID) > 512 {
					return errors.New("Missing status message ID")
				}
				switch st.Status {
				case "sent", "delivered", "read", "failed":
				default:
					continue
				}
				ref := s.Collection("whatsapp_inbox").Doc(digest(c.PhoneNumberID + ":" + st.ID))
				e := s.DB.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
					rec := WARecord{ID: ref.ID, ExternalID: st.ID, PhoneNumberID: c.PhoneNumberID, Peer: normalizedWAPhone(st.Recipient)}
					d, e := tx.Get(ref)
					if e == nil {
						if e = d.DataTo(&rec); e != nil {
							return e
						}
					} else if status.Code(e) != codes.NotFound {
						return e
					}
					if waRank(st.Status) > waRank(rec.Status) || (st.Status == "failed" && waRank(rec.Status) < 2) {
						rec.Status = st.Status
					}
					if rec.JobID != "" {
						if e := tx.Update(s.Collection("jobs").Doc(rec.JobID).Collection("messages").Doc(rec.ID), []firestore.Update{{Path: "Status", Value: rec.Status}}); e != nil {
							return e
						}
					}
					return tx.Set(ref, rec)
				})
				if e != nil {
					return e
				}
			}
		}
	}
	_, e := s.Collection("whatsapp_events").Doc(digest(string(body))).Create(ctx, map[string]any{"body": string(body), "received": time.Now().UTC()})
	if status.Code(e) == codes.AlreadyExists {
		return nil
	}
	return e
}
func waOriginal(m WAMessage) string {
	if m.Type == "text" {
		return m.Text.Body
	}
	for _, a := range []WAMedia{m.Image, m.Video, m.Audio, m.Document, m.Sticker} {
		if a.ID != "" {
			return a.Caption
		}
	}
	return "[" + m.Type + "]"
}
func (s *Server) whatsAppInbox(w http.ResponseWriter, r *http.Request, u User) {
	if !u.Manager() {
		fail(w, 403, "Переписка доступна только владельцу")
		return
	}
	if r.Method == "GET" {
		q := s.Collection("whatsapp_inbox").OrderBy(firestore.DocumentID, firestore.Asc)
		if cursor := r.URL.Query().Get("cursor"); cursor != "" {
			if !ledgerKey.MatchString(cursor) {
				fail(w, 400, "Некорректный курсор")
				return
			}
			q = q.StartAfter(cursor)
		}
		ds, e := q.Limit(101).Documents(r.Context()).GetAll()
		if e != nil {
			fail(w, 503, "Ошибка WhatsApp")
			return
		}
		next := ""
		if len(ds) > 100 {
			ds = ds[:100]
			next = ds[99].Ref.ID
		}
		out := []WARecord{}
		for _, d := range ds {
			var rec WARecord
			if d.DataTo(&rec) != nil {
				fail(w, 503, "Ошибка сообщения")
				return
			}
			out = append(out, rec)
		}
		write(w, 200, map[string]any{"messages": out, "cursor": next})
		return
	}
	if r.Method != "POST" {
		fail(w, 405, "Method not allowed")
		return
	}
	var req struct {
		ID    string `json:"id"`
		JobID string `json:"job_id"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil || len(req.ID) != 64 || len(req.JobID) != 64 || !ledgerKey.MatchString(req.ID) || !ledgerKey.MatchString(req.JobID) {
		fail(w, 400, "Некорректная привязка")
		return
	}
	e := s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
		ref := s.Collection("whatsapp_inbox").Doc(req.ID)
		d, e := tx.Get(ref)
		if e != nil {
			return e
		}
		var rec WARecord
		if e = d.DataTo(&rec); e != nil {
			return e
		}
		if rec.JobID != "" && rec.JobID != req.JobID {
			return ErrConflict
		}
		if rec.Message.ID == "" {
			return errors.New("Текст сообщения ещё не получен")
		}
		jd, e := tx.Get(s.Collection("jobs").Doc(req.JobID))
		if e != nil {
			return e
		}
		var j Job
		if e = jd.DataTo(&j); e != nil {
			return e
		}
		if rec.Peer == "" || normalizedWAPhone(j.Phone) != rec.Peer {
			return errors.New("Телефон клиента заказа не совпадает; проверьте контакт")
		}
		mr := jd.Ref.Collection("messages").Doc(rec.ID)
		md, e := tx.Get(mr)
		if e == nil {
			_ = md
			return nil
		}
		if status.Code(e) != codes.NotFound {
			return e
		}
		rec.JobID = j.ID
		rec.BoundBy = u.ID
		rec.BoundAt = time.Now().UTC()
		attachments := []string{}
		for _, a := range []WAMedia{rec.Message.Image, rec.Message.Video, rec.Message.Audio, rec.Message.Document, rec.Message.Sticker} {
			if a.ID != "" {
				attachments = append(attachments, "whatsapp-media:"+a.ID)
			}
		}
		m := Message{ID: rec.ID, JobID: j.ID, VehicleID: vehicleID(j), Source: "whatsapp", Author: rec.Peer, Language: j.Language, At: rec.At, Original: waOriginal(rec.Message), Direction: rec.Direction, Status: rec.Status, Attachments: attachments}
		if e = tx.Create(mr, m); e != nil {
			return e
		}
		return tx.Set(ref, rec)
	})
	if e != nil {
		fail(w, 409, e.Error())
		return
	}
	write(w, 200, map[string]bool{"ok": true})
}
