package crm

import (
	"bytes"
	"cloud.google.com/go/firestore"
	"context"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"html"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"time"
)

type Email struct {
	ResetURL  string    `json:"-"`
	Language  string    `json:"language,omitempty"`
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	Parent    string    `json:"parent"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Direction string    `json:"direction"`
	Status    string    `json:"status"`
	Actor     string    `json:"actor"`
	Created   time.Time `json:"created"`
	Error     string    `json:"error"`
}
type mailIndex struct {
	Job   string
	Email string
}

func cleanHeader(s string) bool { return !strings.ContainsAny(s, "\r\n\x00") && len(s) < 998 }
func validAddress(s string) bool {
	a, e := mail.ParseAddress(s)
	return e == nil && a.Address == s && cleanHeader(s)
}
func envValue(k, f string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return f
}
func mailConfig() (string, string, string, string, error) {
	from := os.Getenv("CRM_MAIL_FROM")
	if from == "" {
		from = os.Getenv("MAIL_FROM")
	}
	a, e := mail.ParseAddress(from)
	if e != nil || !cleanHeader(from) {
		return "", "", "", "", errors.New("Настройте CRM_MAIL_FROM")
	}
	reply := envValue("CRM_MAIL_REPLY_TO", envValue("MAIL_REPLY_TO", a.Address))
	if !validAddress(reply) {
		return "", "", "", "", errors.New("Некорректный Reply-To")
	}
	host := os.Getenv("SMTP_HOST")
	port := envValue("SMTP_PORT", "587")
	if host == "" || (port != "465" && port != "587") {
		return "", "", "", "", errors.New("Настройте SMTP_HOST и SMTP_PORT (465/587)")
	}
	return a.Address, reply, host, port, nil
}
func renderMail(e Email, reply string) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "From: %s\r\nTo: %s\r\nReply-To: %s\r\nDate: %s\r\nMessage-ID: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\n", e.From, e.To, reply, e.Created.Format(time.RFC1123Z), e.MessageID, mime.QEncoding.Encode("utf-8", e.Subject))
	if e.Parent != "" {
		fmt.Fprintf(&out, "In-Reply-To: %s\r\nReferences: %s\r\n", e.Parent, e.Parent)
	}
	var body bytes.Buffer
	m := multipart.NewWriter(&body)
	fmt.Fprintf(&out, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", m.Boundary())
	for _, p := range []struct{ kind, text string }{{"text/plain", e.Body}, {"text/html", brandedEmailHTML(e)}} {
		w, _ := m.CreatePart(textproto.MIMEHeader{"Content-Type": {p.kind + "; charset=utf-8"}, "Content-Transfer-Encoding": {"base64"}})
		data := base64.StdEncoding.EncodeToString([]byte(p.text))
		for len(data) > 76 {
			fmt.Fprint(w, data[:76]+"\r\n")
			data = data[76:]
		}
		fmt.Fprint(w, data+"\r\n")
	}
	_ = m.Close()
	out.Write(body.Bytes())
	return out.Bytes()
}

// accepted means the SMTP server accepted the message, not delivered to an inbox.
func smtpSend(e Email) (string, error) {
	_, reply, host, port, err := mailConfig()
	if err != nil {
		return "failed", err
	}
	addr := net.JoinHostPort(host, port)
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	var conn net.Conn
	config := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if port == "465" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, config)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return "failed", errors.New("Нет соединения с SMTP")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(40 * time.Second))
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return "failed", errors.New("Ошибка SMTP handshake")
	}
	defer c.Close()
	if port == "587" {
		if err = c.StartTLS(config); err != nil {
			return "failed", errors.New("SMTP требует защищённое STARTTLS соединение")
		}
	}
	username, password := os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD")
	if username == "" || password == "" {
		return "failed", errors.New("Не настроены SMTP_USERNAME/SMTP_PASSWORD")
	}
	if err = c.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
		return "failed", errors.New("SMTP отклонил авторизацию")
	}
	if err = c.Mail(e.From); err != nil {
		return "failed", errors.New("SMTP отклонил отправителя")
	}
	if err = c.Rcpt(e.To); err != nil {
		return "failed", errors.New("SMTP отклонил адрес получателя")
	}
	w, err := c.Data()
	if err != nil {
		return "failed", errors.New("SMTP не принял письмо")
	}
	if _, err = w.Write(renderMail(e, reply)); err != nil {
		return "unknown", errors.New("Сбой передачи: проверьте SMTP-журнал до повторной отправки")
	}
	if err = w.Close(); err != nil {
		return "unknown", errors.New("Нет однозначного подтверждения SMTP: проверьте журнал перед повтором")
	}
	_ = c.Quit()
	return "accepted", nil
}
func (s *Server) emails(w http.ResponseWriter, r *http.Request, u User, j Job) {
	if !u.Manager() {
		fail(w, 403, "Переписка доступна только владельцу")
		return
	}
	if r.Method == "POST" && os.Getenv("CRM_MAIL_SEND_ENABLED") != "1" {
		fail(w, 503, "Отправка писем отключена для этой ревизии")
		return
	}
	ref := s.Collection("jobs").Doc(j.ID)
	if r.Method == "GET" {
		q := ref.Collection("emails").OrderBy("Created", firestore.Asc).OrderBy(firestore.DocumentID, firestore.Asc)
		if c := r.URL.Query().Get("cursor"); c != "" {
			d, e := ref.Collection("emails").Doc(c).Get(r.Context())
			if e != nil {
				fail(w, 400, "Некорректный курсор")
				return
			}
			q = q.StartAfter(d)
		}
		ds, e := q.Limit(101).Documents(r.Context()).GetAll()
		if e != nil {
			fail(w, 503, "Не удалось прочитать письма")
			return
		}
		next := ""
		if len(ds) > 100 {
			ds = ds[:100]
			next = ds[99].Ref.ID
		}
		out := []Email{}
		for _, d := range ds {
			var m Email
			if d.DataTo(&m) != nil {
				fail(w, 503, "Ошибка письма")
				return
			}
			out = append(out, m)
		}
		write(w, 200, map[string]any{"emails": out, "cursor": next})
		return
	}
	if r.Method != "POST" || !u.Manager() {
		fail(w, 403, "Письма отправляет владелец или менеджер")
		return
	}
	var x struct {
		Revision int64  `json:"revision"`
		Subject  string `json:"subject"`
		Body     string `json:"body"`
		Parent   string `json:"parent"`
	}
	if err := decode(w, r, &x); err != nil {
		fail(w, 400, err.Error())
		return
	}
	if !cleanHeader(x.Subject) || strings.TrimSpace(x.Subject) == "" || strings.TrimSpace(x.Body) == "" || len(x.Body) > 40000 {
		fail(w, 400, "Укажите тему и текст письма до 40 КБ")
		return
	}
	from, _, _, _, err := mailConfig()
	if err != nil {
		fail(w, 503, err.Error())
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if len(key) < 16 || len(key) > 128 {
		fail(w, 400, "Нужен Idempotency-Key")
		return
	}
	id := digest(j.ID + ":" + key)
	mr := ref.Collection("emails").Doc(id)
	var m Email
	created := false
	err = s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
		created = false
		existing, e := tx.Get(mr)
		if e == nil {
			return existing.DataTo(&m)
		}
		if status.Code(e) != codes.NotFound {
			return e
		}
		d, e := tx.Get(ref)
		if e != nil {
			return e
		}
		var live Job
		if e = d.DataTo(&live); e != nil {
			return e
		}
		if live.Revision != x.Revision {
			return ErrConflict
		}
		if !validAddress(live.Email) {
			return errors.New("Добавьте корректный email клиента в карточку заявки")
		}
		if x.Parent != "" {
			ix, e := tx.Get(s.Collection("mail_index").Doc(digest(x.Parent)))
			if e != nil {
				return errors.New("Родительское письмо не найдено")
			}
			var link mailIndex
			if ix.DataTo(&link) != nil || link.Job != j.ID {
				return ErrForbidden
			}
		}
		m = Email{Language: live.Language, ID: id, MessageID: "<" + id + "@" + strings.Split(from, "@")[1] + ">", Parent: x.Parent, From: from, To: live.Email, Subject: x.Subject, Body: x.Body, Direction: "outgoing", Status: "sending", Actor: u.ID, Created: time.Now().UTC()}
		if e = tx.Create(mr, m); e != nil {
			return e
		}
		if e = tx.Create(s.Collection("mail_index").Doc(digest(m.MessageID)), mailIndex{Job: j.ID, Email: id}); e != nil {
			return e
		}
		if e = tx.Create(ref.Collection("messages").Doc("email-"+id), emailMessage(live, m)); e != nil {
			return e
		}
		created = true
		return nil
	})
	if err != nil {
		fail(w, 409, err.Error())
		return
	}
	if !created {
		write(w, 200, m)
		return
	}
	sendStatus, sendErr := smtpSend(m)
	m.Status = sendStatus
	if sendErr != nil {
		m.Error = sendErr.Error()
	}
	// Complete only these fields; no automatic retry after an ambiguous network result.
	batch := s.DB.Batch()
	batch.Update(mr, []firestore.Update{{Path: "Status", Value: m.Status}, {Path: "Error", Value: m.Error}})
	batch.Update(ref.Collection("messages").Doc("email-"+id), []firestore.Update{{Path: "Status", Value: m.Status}})
	_, err = batch.Commit(r.Context())
	if err != nil {
		fail(w, 503, "Результат отправки не удалось записать. Не отправляйте повторно до проверки цепочки и SMTP-журнала")
		return
	}
	write(w, 200, m)
}
func (s *Server) inboundMail(w http.ResponseWriter, r *http.Request) {
	expected := os.Getenv("CRM_MAIL_INBOUND_TOKEN")
	given := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if r.Method != "POST" || len(expected) < 32 || subtle.ConstantTimeCompare([]byte(expected), []byte(given)) != 1 {
		fail(w, 403, "Нет доступа")
		return
	}
	var x struct {
		MessageID string `json:"message_id"`
		InReplyTo string `json:"in_reply_to"`
		From      string `json:"from"`
		To        string `json:"to"`
		Subject   string `json:"subject"`
		Body      string `json:"body"`
	}
	if e := decode(w, r, &x); e != nil {
		fail(w, 400, e.Error())
		return
	}
	if !validAddress(x.From) || !validAddress(x.To) || !cleanHeader(x.Subject) || !cleanHeader(x.MessageID) || !strings.HasPrefix(x.MessageID, "<") || !strings.HasSuffix(x.MessageID, ">") || x.InReplyTo == "" || len(x.Body) > 40000 {
		fail(w, 400, "Некорректное входящее письмо")
		return
	}
	_, reply, _, _, e := mailConfig()
	if e != nil || !strings.EqualFold(x.To, reply) {
		fail(w, 400, "Неверный адрес входящей почты")
		return
	}
	d, e := s.Collection("mail_index").Doc(digest(x.InReplyTo)).Get(r.Context())
	if e != nil {
		fail(w, 422, "Не найдена цепочка. Сохраните письмо для ручного разбора")
		return
	}
	var link mailIndex
	if d.DataTo(&link) != nil {
		fail(w, 503, "Ошибка цепочки")
		return
	}
	id := digest(x.MessageID)
	ref := s.Collection("jobs").Doc(link.Job)
	m := Email{ID: id, MessageID: x.MessageID, Parent: x.InReplyTo, From: x.From, To: x.To, Subject: x.Subject, Body: x.Body, Direction: "incoming", Status: "received", Actor: "mail-connector", Created: time.Now().UTC()}
	e = s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
		existing, err := tx.Get(s.Collection("mail_index").Doc(id))
		if err == nil {
			var prev mailIndex
			if existing.DataTo(&prev) != nil || prev.Job != link.Job {
				return errors.New("Конфликт Message-ID")
			}
			return nil
		}
		if status.Code(err) != codes.NotFound {
			return err
		}
		d, err := tx.Get(ref)
		if err != nil {
			return err
		}
		var j Job
		if d.DataTo(&j) != nil {
			return errors.New("Ошибка заявки")
		}
		if !strings.EqualFold(j.Email, x.From) {
			return errors.New("Отправитель отличается от email клиента; требуется ручной разбор")
		}
		if err = tx.Create(ref.Collection("emails").Doc(id), m); err != nil {
			return err
		}
		if err = tx.Create(ref.Collection("messages").Doc("email-"+id), emailMessage(j, m)); err != nil {
			return err
		}
		return tx.Create(s.Collection("mail_index").Doc(id), mailIndex{Job: link.Job, Email: id})
	})
	if e != nil {
		fail(w, 422, e.Error())
		return
	}
	write(w, 200, map[string]bool{"ok": true})
}

// Free-form subject/body are preserved; the document language follows the job.
func brandedEmailHTML(e Email) string {
	lang := "fi"
	if e.Language == "en" {
		lang = "en"
	} else if e.Language == "ru" {
		lang = "ru"
	}
	resetAction := ""
	if e.ResetURL != "" {
		u, err := url.Parse(e.ResetURL)
		if err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil &&
			strings.HasPrefix(u.Path, "/crm/") && strings.HasPrefix(u.Fragment, "reset=") {
			resetAction = `<p style="margin:24px 0"><a style="display:inline-block;background:#f97315;color:#23372b;padding:14px 20px;text-decoration:none;font-weight:bold;border-radius:8px" href="` + html.EscapeString(e.ResetURL) + `">Установить новый пароль</a></p>`
		}
	}
	return `<!doctype html><html lang="` + lang + `"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head><body style="margin:0;background:#f4f5f2;font:16px/1.7 Arial,sans-serif;color:#23372b"><table role="presentation" width="100%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:24px 12px"><table role="presentation" width="100%" style="max-width:640px;background:#fff" cellspacing="0" cellpadding="0"><tr><td style="padding:24px;background:#23372b;border-bottom:4px solid #f97315"><img src="https://liikkuvakorjaamo.fi/static/img/logo-green.png?v=20261112" alt="Liikkuva auto korjaamo" width="280" style="display:block;max-width:100%;height:auto"></td></tr><tr><td style="padding:24px"><h1 style="font-size:24px">` + html.EscapeString(e.Subject) + `</h1><div style="white-space:pre-wrap">` + html.EscapeString(e.Body) + `</div>` + resetAction + `</td></tr><tr><td style="padding:20px 24px;background:#23372b;color:#fff">Liikkuva auto korjaamo</td></tr></table></td></tr></table></body></html>`
}
