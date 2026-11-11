package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

const standaloneEmailRoute = "/admin/email"

type standaloneEmailData struct {
	Token, AccessCSRF, To, Subject, Message, Error, From, Language string
	Sent, Enabled                                                  bool
}
type standaloneEmailSender func(Config, string, string, string, string, ...EmailAttachment) error

// Completely separate from leads: no Firestore, Sheets, Telegram or client status changes.
func (h *AdminHandler) StandaloneEmail(w http.ResponseWriter, r *http.Request) {
	h.standaloneEmailWithSender(w, r, sendSMTPMail)
}
func standaloneEmailToken(secret string) (string, error) {
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	body := strconv.FormatInt(time.Now().Add(2*time.Hour).Unix(), 10) + "." + hex.EncodeToString(nonce[:])
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("standalone-email:" + body))
	return body + "." + hex.EncodeToString(mac.Sum(nil)), nil
}
func validStandaloneEmailToken(secret, token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	expires, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || expires <= time.Now().Unix() || expires > time.Now().Add(2*time.Hour+time.Minute).Unix() {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("standalone-email:" + parts[0] + "." + parts[1]))
	signature, err := hex.DecodeString(parts[2])
	return err == nil && hmac.Equal(signature, mac.Sum(nil))
}
func validateStandaloneEmail(to, subject, message string) error {
	if len(to) > 254 || strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("Укажите один корректный email получателя.")
	}
	a, err := mail.ParseAddress(to)
	if err != nil || a.Address != to || !strings.Contains(a.Address, "@") {
		return fmt.Errorf("Укажите один email без имени, запятых и пробелов вокруг адреса.")
	}
	if subject == "" || len([]rune(subject)) > 200 || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("Тема обязательна: до 200 символов, одна строка.")
	}
	if message == "" || len([]rune(message)) > 20000 {
		return fmt.Errorf("Текст обязателен: до 20 000 символов.")
	}
	return nil
}
func standaloneEmailHTML(message string) string { return standaloneEmailHTMLLanguage(message, "ru") }
func standaloneEmailHTMLLanguage(message, lang string) string {
	if lang != "fi" && lang != "en" && lang != "ru" {
		lang = "fi"
	}
	return `<!doctype html><html lang="` + lang + `"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head><body style="margin:0;background:#f5f6f2;font:16px/1.7 Arial,sans-serif;color:#23372b"><table role="presentation" width="100%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:24px 12px"><table role="presentation" width="100%" style="max-width:640px;background:#fff;border:1px solid #dce5de" cellspacing="0" cellpadding="0"><tr><td style="padding:24px;background:#23372b;border-bottom:4px solid #f97315"><img src="https://liikkuvakorjaamo.fi/static/img/logo-green.png?v=20261111" width="280" alt="Liikkuva auto korjaamo" style="display:block;width:280px;max-width:100%;height:auto"></td></tr><tr><td style="padding:28px 24px;white-space:pre-wrap">` + html.EscapeString(message) + `</td></tr><tr><td style="padding:20px 24px;background:#23372b;color:#fff">Liikkuva auto korjaamo</td></tr></table></td></tr></table></body></html>`
}

func (h *AdminHandler) standaloneEmailWithSender(w http.ResponseWriter, r *http.Request, send standaloneEmailSender) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	if !isAdminAuthenticated(h.cfg, r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data := standaloneEmailData{Language: "fi", AccessCSRF: accessCSRF(r), From: h.cfg.MailFrom, Enabled: mailNotificationsEnabled(h.cfg)}
	render := func(status int) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		_ = standaloneEmailTemplate.Execute(w, data)
	}
	if r.Method == http.MethodGet {
		token, err := standaloneEmailToken(sessionToken(r))
		if err != nil {
			http.Error(w, "Cannot open email form", http.StatusInternalServerError)
			return
		}
		data.Token = token
		data.Sent = r.URL.Query().Get("sent") == "1"
		render(http.StatusOK)
		return
	}
	// Authenticated __session and a signed, expiring form token are required. Origin is checked by middleware.
	r.Body = http.MaxBytesReader(w, r.Body, maxEmailAttachmentBytes+(1<<20))
	err := r.ParseMultipartForm(2 << 20)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err != nil {
		data.Error = "Не удалось прочитать форму. Общий размер вложений — до 15 МБ. Откройте форму заново."
		render(http.StatusBadRequest)
		return
	}
	data.Language = sanitizeLanguage(r.FormValue("email_language"), "fi")
	data.To = strings.TrimSpace(r.PostFormValue("email_to"))
	data.Subject = strings.TrimSpace(r.PostFormValue("email_subject"))
	data.Message = strings.TrimSpace(r.PostFormValue("email_message"))
	data.Token = r.PostFormValue("csrf_token")
	if !validStandaloneEmailToken(sessionToken(r), data.Token) {
		data.Token = ""
		data.Error = "Срок действия формы истёк или форма открыта в другой вкладке. Скопируйте текст и откройте новую форму."
		render(http.StatusForbidden)
		return
	}
	if err := validateStandaloneEmail(data.To, data.Subject, data.Message); err != nil {
		data.Error = err.Error()
		render(http.StatusBadRequest)
		return
	}
	attachments, err := readEmailAttachments(r)
	if err != nil {
		data.Error = err.Error() + ". Выберите вложения заново."
		render(http.StatusBadRequest)
		return
	}
	if !data.Enabled {
		data.Error = "Отправка почты не настроена. Проверьте SMTP и MAIL_FROM."
		render(http.StatusServiceUnavailable)
		return
	}
	if err := send(h.cfg, data.To, data.Subject, data.Message, standaloneEmailHTMLLanguage(data.Message, data.Language), attachments...); err != nil {
		// A lost SMTP response can be ambiguous: never auto-retry and risk duplicate mail.
		data.Error = "Не удалось подтвердить отправку. Проверьте журнал почтового сервиса перед повтором. Текст сохранён в форме; вложения нужно выбрать заново."
		render(http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, standaloneEmailRoute+"?sent=1", http.StatusSeeOther)
}

var standaloneEmailTemplate = template.Must(template.New("standalone-email").Parse(`<!doctype html>
<html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Новое письмо — Liikkuva Auto Korjaamo</title>
<style>*{box-sizing:border-box}body{margin:0;background:#f5f6f2;color:#23372b;font:16px/1.5 Arial,sans-serif}main{max-width:820px;margin:32px auto;padding:28px;background:white;border-radius:18px}a{color:#f97315}h1{margin:12px 0}label{display:block;margin-top:18px;font-weight:bold}input,textarea,select{width:100%;padding:12px;border:1px solid #b8c6d6;border-radius:8px;font:inherit}textarea{min-height:260px}button{margin-top:22px;padding:14px 24px;border:0;border-radius:8px;background:#23372b;color:white;font:inherit;cursor:pointer}button:disabled{opacity:.5;cursor:wait}.note{color:#536779;font-size:14px}.ok,.err{padding:14px;border-radius:8px}.ok{background:#f4f5f2}.err{background:#f4f5f2}small{display:block}@media(max-width:600px){main{margin:0;padding:20px;border-radius:0}}</style></head>
<body><main><a href="/admin">← Админка</a><h1>Новое письмо</h1><p>Любой получатель, своя тема и текст. Заявка не требуется.</p><p class="note">Отправитель: {{.From}}</p>
{{if .Sent}}<p class="ok" role="status">Почтовый сервер принял письмо. Доставка во входящие зависит от почтового сервиса получателя.</p>{{end}}
{{if .Error}}<p class="err" role="alert">{{.Error}}</p>{{end}}{{if not .Enabled}}<p class="err">Отправка отключена: настройки почты не заполнены.</p>{{end}}
<form method="post" action="/admin/email" enctype="multipart/form-data" id="compose">
<input type="hidden" name="csrf_token" value="{{.Token}}"><input type="hidden" name="access_csrf" value="{{.AccessCSRF}}">
<label for="to">Кому</label><input id="to" name="email_to" type="email" maxlength="254" required value="{{.To}}" placeholder="name@example.com">
<label for="language">Язык оформления письма</label><select id="language" name="email_language"><option value="fi" {{if eq .Language "fi"}}selected{{end}}>Suomi</option><option value="en" {{if eq .Language "en"}}selected{{end}}>English</option></select><p class="note">Текст письма вводится на выбранном языке; автоматического перевода нет.</p><label for="subject">Тема</label><input id="subject" name="email_subject" maxlength="200" required value="{{.Subject}}">
<label for="message">Текст письма</label><textarea id="message" name="email_message" maxlength="20000" required>{{.Message}}</textarea>
<p class="note">Укажите приветствие и подпись в тексте. Автоматические фразы о заявке или автомобиле не добавляются.</p>
<label for="files">Вложения</label><input id="files" type="file" name="custom_email_attachments" multiple><small class="note">До 10 файлов, суммарно до 15 МБ.</small>
<button type="submit" {{if or (not .Enabled) (not .Token)}}disabled{{end}}>Отправить письмо</button><p><a href="/admin/email">Открыть новую форму</a></p></form>
<script>document.getElementById('compose').addEventListener('submit',function(e){const f=document.getElementById('files').files;if(f.length>10||Array.from(f).reduce((n,x)=>n+x.size,0)>15*1024*1024){e.preventDefault();alert('До 10 файлов, суммарно до 15 МБ.');return;}const b=this.querySelector('button');b.disabled=true;b.textContent='Отправка…';});</script>
</main></body></html>`))
