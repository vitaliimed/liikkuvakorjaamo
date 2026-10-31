package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

func sendLeadToTelegram(cfg Config, lead Lead) error {
	if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
		return fmt.Errorf("telegram config is empty")
	}

	if err := sendTelegramText(cfg, liikkuvaTelegramHeader()+buildTelegramMessage(lead)); err != nil {
		return err
	}

	if len(lead.PhotoPaths) > 0 {
		for i, photoPath := range lead.PhotoPaths {
			caption := liikkuvaTelegramPhotoCaption(lead, i+1)
			if err := sendTelegramDocumentFromFile(cfg, photoPath, caption); err != nil {
				return err
			}
		}
		return nil
	}

	for i, photoRef := range lead.PhotoURLs {
		caption := liikkuvaTelegramPhotoCaption(lead, i+1)
		if err := sendTelegramDocumentFromRef(cfg, photoRef, caption); err != nil {
			return err
		}
	}

	return nil
}

func liikkuvaTelegramHeader() string {
	return "🟦 <b>LIIKKUVA · liikkuvakorjaamo.fi</b>\n#Liikkuva\n\n"
}

func liikkuvaTelegramPhotoCaption(lead Lead, number int) string {
	car := []rune(strings.TrimSpace(lead.Car))
	if len(car) > 80 {
		car = car[:80]
	}
	return fmt.Sprintf("LIIKKUVA · liikkuvakorjaamo.fi\n#Liikkuva · Photo %d · %s", number, string(car))
}

func sendTelegramText(cfg Config, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.TelegramBotToken)

	form := url.Values{}
	form.Set("chat_id", cfg.TelegramChatID)
	form.Set("text", message)
	form.Set("parse_mode", "HTML")
	form.Set("link_preview_options", `{"is_disabled":true}`)

	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("Telegram: проверьте формат токена бота")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Telegram: не удалось соединиться с сервером отправки")
	}
	defer resp.Body.Close()

	return checkTelegramResponse(resp)
}

func sendTelegramDocumentFromFile(cfg Config, filePath, caption string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open telegram upload file: %w", err)
	}
	defer file.Close()

	return sendTelegramDocument(cfg, file, filepath.Base(filePath), caption)
}

func sendTelegramDocumentFromRef(cfg Config, photoRef, caption string) error {
	if strings.TrimSpace(photoRef) == "" {
		return nil
	}

	if strings.HasPrefix(photoRef, "gs://") {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		bucketName, objectName, err := splitGCSRef(photoRef)
		if err != nil {
			return err
		}

		client, err := storage.NewClient(ctx)
		if err != nil {
			return fmt.Errorf("create storage client for telegram upload: %w", err)
		}
		defer client.Close()

		reader, err := client.Bucket(bucketName).Object(objectName).NewReader(ctx)
		if err != nil {
			return fmt.Errorf("open gcs object for telegram upload: %w", err)
		}
		defer reader.Close()

		return sendTelegramDocument(cfg, reader, filepath.Base(objectName), caption)
	}

	return sendTelegramDocumentFromFile(cfg, photoRef, caption)
}

func splitGCSRef(ref string) (string, string, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(ref), "gs://")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid gcs ref: %s", ref)
	}
	return parts[0], parts[1], nil
}

func sendTelegramDocument(cfg Config, source io.Reader, filename, caption string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", cfg.TelegramBotToken)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("chat_id", cfg.TelegramChatID); err != nil {
		return fmt.Errorf("write chat_id: %w", err)
	}
	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return fmt.Errorf("write caption: %w", err)
		}
	}

	part, err := writer.CreateFormFile("document", filename)
	if err != nil {
		return fmt.Errorf("create telegram file part: %w", err)
	}

	if _, err := io.Copy(part, source); err != nil {
		return fmt.Errorf("copy telegram file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, &body)
	if err != nil {
		return fmt.Errorf("Telegram: проверьте формат токена бота")
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Telegram: не удалось соединиться с сервером вложений")
	}
	defer resp.Body.Close()

	return checkTelegramResponse(resp)
}

func checkTelegramResponse(response *http.Response) error {
	var result struct {
		OK   bool `json:"ok"`
		Code int  `json:"error_code"`
	}
	err := json.NewDecoder(io.LimitReader(response.Body, 128*1024)).Decode(&result)
	code := response.StatusCode
	if code >= 200 && code < 300 && err == nil && result.OK {
		return nil
	}
	if result.Code >= 400 && result.Code <= 599 {
		code = result.Code
	}
	reason := "неожиданный ответ сервера, проверьте доступность Telegram"
	switch code {
	case 400:
		reason = "проверьте Chat ID и формат сообщения"
	case 401:
		reason = "проверьте токен бота"
	case 403:
		reason = "боту запрещена отправка в этот чат"
	case 429:
		reason = "лимит отправки, повторите позже"
	}
	// Transport errors and response bodies can contain the token-bearing URL.
	// Keep shared AutoHelp/Liikkuva credentials out of logs and stored errors.
	return fmt.Errorf("Telegram: HTTP %d, %s", code, reason)
}

func buildTelegramMessage(lead Lead) string {
	var b strings.Builder

	if lead.Language == "fi" {
		b.WriteString("🚗 <b>Uusi liidi</b>\n\n")
		addTelegramLine(&b, "Aika", lead.CreatedAt)
		addTelegramLine(&b, "Nimi", lead.Name)
		addTelegramLine(&b, "Puhelin", lead.Phone)
		addTelegramLine(&b, "Sähköposti", lead.Email)
		addTelegramLine(&b, "Osoite", lead.Address)
		addTelegramLine(&b, "Osoitteen tarkennus", lead.AddressHint)
		addTelegramLine(&b, "Karttalinkki", lead.MapsURL)

		if lead.Latitude != "" || lead.Longitude != "" {
			addTelegramLine(&b, "Koordinaatit", strings.TrimSpace(lead.Latitude+", "+lead.Longitude))
		}

		addTelegramLine(&b, "Rekisterinumero", lead.Car)
		addTelegramLine(&b, "Käyntimaksu", fmt.Sprintf("%d €", liikkuvaLeadFee(lead)))

		if len(lead.PhotoURLs) > 0 {
			addTelegramLine(&b, "Kuvia", fmt.Sprintf("%d", len(lead.PhotoURLs)))
		}

		if strings.TrimSpace(lead.Message) != "" {
			b.WriteString("\n<b>Viesti:</b>\n")
			b.WriteString(html.EscapeString(lead.Message))
		}

		return b.String()
	}

	b.WriteString("🚗 <b>New lead</b>\n\n")
	addTelegramLine(&b, "Time", lead.CreatedAt)
	addTelegramLine(&b, "Name", lead.Name)
	addTelegramLine(&b, "Phone", lead.Phone)
	addTelegramLine(&b, "Email", lead.Email)
	addTelegramLine(&b, "Address", lead.Address)
	addTelegramLine(&b, "Address note", lead.AddressHint)
	addTelegramLine(&b, "Map link", lead.MapsURL)

	if lead.Latitude != "" || lead.Longitude != "" {
		addTelegramLine(&b, "Coordinates", strings.TrimSpace(lead.Latitude+", "+lead.Longitude))
	}

	addTelegramLine(&b, "Registration number", lead.Car)
	addTelegramLine(&b, "Visit fee", fmt.Sprintf("€%d", liikkuvaLeadFee(lead)))

	if len(lead.PhotoURLs) > 0 {
		addTelegramLine(&b, "Photos", fmt.Sprintf("%d", len(lead.PhotoURLs)))
	}

	if strings.TrimSpace(lead.Message) != "" {
		b.WriteString("\n<b>Message:</b>\n")
		b.WriteString(html.EscapeString(lead.Message))
	}

	return b.String()
}

func addTelegramLine(b *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(label))
	b.WriteString(":</b> ")
	b.WriteString(html.EscapeString(value))
	b.WriteString("\n")
}
