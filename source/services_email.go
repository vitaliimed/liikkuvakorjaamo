package main

import (
	"errors"
	"fmt"
	"html"

	"net/mail"

	"strings"
)

func mailNotificationsEnabled(cfg Config) bool {
	return strings.TrimSpace(cfg.SMTPHost) != "" &&
		strings.TrimSpace(cfg.SMTPPort) != "" &&
		strings.TrimSpace(cfg.SMTPUsername) != "" &&
		strings.TrimSpace(cfg.SMTPPassword) != "" &&
		strings.TrimSpace(cfg.MailFrom) != ""
}

func sendReadyForPickupEmail(cfg Config, lead Lead) error {
	if !mailNotificationsEnabled(cfg) {
		return fmt.Errorf("mail config is incomplete")
	}

	to := strings.TrimSpace(lead.Email)
	if to == "" {
		return fmt.Errorf("client email is empty")
	}

	subject := buildReadyEmailSubject(lead.Language, lead)
	textBody := buildReadyEmailTextBody(cfg, lead)
	htmlBody := buildReadyEmailHTMLBody(cfg, lead)

	return sendSMTPMail(cfg, to, subject, textBody, htmlBody)
}

func buildReadyEmailSubject(lang string, lead Lead) string {
	car := strings.TrimSpace(lead.Car)

	if strings.EqualFold(strings.TrimSpace(lang), "fi") {
		if car != "" {
			return "🚗 Liikkuva Auto Korjaamo: Ajoneuvo " + car + " on valmis"
		}
		return "🚗 Liikkuva Auto Korjaamo: Ajoneuvosi on valmis"
	}

	if car != "" {
		return "🚗 Liikkuva Auto Korjaamo: Vehicle " + car + " is ready"
	}
	return "🚗 Liikkuva Auto Korjaamo: Your vehicle is ready"
}

func buildReadyEmailTextBody(cfg Config, lead Lead) string {
	name := strings.TrimSpace(lead.Name)
	if name == "" {
		if strings.EqualFold(strings.TrimSpace(lead.Language), "fi") {
			name = "asiakas"
		} else {
			name = "customer"
		}
	}

	car := strings.TrimSpace(lead.Car)
	serviceEmail := serviceVisibleEmail(cfg)
	website := strings.TrimRight(strings.TrimSpace(cfg.SiteURL), "/")

	if strings.EqualFold(strings.TrimSpace(lead.Language), "fi") {
		lines := []string{
			"Hei " + name + ",",
			"",
			"✅ Ajoneuvosi on valmis.",
		}

		if car != "" {
			lines = append(lines, "🚘 Rekisterinumero: "+car)
		}

		lines = append(lines,
			"",
			"Voit vastata tähän viestiin tai ottaa meihin yhteyttä sopiaksesi luovutuksesta.",
		)

		if serviceEmail != "" {
			lines = append(lines, "📧 Sähköposti: "+serviceEmail)
		}
		if website != "" {
			lines = append(lines, "🌐 Verkkosivusto: "+website)
		}

		lines = append(lines,
			"",
			"Ystävällisin terveisin,",
			"Liikkuva Auto Korjaamo",
		)

		return strings.Join(lines, "\n")
	}

	lines := []string{
		"Hello " + name + ",",
		"",
		"✅ Your vehicle is ready.",
	}

	if car != "" {
		lines = append(lines, "🚘 Vehicle registration: "+car)
	}

	lines = append(lines,
		"",
		"You can reply to this email or contact us to arrange pickup or handover.",
	)

	if serviceEmail != "" {
		lines = append(lines, "📧 Service email: "+serviceEmail)
	}
	if website != "" {
		lines = append(lines, "🌐 Website: "+website)
	}

	lines = append(lines,
		"",
		"Best regards,",
		"Liikkuva Auto Korjaamo",
	)

	return strings.Join(lines, "\n")
}

func buildReadyEmailHTMLBody(cfg Config, lead Lead) string {
	title := "Your vehicle is ready"
	message := "Your vehicle is ready. You can reply to this email to arrange pickup or handover."
	if strings.EqualFold(strings.TrimSpace(lead.Language), "fi") {
		title = "Ajoneuvosi on valmis"
		message = "Ajoneuvosi on valmis. Voit vastata tähän viestiin sopiaksesi luovutuksesta."
	}
	return buildBrandedActionEmailHTML(cfg, lead, title, "<p>"+html.EscapeString(message)+"</p>", "", "")
}

func optionalInfoRowHTML(icon, label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	escapedLabel := html.EscapeString(label)
	escapedValue := html.EscapeString(value)

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return `<div style="font-size:15px;line-height:1.8;margin:6px 0;"><span style="margin-right:8px;">` + icon + `</span><strong>` + escapedLabel + `:</strong> <a href="` + escapedValue + `" style="color:#23372b;text-decoration:none;">` + escapedValue + `</a></div>`
	}

	if strings.Contains(value, "@") {
		return `<div style="font-size:15px;line-height:1.8;margin:6px 0;"><span style="margin-right:8px;">` + icon + `</span><strong>` + escapedLabel + `:</strong> <a href="mailto:` + escapedValue + `" style="color:#23372b;text-decoration:none;">` + escapedValue + `</a></div>`
	}

	return `<div style="font-size:15px;line-height:1.8;margin:6px 0;"><span style="margin-right:8px;">` + icon + `</span><strong>` + escapedLabel + `:</strong> ` + escapedValue + `</div>`
}

func optionalPlateRowHTML(value string, lang string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	label := "Vehicle registration"
	if strings.EqualFold(strings.TrimSpace(lang), "fi") {
		label = "Rekisterinumero"
	}

	return `<tr>
		<td style="padding:0 0 14px 0;">
			<div style="background:#f4f5f2;border:1px solid #e5e9e6;border-radius:14px;padding:14px 16px;">
				<div style="font-size:13px;line-height:1.4;color:#23372b;font-weight:700;text-transform:uppercase;letter-spacing:.04em;">🚘 ` + html.EscapeString(label) + `</div>
				<div style="font-size:22px;line-height:1.3;color:#0f172a;font-weight:800;margin-top:6px;">` + html.EscapeString(value) + `</div>
			</div>
		</td>
	</tr>`
}

// isSMTPIPAuthorizationDenied recognizes Brevo's permanent SMTP key IP
// authorization rejection. It is not a connectivity/transient/delivery error.
// Keep the original failure in Firestore for manual retry once IP policy is fixed.
func isSMTPIPAuthorizationDenied(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "525") &&
		strings.Contains(message, "unauthorized ip address")
}

func sendSMTPMail(cfg Config, to, subject, textBody, htmlBody string, attachments ...EmailAttachment) error {
	return liikkuvaSendSMTP(cfg, to, subject, textBody, htmlBody, attachments...)
}

func sendCustomClientEmail(cfg Config, lead Lead, subject, message string, attachments ...EmailAttachment) error {
	if !mailNotificationsEnabled(cfg) {
		return fmt.Errorf("mail config is incomplete")
	}

	to := strings.TrimSpace(lead.Email)
	if to == "" {
		return fmt.Errorf("client email is empty")
	}

	subject = strings.TrimSpace(subject)
	message = strings.TrimSpace(message)

	if subject == "" {
		return fmt.Errorf("subject is empty")
	}
	if message == "" {
		return fmt.Errorf("message is empty")
	}

	textBody := buildCustomEmailTextBody(cfg, lead, message)
	htmlBody := buildCustomEmailHTMLBody(cfg, lead, subject, message)

	return sendSMTPMail(cfg, to, subject, textBody, htmlBody, attachments...)
}

func buildCustomEmailTextBody(cfg Config, lead Lead, message string) string {
	name := strings.TrimSpace(lead.Name)
	if name == "" {
		if strings.EqualFold(strings.TrimSpace(lead.Language), "fi") {
			name = "asiakas"
		} else {
			name = "customer"
		}
	}

	car := strings.TrimSpace(lead.Car)
	serviceEmail := serviceVisibleEmail(cfg)
	website := strings.TrimRight(strings.TrimSpace(cfg.SiteURL), "/")

	greeting, plateLabel, emailLabel, websiteLabel, closing := "Hello", "Vehicle registration", "Service email", "Website", "Best regards"
	if strings.EqualFold(strings.TrimSpace(lead.Language), "fi") {
		greeting, plateLabel, emailLabel, websiteLabel, closing = "Hei", "Rekisterinumero", "Sähköposti", "Verkkosivusto", "Ystävällisin terveisin"
	}
	lines := []string{
		greeting + " " + name + ",",
		"",
		message,
	}

	if car != "" {
		lines = append(lines, "", plateLabel+": "+car)
	}

	if serviceEmail != "" {
		lines = append(lines, "", emailLabel+": "+serviceEmail)
	}
	if website != "" {
		lines = append(lines, websiteLabel+": "+website)
	}

	lines = append(lines, "", closing+",", "Liikkuva Auto Korjaamo")
	return strings.Join(lines, "\n")
}

func extractAddress(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	parsed, err := mail.ParseAddress(value)
	if err == nil && parsed != nil {
		return strings.TrimSpace(parsed.Address)
	}

	if strings.Contains(value, "@") && !strings.ContainsAny(value, " <>") {
		return value
	}

	return ""
}

func serviceVisibleEmail(cfg Config) string {
	return extractAddress(cfg.MailFrom)
}

func buildCustomEmailHTMLBody(cfg Config, lead Lead, subject, message string) string {
	return buildBrandedActionEmailHTML(cfg, lead, subject, "<p>"+nl2br(html.EscapeString(message))+"</p>", "", "")
}

var errLeadPriceSnapshotMissing = errors.New("saved visit/diagnostic price is missing")

func sendLeadConfirmationEmail(cfg Config, lead Lead) error {
	if !mailNotificationsEnabled(cfg) {
		return fmt.Errorf("mail config is incomplete")
	}
	if strings.TrimSpace(lead.Email) == "" {
		return fmt.Errorf("client email is empty")
	}
	// Prices are snapshotted when the form is submitted. Never substitute
	// compiled-in defaults for a missing snapshot: those may be stale after
	// the owner edits the price in the admin panel.
	if lead.VisitPriceEUR < 1 || lead.DiagnosticFromEUR < 1 {
		return fmt.Errorf("confirmation email not sent: %w; check the lead and pricing settings", errLeadPriceSnapshotMissing)
	}

	return sendSMTPMail(
		cfg,
		lead.Email,
		buildLeadConfirmationSubject(lead.Language),
		buildLeadConfirmationText(cfg, lead),
		buildLeadConfirmationHTML(cfg, lead),
	)
}

func buildLeadConfirmationSubject(lang string) string {
	if strings.EqualFold(strings.TrimSpace(lang), "fi") {
		return "Liikkuva Auto Korjaamo: Pyyntö vastaanotettu"
	}
	return "Liikkuva Auto Korjaamo: Request received"
}

func buildLeadConfirmationText(cfg Config, lead Lead) string {
	name := clientEmailName(lead)
	website := strings.TrimRight(strings.TrimSpace(cfg.SiteURL), "/")
	if strings.EqualFold(strings.TrimSpace(lead.Language), "fi") {
		lines := []string{
			"Hei " + name + ",",
			"",
			"Kiitos pyynnöstäsi. Olemme vastaanottaneet sen onnistuneesti. Odota yhteydenottoamme: asiakaspalvelumme ottaa sinuun yhteyttä sopiakseen hinnasta, ajasta ja työn yksityiskohdista.",
			"",
			fmt.Sprintf("Automekaanikon käynti alkaen %d €.", liikkuvaLeadFee(lead)),
			"Käyntimaksu kattaa vain mekaanikon saapumisen sovittuun paikkaan. Se ei sisällä diagnostiikkaa eikä korjausta.",
			fmt.Sprintf("Vianmääritys ja pienet korjaustyöt alkaen %d €, veloitetaan erikseen käyntimaksun lisäksi. Hinta sisältää vianmäärityksen ja tilanteeseen sopivan pienen korjauksen.", liikkuvaLeadDiagnosticFee(lead)),
			"Varaosat, vaativammat korjaukset ja lisätyöt hinnoitellaan ja sovitaan erikseen ennen työn aloittamista.",
			"",
			"Otamme sinuun yhteyttä sopiaksemme käyntiajan. Kun aika on vahvistettu, odota mekaanikkoa ilmoittamassasi paikassa sovittuna ajankohtana. Varmista, että autolle on esteetön pääsy.",
		}
		if lead.Car != "" {
			lines = append(lines, "", "Rekisterinumero: "+lead.Car)
		}
		if website != "" {
			lines = append(lines, "", "Verkkosivu: "+website)
		}
		return strings.Join(append(lines, "", "Ystävällisin terveisin,", "Liikkuva Auto Korjaamo"), "\n")
	}

	lines := []string{
		"Hello " + name + ",",
		"",
		"Thank you for your request. We have received it successfully. Please wait for our operator to contact you to agree on the price, time and details of the work.",
		"",
		fmt.Sprintf("Mechanic visit from €%d.", liikkuvaLeadFee(lead)),
		"The visit fee only covers the mechanic arriving at your agreed location. Diagnostics and repairs are not included.",
		fmt.Sprintf("Diagnostics and minor repairs from €%d are charged separately in addition to the visit fee. This covers diagnostics and a suitable minor repair where possible.", liikkuvaLeadDiagnosticFee(lead)),
		"Spare parts, more complex repairs and additional work are quoted and agreed separately before work begins.",
		"",
		"We will contact you to agree on an appointment time. Once confirmed, please wait for the mechanic at the location you provided in your request at the agreed time. Make sure the vehicle is accessible.",
	}
	if lead.Car != "" {
		lines = append(lines, "", "Vehicle registration: "+lead.Car)
	}
	if website != "" {
		lines = append(lines, "", "Website: "+website)
	}
	return strings.Join(append(lines, "", "Best regards,", "Liikkuva Auto Korjaamo"), "\n")
}

func buildLeadConfirmationHTML(cfg Config, lead Lead) string {
	if strings.EqualFold(strings.TrimSpace(lead.Language), "fi") {
		content := `<p style="margin:0 0 16px;">Kiitos pyynnöstäsi. Olemme vastaanottaneet sen onnistuneesti. Odota yhteydenottoamme: asiakaspalvelumme ottaa sinuun yhteyttä sopiakseen hinnasta, ajasta ja työn yksityiskohdista.</p>
		<div style="background:#f4f5f2;border:1px solid #e5e9e6;border-radius:16px;padding:18px;margin:0 0 18px;">
			<div style="font-size:24px;font-weight:800;color:#0f172a;">Automekaanikon käynti alkaen ` + fmt.Sprintf("%d €", liikkuvaLeadFee(lead)) + `</div>
			<p style="margin:10px 0 0;">Käyntimaksu kattaa vain mekaanikon saapumisen sovittuun paikkaan. Diagnostiikka ja korjaustyöt eivät sisälly käyntimaksuun.</p>
		</div>
		<p style="margin:0 0 14px;"><strong>Vianmääritys ja pienet korjaustyöt alkaen ` + fmt.Sprintf("%d €", liikkuvaLeadDiagnosticFee(lead)) + `.</strong> Hinta veloitetaan käyntimaksun lisäksi ja sisältää vianmäärityksen sekä soveltuvan pienen korjauksen, jos se on mahdollista. Varaosat ja vaativammat työt sovitaan ja hinnoitellaan erikseen ennen työn aloittamista.</p>
		<p style="margin:0;font-weight:700;">Otamme sinuun yhteyttä sopiaksemme ajan. Kun aika on vahvistettu, odota mekaanikkoa ilmoittamassasi paikassa sovittuna ajankohtana ja varmista pääsy autolle.</p>`
		return buildBrandedActionEmailHTML(cfg, lead, "Pyyntö vastaanotettu", content, "", "")
	}

	content := `<p style="margin:0 0 16px;">Thank you for your request. We have received it successfully. Please wait for our operator to contact you to agree on the price, time and details of the work.</p>
	<div style="background:#f4f5f2;border:1px solid #e5e9e6;border-radius:16px;padding:18px;margin:0 0 18px;">
		<div style="font-size:24px;font-weight:800;color:#0f172a;">Mechanic visit from €` + fmt.Sprintf("%d", liikkuvaLeadFee(lead)) + `</div>
		<p style="margin:10px 0 0;">The visit fee only covers the mechanic arriving at your agreed location. Diagnostics and repairs are not included.</p>
	</div>
	<p style="margin:0 0 14px;"><strong>Diagnostics and minor repairs from €` + fmt.Sprintf("%d", liikkuvaLeadDiagnosticFee(lead)) + `.</strong> Charged in addition to the visit fee; includes diagnostics and a suitable minor repair where possible. Parts and more complex repairs are agreed and priced separately before work begins.</p>
	<p style="margin:0;font-weight:700;">We will contact you to agree on a time. Once confirmed, please wait for the mechanic at the location in your request at the agreed time and ensure access to the vehicle.</p>`
	return buildBrandedActionEmailHTML(cfg, lead, "Request received", content, "", "")
}

func sendReviewInvitationEmail(cfg Config, lead Lead, reviewURL string) error {
	if !mailNotificationsEnabled(cfg) {
		return fmt.Errorf("mail config is incomplete")
	}
	if strings.TrimSpace(lead.Email) == "" {
		return fmt.Errorf("client email is empty")
	}
	if strings.TrimSpace(reviewURL) == "" {
		return fmt.Errorf("review URL is empty")
	}

	lang := sanitizeLanguage(lead.Language, "fi")
	subject := "Liikkuva Auto Korjaamo: Kerro kokemuksestasi"
	button := "Jätä arvostelu"
	textBody := "Hei " + clientEmailName(lead) + ",\n\nKiitos, että valitsit palvelumme. Arvostaisimme palautettasi palvelustamme.\n\nJätä arvostelu tästä linkistä:\n" + reviewURL + "\n\nArvostelu julkaistaan vasta hyväksynnän jälkeen.\n\nYstävällisin terveisin,\nLiikkuva"
	content := `<p style="margin:0 0 16px;">Kiitos, että valitsit palvelumme. Arvostaisimme palautettasi palvelustamme.</p><p style="margin:0;">Arvostelu julkaistaan vasta hyväksynnän jälkeen.</p>`
	if lang == "en" {
		subject = "Liikkuva Auto Korjaamo: Tell us about your experience"
		button = "Leave a review"
		textBody = "Hello " + clientEmailName(lead) + ",\n\nThank you for choosing Liikkuva Auto Korjaamo. We would appreciate your feedback about our service.\n\nLeave your review using this link:\n" + reviewURL + "\n\nThe review is published only after approval.\n\nBest regards,\nLiikkuva"
		content = `<p style="margin:0 0 16px;">Thank you for choosing Liikkuva Auto Korjaamo. We would appreciate your feedback about our service.</p><p style="margin:0;">The review is published only after approval.</p>`
	}

	return sendSMTPMail(cfg, lead.Email, subject, textBody, buildBrandedActionEmailHTML(cfg, lead, subject, content, button, reviewURL))
}

func clientEmailName(lead Lead) string {
	name := strings.TrimSpace(lead.Name)
	if name != "" {
		return name
	}
	if strings.EqualFold(strings.TrimSpace(lead.Language), "fi") {
		return "asiakas"
	}
	return "customer"
}

func buildBrandedActionEmailHTML(cfg Config, lead Lead, title, contentHTML, buttonLabel, buttonURL string) string {
	lang, greeting, closing, contact := "en", "Hello", "Best regards", "Contact us"
	path := "/en"
	if strings.EqualFold(strings.TrimSpace(lead.Language), "fi") {
		lang, greeting, closing, contact, path = "fi", "Hei", "Ystävällisin terveisin", "Ota yhteyttä", "/"
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.SiteURL), "/")
	if base == "" {
		base = "https://liikkuvakorjaamo.fi"
	}
	name := strings.TrimSpace(lead.Name)
	if name == "" {
		if lang == "fi" {
			name = "asiakas"
		} else {
			name = "customer"
		}
	}
	buttonHTML := ""
	if strings.TrimSpace(buttonLabel) != "" && strings.TrimSpace(buttonURL) != "" {
		buttonHTML = `<p><a href="` + html.EscapeString(buttonURL) + `" style="display:inline-block;background:#f97315;color:#23372b;text-decoration:none;font-weight:bold;padding:14px 22px;border-radius:10px;">` + html.EscapeString(buttonLabel) + `</a></p>`
	}
	contactHTML := ""
	if email := serviceVisibleEmail(cfg); email != "" {
		contactHTML = `<br>` + contact + `: <a style="color:#ffffff" href="mailto:` + html.EscapeString(email) + `">` + html.EscapeString(email) + `</a>`
	}
	return `<!doctype html><html lang="` + lang + `"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + html.EscapeString(title) + `</title></head><body style="margin:0;background:#f5f6f2;font-family:Arial,Helvetica,sans-serif;color:#23372b"><table role="presentation" width="100%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:24px 12px"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:640px;background:#ffffff;border:1px solid #dce5de;border-radius:16px;overflow:hidden"><tr><td style="background:#23372b;padding:24px;border-bottom:4px solid #f97315"><a href="` + html.EscapeString(base+path) + `"><img src="` + html.EscapeString(base) + `/static/img/logo-green.png?v=20261111" alt="Liikkuva auto korjaamo" width="280" style="display:block;width:280px;max-width:100%;height:auto;border:0"></a></td></tr><tr><td style="padding:28px 24px;font-size:16px;line-height:1.75"><h1 style="font-size:26px;line-height:1.3;margin:0 0 24px">` + html.EscapeString(title) + `</h1><p>` + greeting + ` ` + html.EscapeString(name) + `,</p>` + optionalPlateBlockHTML(lead.Car, lang) + contentHTML + buttonHTML + `<p>` + closing + `,<br>Liikkuva Auto Korjaamo</p></td></tr><tr><td style="background:#23372b;padding:24px;color:#ffffff;font-size:14px;line-height:1.8"><a href="` + html.EscapeString(base+path) + `" style="color:#ffffff">liikkuvakorjaamo.fi</a>` + contactHTML + `</td></tr></table></td></tr></table></body></html>`
}

func optionalPlateBlockHTML(value, lang string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	label := "Vehicle registration"
	if strings.EqualFold(strings.TrimSpace(lang), "fi") {
		label = "Rekisterinumero"
	}
	return `<div style="background:#f4f5f2;border:1px solid #e5e9e6;border-radius:14px;padding:14px 16px;margin:0 0 18px;"><div style="font-size:13px;color:#23372b;font-weight:700;text-transform:uppercase;">` + html.EscapeString(label) + `</div><div style="font-size:22px;color:#0f172a;font-weight:800;margin-top:6px;">` + html.EscapeString(value) + `</div></div>`
}

func nl2br(value string) string {
	return strings.ReplaceAll(value, "\n", "<br>")
}
