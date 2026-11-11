package main

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

func mailNotificationsEnabled(cfg Config) bool {
	return strings.TrimSpace(cfg.SMTPHost) != "" && strings.TrimSpace(cfg.SMTPPort) != "" && strings.TrimSpace(cfg.SMTPUsername) != "" && strings.TrimSpace(cfg.SMTPPassword) != "" && strings.TrimSpace(cfg.MailFrom) != ""
}

func liikkuvaEmailDefaults(s SiteSettings) SiteSettings {
	defaults := []struct {
		value    *string
		fallback string
	}{
		{&s.EmailLogoURL, "/static/img/icon-192x192.png"},
		{&s.EmailConfirmationSubjectFI, "Liikkuva: Kiitos yhteydenotostasi"},
		{&s.EmailConfirmationSubjectEN, "Liikkuva: Thank you for your request"},
		{&s.EmailConfirmationTextFI, "Kiitos pyynnöstäsi, {name}! Olemme vastaanottaneet yhteydenottosi.\n\nAsiakaspalvelumme soittaa sinulle vahvistaakseen tilanteen ja sopiakseen käynnistä. Odotathan yhteydenottoamme.\n\nAika ja työn sisältö vahvistetaan kanssasi ennen käyntiä."},
		{&s.EmailConfirmationTextEN, "Thank you for your request, {name}! We have received it successfully.\n\nOur service manager will call you to discuss the situation and arrange a visit. Please wait for our call.\n\nWe will confirm the appointment and scope of work with you before the visit."},
		{&s.EmailReadySubjectFI, "Liikkuva: Ajoneuvosi on valmis"},
		{&s.EmailReadySubjectEN, "Liikkuva: Your vehicle is ready"},
		{&s.EmailReadyTextFI, "Ajoneuvosi on valmis.\n\nVoit vastata tähän viestiin tai ottaa meihin yhteyttä sopiaksesi auton luovutuksesta ja työn yksityiskohdista. Kiitos, että valitsit Liikkuvan!"},
		{&s.EmailReadyTextEN, "Your vehicle is ready.\n\nReply to this email or contact us to arrange the handover and discuss the completed work. Thank you for choosing Liikkuva!"},
		{&s.EmailSignatureFI, "Ystävällisin terveisin,\nLiikkuva Auto Korjaamo"},
		{&s.EmailSignatureEN, "Best regards,\nLiikkuva Auto Korjaamo"},
	}
	for _, d := range defaults {
		if strings.TrimSpace(*d.value) == "" {
			*d.value = d.fallback
		}
	}
	return liikkuvaProDefaults(s)
}

// Read the current CMS once per email. No browser or third-party credentials are involved.
func withEmailSettings(cfg Config) Config {
	if cfg.EmailSettings != nil {
		return cfg
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, err := loadSiteSettings(ctx, cfg)
	if err != nil {
		s = SiteSettings{}
	}
	s = liikkuvaEmailDefaults(s)
	cfg.EmailSettings = &s
	return cfg
}
func emailSettings(cfg Config) SiteSettings {
	if cfg.EmailSettings != nil {
		return liikkuvaEmailDefaults(*cfg.EmailSettings)
	}
	return liikkuvaEmailDefaults(SiteSettings{})
}
func emailLanguage(lang string) string {
	if lang == "fi" || lang == "ru" {
		return lang
	}
	return "en"
}
func emailTemplateText(value string, lead Lead, s SiteSettings) string {
	return strings.NewReplacer("{name}", clientEmailName(lead), "{car}", lead.Car, "{visit}", fmt.Sprint(liikkuvaLeadFee(lead)), "{diagnostic}", fmt.Sprint(s.ProDiagnosticEUR)).Replace(value)
}
func emailSignature(cfg Config, lang string) string {
	s := emailSettings(cfg)
	if lang == "fi" {
		return s.EmailSignatureFI
	}
	if lang == "ru" {
		return "Liikkuva Auto Korjaamo"
	}
	return s.EmailSignatureEN
}
func emailGreeting(lead Lead) string {
	if lead.Language == "fi" {
		return "Hei " + clientEmailName(lead) + ","
	}
	return "Hello " + clientEmailName(lead) + ","
}
func emailSubject(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ")
}
func emailTextFooter(cfg Config, lang string) string {
	return "\n\n" + emailSignature(cfg, lang) + "\n" + serviceVisibleEmail(cfg) + "\n" + strings.TrimRight(cfg.SiteURL, "/")
}
func sendReadyForPickupEmail(cfg Config, lead Lead) error {
	if !mailNotificationsEnabled(cfg) {
		return fmt.Errorf("mail config is incomplete")
	}
	if strings.TrimSpace(lead.Email) == "" {
		return fmt.Errorf("client email is empty")
	}
	cfg = withEmailSettings(cfg)
	s := emailSettings(cfg)
	subject := s.EmailReadySubjectEN
	if lead.Language == "fi" {
		subject = s.EmailReadySubjectFI
	}
	return sendSMTPMail(cfg, lead.Email, emailSubject(emailTemplateText(subject, lead, s)), buildReadyEmailTextBody(cfg, lead), buildReadyEmailHTMLBody(cfg, lead))
}
func buildReadyEmailSubject(lang string, lead Lead) string {
	s := liikkuvaEmailDefaults(SiteSettings{})
	if lang == "fi" {
		return s.EmailReadySubjectFI
	}
	return s.EmailReadySubjectEN
}
func readyMessage(cfg Config, lead Lead) string {
	s := emailSettings(cfg)
	value := s.EmailReadyTextEN
	if lead.Language == "fi" {
		value = s.EmailReadyTextFI
	}
	return emailTemplateText(value, lead, s)
}
func buildReadyEmailTextBody(cfg Config, lead Lead) string {
	return emailGreeting(lead) + "\n\n" + readyMessage(cfg, lead) + plateText(lead) + emailTextFooter(cfg, lead.Language)
}
func buildReadyEmailHTMLBody(cfg Config, lead Lead) string {
	title := "Your vehicle is ready"
	if lead.Language == "fi" {
		title = "Ajoneuvosi on valmis"
	}
	return buildBrandedActionEmailHTML(cfg, lead, title, `<p style="margin:0;">`+nl2br(html.EscapeString(readyMessage(cfg, lead)))+`</p>`, "", "")
}
func sendSMTPMail(cfg Config, to, subject, textBody, htmlBody string, attachments ...EmailAttachment) error {
	return liikkuvaSendSMTP(cfg, to, subject, textBody, htmlBody, attachments...)
}
func sendCustomClientEmail(cfg Config, lead Lead, subject, message string, attachments ...EmailAttachment) error {
	if !mailNotificationsEnabled(cfg) {
		return fmt.Errorf("mail config is incomplete")
	}
	if err := validateStandaloneEmail(strings.TrimSpace(lead.Email), strings.TrimSpace(subject), strings.TrimSpace(message)); err != nil {
		return err
	}
	cfg = withEmailSettings(cfg)
	return sendSMTPMail(cfg, lead.Email, subject, buildCustomEmailTextBody(cfg, lead, message), buildCustomEmailHTMLBody(cfg, lead, subject, message), attachments...)
}
func buildCustomEmailTextBody(cfg Config, lead Lead, message string) string {
	return emailGreeting(lead) + "\n\n" + message + plateText(lead) + emailTextFooter(cfg, lead.Language)
}
func buildCustomEmailHTMLBody(cfg Config, lead Lead, subject, message string) string {
	return buildBrandedActionEmailHTML(cfg, lead, subject, `<div style="overflow-wrap:anywhere;">`+nl2br(html.EscapeString(message))+`</div>`, "", "")
}
func extractAddress(value string) string {
	parsed, err := mail.ParseAddress(strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	return parsed.Address
}
func serviceVisibleEmail(cfg Config) string {
	if cfg.EmailSettings != nil && strings.TrimSpace(cfg.EmailSettings.ServiceEmail) != "" {
		return extractAddress(cfg.EmailSettings.ServiceEmail)
	}
	return extractAddress(cfg.MailFrom)
}
func sendLeadConfirmationEmail(cfg Config, lead Lead) error {
	if !mailNotificationsEnabled(cfg) {
		return fmt.Errorf("mail config is incomplete")
	}
	if strings.TrimSpace(lead.Email) == "" {
		return fmt.Errorf("client email is empty")
	}
	cfg = withEmailSettings(cfg)
	s := emailSettings(cfg)
	subject := s.EmailConfirmationSubjectEN
	if lead.Language == "fi" {
		subject = s.EmailConfirmationSubjectFI
	}
	return sendSMTPMail(cfg, lead.Email, emailSubject(emailTemplateText(subject, lead, s)), buildLeadConfirmationText(cfg, lead), buildLeadConfirmationHTML(cfg, lead))
}
func buildLeadConfirmationSubject(lang string) string {
	s := liikkuvaEmailDefaults(SiteSettings{})
	if lang == "fi" {
		return s.EmailConfirmationSubjectFI
	}
	return s.EmailConfirmationSubjectEN
}
func confirmationMessage(cfg Config, lead Lead) string {
	s := emailSettings(cfg)
	value := s.EmailConfirmationTextEN
	if lead.Language == "fi" {
		value = s.EmailConfirmationTextFI
	}
	return emailTemplateText(value, lead, s)
}
func confirmationPrice(lead Lead) string {
	if lead.Language == "fi" {
		return fmt.Sprintf("Asentajan käynti %d €. Diagnostiikka, korjaukset ja varaosat sovitaan ja veloitetaan erikseen.", liikkuvaLeadFee(lead))
	}
	return fmt.Sprintf("Mechanic visit €%d. Diagnostics, repairs and spare parts are agreed and charged separately.", liikkuvaLeadFee(lead))
}
func buildLeadConfirmationText(cfg Config, lead Lead) string {
	return emailGreeting(lead) + "\n\n" + confirmationMessage(cfg, lead) + "\n\n" + confirmationPrice(lead) + plateText(lead) + emailTextFooter(cfg, lead.Language)
}
func buildLeadConfirmationHTML(cfg Config, lead Lead) string {
	title := "Thank you for your request"
	if lead.Language == "fi" {
		title = "Kiitos yhteydenotostasi"
	}
	content := `<div style="overflow-wrap:anywhere;">` + nl2br(html.EscapeString(confirmationMessage(cfg, lead))) + `</div><p style="background:#eef3ff;border-radius:12px;padding:18px;color:#183365;margin:24px 0 0;">` + html.EscapeString(confirmationPrice(lead)) + `</p>`
	return buildBrandedActionEmailHTML(cfg, lead, title, content, "", "")
}
func sendReviewInvitationEmail(cfg Config, lead Lead, reviewURL string) error {
	if !mailNotificationsEnabled(cfg) || lead.Email == "" || reviewURL == "" {
		return fmt.Errorf("mail config, recipient or review URL is empty")
	}
	cfg = withEmailSettings(cfg)
	subject, button, message := "Liikkuva: Tell us about your experience", "Leave a review", "Thank you for choosing Liikkuva. We would appreciate your feedback. Reviews are published only after approval."
	if lead.Language == "fi" {
		subject, button, message = "Liikkuva: Kerro kokemuksestasi", "Jätä arvostelu", "Kiitos, että valitsit Liikkuvan. Arvostaisimme palautettasi palvelustamme. Arvostelu julkaistaan vasta hyväksynnän jälkeen."
	}
	return sendSMTPMail(cfg, lead.Email, subject, emailGreeting(lead)+"\n\n"+message+"\n"+reviewURL+emailTextFooter(cfg, lead.Language), buildBrandedActionEmailHTML(cfg, lead, subject, `<p>`+html.EscapeString(message)+`</p>`, button, reviewURL))
}
func clientEmailName(lead Lead) string {
	if name := strings.TrimSpace(lead.Name); name != "" {
		return name
	}
	if lead.Language == "fi" {
		return "asiakas"
	}
	return "customer"
}
func plateText(lead Lead) string {
	if lead.Car == "" {
		return ""
	}
	label := "Vehicle registration"
	if lead.Language == "fi" {
		label = "Rekisterinumero"
	}
	return "\n\n" + label + ": " + lead.Car
}
func safeEmailImage(value, siteURL string) string {
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		value = strings.TrimRight(siteURL, "/") + value
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return ""
	}
	return value
}
func brandedEmailShell(cfg Config, lang, title, content, buttonLabel, buttonURL string) string {
	s := emailSettings(cfg)
	brand := s.ProAccent
	image := safeEmailImage(s.EmailLogoURL, cfg.SiteURL)
	logo := ""
	if image != "" {
		logo = `<img src="` + html.EscapeString(image) + `" width="48" height="48" alt="" style="display:block;width:48px;height:48px;object-fit:contain;background:#fff;border-radius:12px;">`
	}
	button := ""
	u, err := url.Parse(buttonURL)
	if buttonLabel != "" && err == nil && u.Scheme == "https" && u.Host != "" {
		button = `<p style="margin:24px 0;"><a href="` + html.EscapeString(buttonURL) + `" style="display:inline-block;padding:14px 22px;border-radius:9px;background:` + brand + `;color:#fff;font-weight:700;text-decoration:none;">` + html.EscapeString(buttonLabel) + `</a></p>`
	}
	contact := ""
	if email := serviceVisibleEmail(cfg); email != "" {
		contact = `<br><a style="color:#2458d6;" href="mailto:` + html.EscapeString(email) + `">` + html.EscapeString(email) + `</a>`
	}
	website := strings.TrimRight(cfg.SiteURL, "/")
	if website == "" {
		website = "https://liikkuvakorjaamo.fi"
	}
	return `<!doctype html><html lang="` + emailLanguage(lang) + `"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"><title>` + html.EscapeString(title) + ` — Liikkuva</title></head><body style="margin:0;background:#eef3fb;color:#10213a;font-family:Arial,Helvetica,sans-serif;"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#eef3fb;"><tr><td align="center" style="padding:28px 12px;"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:620px;background:#fff;border:1px solid #dbe3f1;border-radius:18px;overflow:hidden;"><tr><td style="padding:24px 28px;background:` + brand + `;color:#fff;"><table role="presentation" cellspacing="0" cellpadding="0"><tr><td style="padding-right:14px;">` + logo + `</td><td style="font-size:25px;line-height:1.1;font-weight:800;">Liikkuva<br><span style="font-size:12px;font-weight:600;letter-spacing:1px;">AUTO KORJAAMO</span></td></tr></table></td></tr><tr><td style="padding:30px 28px;font-size:16px;line-height:1.75;"><h1 style="font-size:28px;line-height:1.25;margin:0 0 24px;color:#10213a;">` + html.EscapeString(title) + `</h1>` + content + button + `<div style="margin-top:28px;padding-top:22px;border-top:1px solid #dbe3f1;color:#52657f;font-size:14px;">` + nl2br(html.EscapeString(emailSignature(cfg, lang))) + contact + `<br><a style="color:#2458d6;" href="` + html.EscapeString(website) + `">liikkuvakorjaamo.fi</a></div></td></tr></table></td></tr></table></body></html>`
}
func brandedPlainEmailHTML(cfg Config, lang, subject, message string) string {
	return brandedEmailShell(cfg, lang, subject, `<div style="overflow-wrap:anywhere;">`+nl2br(html.EscapeString(message))+`</div>`, "", "")
}
func buildBrandedActionEmailHTML(cfg Config, lead Lead, title, contentHTML, buttonLabel, buttonURL string) string {
	return brandedEmailShell(cfg, lead.Language, title, `<p style="margin:0 0 20px;">`+html.EscapeString(emailGreeting(lead))+`</p>`+optionalPlateBlockHTML(lead.Car, lead.Language)+contentHTML, buttonLabel, buttonURL)
}
func optionalPlateBlockHTML(value, lang string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	label := "Vehicle registration"
	if lang == "fi" {
		label = "Rekisterinumero"
	}
	return `<div style="background:#eef3ff;border-radius:10px;padding:12px 16px;margin:0 0 20px;font-size:14px;">` + label + `: <strong>` + html.EscapeString(value) + `</strong></div>`
}
func nl2br(value string) string { return strings.ReplaceAll(value, "\n", "<br>") }

func (h *AdminHandler) EmailPreview(w http.ResponseWriter, r *http.Request) {
	if !isAdminAuthenticated(h.cfg, r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", 405)
		return
	}
	s, err := loadSiteSettings(r.Context(), h.cfg)
	if err != nil {
		http.Error(w, "Не удалось загрузить настройки", 503)
		return
	}
	s = liikkuvaEmailDefaults(s)
	cfg := h.cfg
	cfg.EmailSettings = &s
	lang := sanitizeLanguage(r.URL.Query().Get("lang"), "fi")
	lead := Lead{Name: "Asiakas", Language: lang, Car: "TEST-123", VisitPriceEUR: liikkuvaVisitFee(s)}
	body := buildLeadConfirmationHTML(cfg, lead)
	if r.URL.Query().Get("kind") == "ready" {
		body = buildReadyEmailHTMLBody(cfg, lead)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' https:; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'self'")
	io.WriteString(w, body)
}
