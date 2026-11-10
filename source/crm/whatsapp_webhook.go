package crm

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "crypto/subtle"
    "encoding/hex"
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "cloud.google.com/go/firestore"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// This endpoint accepts Meta-signed webhooks and safely stores unassigned messages.
// An unrecognized phone number MUST NOT automatically create a car or a customer job.
// Manual/verified linkage and sending are separate follow-on stages.
const whatsappBodyLimit int64 = 1 << 20

type waEnvelope struct {
    Object string `json:"object"`
    Entry []struct {
        Changes []struct {
            Field string `json:"field"`
            Value waValue `json:"value"`
        } `json:"changes"`
    } `json:"entry"`
}
type waValue struct {
    Metadata struct {
        PhoneNumberID string `json:"phone_number_id"`
        DisplayPhoneNumber string `json:"display_phone_number"`
    } `json:"metadata"`
    Contacts []struct {
        WAID string `json:"wa_id"`
        Profile struct {
            Name string `json:"name"`
        } `json:"profile"`
    } `json:"contacts"`
    Messages []waMessage `json:"messages"`
    Statuses []struct {
        ID string `json:"id"`
        Status string `json:"status"`
        Timestamp string `json:"timestamp"`
        RecipientID string `json:"recipient_id"`
    } `json:"statuses"`
}
type waMessage struct {
    ID string `json:"id"`
    From string `json:"from"`
    Timestamp string `json:"timestamp"`
    Type string `json:"type"`
    Text struct {
        Body string `json:"body"`
    } `json:"text"`
    Image struct {
        ID string `json:"id"`
        Caption string `json:"caption"`
    } `json:"image"`
    Document struct {
        ID string `json:"id"`
        Caption string `json:"caption"`
    } `json:"document"`
    Audio struct {
        ID string `json:"id"`
    } `json:"audio"`
    Video struct {
        ID string `json:"id"`
        Caption string `json:"caption"`
    } `json:"video"`
    Button struct {
        Text string `json:"text"`
        Payload string `json:"payload"`
    } `json:"button"`
    Interactive struct {
        Type string `json:"type"`
        ButtonReply struct {
            ID string `json:"id"`
            Title string `json:"title"`
        } `json:"button_reply"`
        ListReply struct {
            ID string `json:"id"`
            Title string `json:"title"`
        } `json:"list_reply"`
    } `json:"interactive"`
}

type waInboxItem struct {
    ID string `json:"id"`
    ExternalID string `json:"external_id"`
    Sender string `json:"sender"`
    ContactName string `json:"contact_name"`
    PhoneNumberID string `json:"phone_number_id"`
    Type string `json:"type"`
    Text string `json:"text"`
    MediaID string `json:"media_id"`
    ButtonID string `json:"button_id"`
    MessageTime time.Time `json:"message_time"`
    ReceivedAt time.Time `json:"received_at"`
    State string `json:"state"`
}
type waDelivery struct {
    ExternalID string `json:"external_id"`
    Recipient string `json:"recipient"`
    PhoneNumberID string `json:"phone_number_id"`
    State string `json:"state"`
    MessageTime time.Time `json:"message_time"`
    ReceivedAt time.Time `json:"received_at"`
}

func metaEventTime(raw string) time.Time {
    seconds, err := strconv.ParseInt(raw, 10, 64)
    if err != nil || seconds < 1 || seconds > 4102444800 {
        return time.Now().UTC()
    }
    return time.Unix(seconds, 0).UTC()
}

func metaMessageContent(m waMessage) (text, mediaID, buttonID string) {
    switch m.Type {
    case "text":
        text = m.Text.Body
    case "image":
        text, mediaID = m.Image.Caption, m.Image.ID
    case "document":
        text, mediaID = m.Document.Caption, m.Document.ID
    case "audio":
        mediaID = m.Audio.ID
    case "video":
        text, mediaID = m.Video.Caption, m.Video.ID
    case "button":
        text, buttonID = m.Button.Text, m.Button.Payload
    case "interactive":
        if m.Interactive.Type == "button_reply" {
            text, buttonID = m.Interactive.ButtonReply.Title, m.Interactive.ButtonReply.ID
        } else if m.Interactive.Type == "list_reply" {
            text, buttonID = m.Interactive.ListReply.Title, m.Interactive.ListReply.ID
        }
    }
    return
}

func validateMetaSignature(body []byte, signature, secret string) bool {
    if secret == "" || !strings.HasPrefix(signature, "sha256=") {
        return false
    }
    raw, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
    if err != nil || len(raw) != sha256.Size {
        return false
    }
    mac := hmac.New(sha256.New, []byte(secret))
    _, _ = mac.Write(body)
    return hmac.Equal(raw, mac.Sum(nil))
}

// webhookHandler isolates HTTP and signature validation so it can be tested
// without contacting Firestore or Meta.
func whatsappWebhookHandler(store func(context.Context, waEnvelope) error) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Cache-Control", "no-store")
        w.Header().Set("X-Robots-Tag", "noindex, nofollow")
        switch r.Method {
        case http.MethodGet:
            token := os.Getenv("WHATSAPP_VERIFY_TOKEN")
            if token == "" {
                http.Error(w, "Not configured", http.StatusServiceUnavailable)
                return
            }
            q := r.URL.Query()
            supplied := q.Get("hub.verify_token")
            if q.Get("hub.mode") != "subscribe" || supplied == "" ||
                subtle.ConstantTimeCompare([]byte(token), []byte(supplied)) != 1 {
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
            secret := os.Getenv("WHATSAPP_APP_SECRET")
            if secret == "" {
                http.Error(w, "Not configured", http.StatusServiceUnavailable)
                return
            }
            r.Body = http.MaxBytesReader(w, r.Body, whatsappBodyLimit)
            body, err := io.ReadAll(r.Body)
            if err != nil {
                var large *http.MaxBytesError
                if errors.As(err, &large) {
                    http.Error(w, "Too large", http.StatusRequestEntityTooLarge)
                } else {
                    http.Error(w, "Invalid request", http.StatusBadRequest)
                }
                return
            }
            if !validateMetaSignature(body, r.Header.Get("X-Hub-Signature-256"), secret) {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            var payload waEnvelope
            if json.Unmarshal(body, &payload) != nil || payload.Object != "whatsapp_business_account" {
                http.Error(w, "Invalid Meta event", http.StatusBadRequest)
                return
            }
            if store == nil || store(r.Context(), payload) != nil {
                // Return non-2xx to allow Meta to retry transient storage errors.
                // Never log the payload or private customer message content.
                http.Error(w, "Temporary storage failure", http.StatusServiceUnavailable)
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
    whatsappWebhookHandler(s.persistWhatsApp).ServeHTTP(w, r)
}

func (s *Server) persistWhatsApp(ctx context.Context, envelope waEnvelope) error {
    for _, entry := range envelope.Entry {
        for _, change := range entry.Changes {
            if change.Field != "messages" {
                continue
            }
            value := change.Value
            names := make(map[string]string, len(value.Contacts))
            for _, c := range value.Contacts {
                names[c.WAID] = c.Profile.Name
            }
            for _, m := range value.Messages {
                if m.ID == "" || m.From == "" || len(m.ID) > 512 {
                    continue
                }
                text, mediaID, buttonID := metaMessageContent(m)
                item := waInboxItem{
                    ID: digest("wa:inbox:" + m.ID),
                    ExternalID: m.ID,
                    Sender: m.From,
                    ContactName: names[m.From],
                    PhoneNumberID: value.Metadata.PhoneNumberID,
                    Type: m.Type,
                    Text: text,
                    MediaID: mediaID,
                    ButtonID: buttonID,
                    MessageTime: metaEventTime(m.Timestamp),
                    ReceivedAt: time.Now().UTC(),
                    State: "unassigned",
                }
                _, err := s.Collection("whatsapp_inbox").Doc(item.ID).Create(ctx, item)
                if err != nil && status.Code(err) != codes.AlreadyExists {
                    return err
                }
            }
            for _, st := range value.Statuses {
                if st.ID == "" || st.Status == "" {
                    continue
                }
                id := digest("wa:status:" + st.ID + ":" + st.Status + ":" + st.Timestamp)
                delivery := waDelivery{
                    ExternalID: st.ID,
                    Recipient: st.RecipientID,
                    PhoneNumberID: value.Metadata.PhoneNumberID,
                    State: st.Status,
                    MessageTime: metaEventTime(st.Timestamp),
                    ReceivedAt: time.Now().UTC(),
                }
                _, err := s.Collection("whatsapp_status_events").Doc(id).Create(ctx, delivery)
                if err != nil && status.Code(err) != codes.AlreadyExists {
                    return err
                }
            }
        }
    }
    return nil
}

// An owner-only read-only queue for unknown/new WhatsApp contacts.
// Automatic merging by plate or guessed phone is intentionally forbidden.
func (s *Server) whatsappInbox(w http.ResponseWriter, r *http.Request, u User) {
    if !u.Manager() {
        fail(w, http.StatusForbidden, "Только владелец видит переписку")
        return
    }
    if r.Method != http.MethodGet {
        fail(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }
    docs, err := s.Collection("whatsapp_inbox").OrderBy("ReceivedAt", firestore.Desc).Limit(50).Documents(r.Context()).GetAll()
    if err != nil {
        fail(w, http.StatusServiceUnavailable, "Не удалось прочитать WhatsApp")
        return
    }
    items := make([]waInboxItem, 0, len(docs))
    for _, d := range docs {
        var item waInboxItem
        if d.DataTo(&item) == nil {
            items = append(items, item)
        }
    }
    write(w, http.StatusOK, map[string]any{"items": items})
}
