package main

import (
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
			lines = append(lines, "📧 Service email: "+serviceEmail)
		}
		if website != "" {
			lines = append(lines, "🌐 Website: "+website)
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
		return `<!DOCTYPE html>
<html lang="fi">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="color-scheme" content="light only">
  <meta name="supported-color-schemes" content="light only">
  <title>Liikkuva Auto Korjaamo</title>
</head>
<body style="margin:0;padding:0;background:#edf3f9;font-family:Arial,Helvetica,sans-serif;color:#0f172a;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background:#edf3f9;margin:0;padding:0;">
    <tr>
      <td align="center" style="padding:24px 12px;">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:640px;background:#ffffff;border:1px solid #dbe7f3;border-radius:22px;overflow:hidden;">
          <tr>
            <td style="background:#0b3b66;padding:24px 28px;">
              <div style="font-size:12px;line-height:1.4;color:#cfe8ff;font-weight:700;letter-spacing:.08em;text-transform:uppercase;">Liikkuva Auto Korjaamo</div>
              <div style="font-size:30px;line-height:1.2;color:#ffffff;font-weight:800;margin-top:10px;">🚗 Ajoneuvosi on valmis</div>
            </td>
          </tr>

          <tr>
            <td style="padding:28px 24px;">
              <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
                <tr>
                  <td style="font-size:18px;line-height:1.6;color:#0f172a;padding:0 0 18px 0;">
                    Hei ` + html.EscapeString(name) + `,
                  </td>
                </tr>

                <tr>
                  <td style="padding:0 0 18px 0;">
                    <div style="background:#f8fafc;border:1px solid #dbe7f3;border-radius:16px;padding:18px 18px;font-size:18px;line-height:1.6;color:#0f172a;font-weight:700;">
                      ✅ Ajoneuvosi on valmis.
                    </div>
                  </td>
                </tr>

                ` + optionalPlateRowHTML(car, lead.Language) + `

                <tr>
                  <td style="font-size:16px;line-height:1.75;color:#334155;padding:0 0 20px 0;">
                    Voit vastata tähän viestiin tai ottaa meihin yhteyttä sopiaksesi luovutuksesta.
                  </td>
                </tr>

                <tr>
                  <td style="padding:0 0 22px 0;">
                    <div style="background:#fff7ed;border:1px solid #fed7aa;border-radius:16px;padding:16px 18px;">
                      ` + optionalInfoRowHTML("📧", "Service email", serviceEmail) + `
                      ` + optionalInfoRowHTML("🌐", "Website", website) + `
                    </div>
                  </td>
                </tr>

                <tr>
                  <td style="font-size:16px;line-height:1.75;color:#334155;padding:0;">
                    Ystävällisin terveisin,<br>
                    <strong style="color:#0f172a;">Liikkuva Auto Korjaamo</strong>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="color-scheme" content="light only">
  <meta name="supported-color-schemes" content="light only">
  <title>Liikkuva Auto Korjaamo</title>
</head>
<body style="margin:0;padding:0;background:#edf3f9;font-family:Arial,Helvetica,sans-serif;color:#0f172a;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background:#edf3f9;margin:0;padding:0;">
    <tr>
      <td align="center" style="padding:24px 12px;">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:640px;background:#ffffff;border:1px solid #dbe7f3;border-radius:22px;overflow:hidden;">
          <tr>
            <td style="background:#0b3b66;padding:24px 28px;">
              <div style="font-size:12px;line-height:1.4;color:#cfe8ff;font-weight:700;letter-spacing:.08em;text-transform:uppercase;">Liikkuva Auto Korjaamo</div>
              <div style="font-size:30px;line-height:1.2;color:#ffffff;font-weight:800;margin-top:10px;">🚗 Your vehicle is ready</div>
            </td>
          </tr>

          <tr>
            <td style="padding:28px 24px;">
              <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
                <tr>
                  <td style="font-size:18px;line-height:1.6;color:#0f172a;padding:0 0 18px 0;">
                    Hello ` + html.EscapeString(name) + `,
                  </td>
                </tr>

                <tr>
                  <td style="padding:0 0 18px 0;">
                    <div style="background:#f8fafc;border:1px solid #dbe7f3;border-radius:16px;padding:18px 18px;font-size:18px;line-height:1.6;color:#0f172a;font-weight:700;">
                      ✅ Your vehicle is ready.
                    </div>
                  </td>
                </tr>

                ` + optionalPlateRowHTML(car, lead.Language) + `

                <tr>
                  <td style="font-size:16px;line-height:1.75;color:#334155;padding:0 0 20px 0;">
                    You can reply to this email or contact us to arrange pickup or handover.
                  </td>
                </tr>

                <tr>
                  <td style="padding:0 0 22px 0;">
                    <div style="background:#fff7ed;border:1px solid #fed7aa;border-radius:16px;padding:16px 18px;">
                      ` + optionalInfoRowHTML("📧", "Service email", serviceEmail) + `
                      ` + optionalInfoRowHTML("🌐", "Website", website) + `
                    </div>
                  </td>
                </tr>

                <tr>
                  <td style="font-size:16px;line-height:1.75;color:#334155;padding:0;">
                    Best regards,<br>
                    <strong style="color:#0f172a;">Liikkuva Auto Korjaamo</strong>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`
}

func optionalInfoRowHTML(icon, label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	escapedLabel := html.EscapeString(label)
	escapedValue := html.EscapeString(value)

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return `<div style="font-size:15px;line-height:1.8;margin:6px 0;"><span style="margin-right:8px;">` + icon + `</span><strong>` + escapedLabel + `:</strong> <a href="` + escapedValue + `" style="color:#0369a1;text-decoration:none;">` + escapedValue + `</a></div>`
	}

	if strings.Contains(value, "@") {
		return `<div style="font-size:15px;line-height:1.8;margin:6px 0;"><span style="margin-right:8px;">` + icon + `</span><strong>` + escapedLabel + `:</strong> <a href="mailto:` + escapedValue + `" style="color:#0369a1;text-decoration:none;">` + escapedValue + `</a></div>`
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
			<div style="background:#eff6ff;border:1px solid #bfdbfe;border-radius:14px;padding:14px 16px;">
				<div style="font-size:13px;line-height:1.4;color:#1d4ed8;font-weight:700;text-transform:uppercase;letter-spacing:.04em;">🚘 ` + html.EscapeString(label) + `</div>
				<div style="font-size:22px;line-height:1.3;color:#0f172a;font-weight:800;margin-top:6px;">` + html.EscapeString(value) + `</div>
			</div>
		</td>
	</tr>`
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

	lines := []string{
		"Hello " + name + ",",
		"",
		message,
	}

	if car != "" {
		lines = append(lines, "", "Vehicle registration: "+car)
	}

	if serviceEmail != "" {
		lines = append(lines, "", "Service email: "+serviceEmail)
	}
	if website != "" {
		lines = append(lines, "Website: "+website)
	}

	lines = append(lines, "", "Best regards,", "Liikkuva Auto Korjaamo")
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
	name := strings.TrimSpace(lead.Name)
	if name == "" {
		name = "customer"
	}

	car := strings.TrimSpace(lead.Car)
	serviceEmail := serviceVisibleEmail(cfg)
	website := strings.TrimRight(strings.TrimSpace(cfg.SiteURL), "/")

	messageHTML := nl2br(html.EscapeString(message))

	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="color-scheme" content="light only">
  <meta name="supported-color-schemes" content="light only">
  <title>Liikkuva Auto Korjaamo</title>
</head>
<body style="margin:0;padding:0;background:#edf3f9;font-family:Arial,Helvetica,sans-serif;color:#0f172a;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background:#edf3f9;margin:0;padding:0;">
    <tr>
      <td align="center" style="padding:24px 12px;">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:640px;background:#ffffff;border:1px solid #dbe7f3;border-radius:22px;overflow:hidden;">
          <tr>
            <td style="background:#0b3b66;padding:24px 28px;">
              <div style="font-size:12px;line-height:1.4;color:#cfe8ff;font-weight:700;letter-spacing:.08em;text-transform:uppercase;">Liikkuva Auto Korjaamo</div>
              <div style="font-size:30px;line-height:1.2;color:#ffffff;font-weight:800;margin-top:10px;">📩 ` + html.EscapeString(subject) + `</div>
            </td>
          </tr>

          <tr>
            <td style="padding:28px 24px;">
              <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
                <tr>
                  <td style="font-size:18px;line-height:1.6;color:#0f172a;padding:0 0 18px 0;">
                    Hello ` + html.EscapeString(name) + `,
                  </td>
                </tr>

                ` + optionalPlateRowHTML(car, lead.Language) + `

                <tr>
                  <td style="font-size:16px;line-height:1.8;color:#334155;padding:0 0 20px 0;">
                    ` + messageHTML + `
                  </td>
                </tr>

                <tr>
                  <td style="padding:0 0 22px 0;">
                    <div style="background:#fff7ed;border:1px solid #fed7aa;border-radius:16px;padding:16px 18px;">
                      ` + optionalInfoRowHTML("📧", "Service email", serviceEmail) + `
                      ` + optionalInfoRowHTML("🌐", "Website", website) + `
                    </div>
                  </td>
                </tr>

                <tr>
                  <td style="font-size:16px;line-height:1.75;color:#334155;padding:0;">
                    Best regards,<br>
                    <strong style="color:#0f172a;">Liikkuva Auto Korjaamo</strong>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`
}

func sendLeadConfirmationEmail(cfg Config, lead Lead) error {
	if !mailNotificationsEnabled(cfg) {
		return fmt.Errorf("mail config is incomplete")
	}
	if strings.TrimSpace(lead.Email) == "" {
		return fmt.Errorf("client email is empty")
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
			"Kiitos pyynnöstäsi. Olemme vastaanottaneet sen onnistuneesti.",
			"",
			fmt.Sprintf("Asentajan käynti maksaa %d €.", liikkuvaLeadFee(lead)),
			"Käyntimaksu kattaa saapumisen. Diagnostiikka, korjaukset ja varaosat sovitaan ja veloitetaan erikseen.",
			"Varaosat ja laajemmat työt eivät sisälly käyntimaksuun. Niistä sovitaan kanssasi ennen työn aloittamista.",
			"",
			"Odota viestiämme. Ilmoitamme, kun asentajamme on valmis lähtemään luoksesi, ja vahvistamme ajan.",
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
		"Thank you for your request. We have received it successfully.",
		"",
		fmt.Sprintf("The mechanic visit costs €%d.", liikkuvaLeadFee(lead)),
		"The visit fee covers travel. Diagnostics, repairs and spare parts are agreed and charged separately.",
		"Parts and more extensive work are not included in the visit fee. We agree them with you before starting the work.",
		"",
		"Please wait for our message. We will let you know when our mechanic is ready to leave for your location and confirm the time.",
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
		content := `<p style="margin:0 0 16px;">Kiitos pyynnöstäsi. Olemme vastaanottaneet sen onnistuneesti.</p>
		<div style="background:#eff6ff;border:1px solid #bfdbfe;border-radius:16px;padding:18px;margin:0 0 18px;">
			<div style="font-size:24px;font-weight:800;color:#0f172a;">Asentajan käynti ` + fmt.Sprintf("%d €", liikkuvaLeadFee(lead)) + `</div>
			<p style="margin:10px 0 0;">Käyntimaksu kattaa saapumisen. Diagnostiikka, korjaukset ja varaosat sovitaan ja veloitetaan erikseen.</p>
		</div>
		<p style="margin:0 0 16px;">Varaosat ja laajemmat työt sovitaan ja veloitetaan erikseen ennen työn aloittamista.</p>
		<p style="margin:0;font-weight:700;">Odota viestiämme. Ilmoitamme, kun asentajamme on valmis lähtemään luoksesi, ja vahvistamme ajan.</p>`
		return buildBrandedActionEmailHTML(cfg, lead, "Pyyntö vastaanotettu", content, "", "")
	}

	content := `<p style="margin:0 0 16px;">Thank you for your request. We have received it successfully.</p>
	<div style="background:#eff6ff;border:1px solid #bfdbfe;border-radius:16px;padding:18px;margin:0 0 18px;">
		<div style="font-size:24px;font-weight:800;color:#0f172a;">Mechanic visit €` + fmt.Sprintf("%d", liikkuvaLeadFee(lead)) + `</div>
		<p style="margin:10px 0 0;">The visit fee covers travel. Diagnostics, repairs and spare parts are agreed and charged separately.</p>
	</div>
	<p style="margin:0 0 16px;">Parts and more extensive work are agreed and charged separately before work begins.</p>
	<p style="margin:0;font-weight:700;">Please wait for our message. We will confirm when our mechanic is ready to leave for your location.</p>`
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
	buttonHTML := ""
	if strings.TrimSpace(buttonLabel) != "" && strings.TrimSpace(buttonURL) != "" {
		buttonHTML = `<p style="margin:22px 0 0;"><a href="` + html.EscapeString(buttonURL) + `" style="display:inline-block;background:#f59e0b;color:#111827;text-decoration:none;font-size:16px;font-weight:800;padding:14px 22px;border-radius:14px;">` + html.EscapeString(buttonLabel) + `</a></p>`
	}

	return `<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Liikkuva Auto Korjaamo</title></head>
<body style="margin:0;padding:0;background:#edf3f9;font-family:Arial,Helvetica,sans-serif;color:#0f172a;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr><td align="center" style="padding:24px 12px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:640px;background:#fff;border:1px solid #dbe7f3;border-radius:22px;overflow:hidden;">
<tr><td style="background:#0b3b66;padding:24px 28px;color:#fff;"><div style="font-size:12px;color:#cfe8ff;font-weight:700;letter-spacing:.08em;text-transform:uppercase;">Liikkuva Auto Korjaamo</div><div style="font-size:30px;line-height:1.2;font-weight:800;margin-top:10px;">` + html.EscapeString(title) + `</div></td></tr>
<tr><td style="padding:28px 24px;font-size:16px;line-height:1.75;color:#334155;"><p style="margin:0 0 18px;font-size:18px;color:#0f172a;">` + html.EscapeString(clientEmailName(lead)) + `,</p>` + optionalPlateBlockHTML(lead.Car, lead.Language) + contentHTML + buttonHTML + `<p style="margin:24px 0 0;">Liikkuva Auto Korjaamo<br><a href="` + html.EscapeString(strings.TrimRight(cfg.SiteURL, "/")) + `" style="color:#0369a1;">` + html.EscapeString(strings.TrimRight(cfg.SiteURL, "/")) + `</a></p></td></tr>
</table></td></tr></table></body></html>`
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
	return `<div style="background:#eff6ff;border:1px solid #bfdbfe;border-radius:14px;padding:14px 16px;margin:0 0 18px;"><div style="font-size:13px;color:#1d4ed8;font-weight:700;text-transform:uppercase;">` + html.EscapeString(label) + `</div><div style="font-size:22px;color:#0f172a;font-weight:800;margin-top:6px;">` + html.EscapeString(value) + `</div></div>`
}

func nl2br(value string) string {
	return strings.ReplaceAll(value, "\n", "<br>")
}
