package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	maxMultipartMemory = 40 << 20
	maxPhotoCount      = 3
	maxPhotoSize       = 10 << 20
)

type LeadHandler struct {
	cfg Config
}

func NewLeadHandler(cfg Config) LeadHandler {
	return LeadHandler{cfg: cfg}
}

func (h LeadHandler) Submit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, APIResponse{OK: false, Message: "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMultipartMemory)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
			writeJSON(w, 400, APIResponse{OK: false, Message: "invalid multipart form"})
			return
		}
	} else if err := r.ParseForm(); err != nil {
		writeJSON(w, 400, APIResponse{OK: false, Message: "invalid form"})
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if strings.TrimSpace(r.FormValue("company")) != "" {
		writeJSON(w, 200, APIResponse{OK: false, Message: "Request could not be accepted"})
		return
	}
	lang := sanitizeLanguage(strings.TrimSpace(r.FormValue("language")), h.cfg.DefaultLanguage)
	lead := buildLeadFromRequest(h.cfg, r, lang)
	settings, settingsErr := loadSiteSettings(r.Context(), h.cfg)
	if settingsErr != nil {
		writeJSON(w, 503, APIResponse{OK: false, Message: saveErrorMessageByLanguage(lang)})
		return
	}
	lead.VisitPriceEUR = liikkuvaVisitFee(settings)
	lead.DiagnosticFromEUR = liikkuvaProDefaults(settings).ProDiagnosticEUR
	if issues := validateLead(lead); len(issues) > 0 {
		writeJSON(w, 400, APIResponse{OK: false, Message: validationMessageByLanguage(lang), Errors: issues})
		return
	}
	photoURLs, photoPaths, err := saveUploadedPhotos(h.cfg, r)
	if err != nil {
		writeJSON(w, 400, APIResponse{OK: false, Message: err.Error()})
		return
	}
	defer deleteTempFiles(photoPaths)
	lead.PhotoURLs = photoURLs
	lead.PhotoPaths = photoPaths
	if _, err := h.handleAsyncSubmit(r.Context(), lead); err != nil {
		log.Printf("Lead was not saved: %v", err)
		writeJSON(w, 503, APIResponse{OK: false, Message: saveErrorMessageByLanguage(lang)})
		return
	}
	writeJSON(w, 200, APIResponse{OK: true, Message: successMessageByLanguage(lang), SubmissionID: lead.SubmissionID, RedirectURL: thankYouRedirectByLanguage(lang)})
}

func (h LeadHandler) ProcessLead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{OK: false, Message: "method not allowed"})
		return
	}

	if !isAuthorizedInternalTaskRequest(h.cfg, r) {
		writeJSON(w, http.StatusUnauthorized, APIResponse{OK: false, Message: "unauthorized"})
		return
	}

	var payload LeadTaskPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Message: "invalid task payload"})
		return
	}

	payload.LeadID = strings.TrimSpace(payload.LeadID)
	if payload.LeadID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Message: "lead_id is required"})
		return
	}

	if err := processLeadFromRecord(r.Context(), h.cfg, payload.LeadID); err != nil {
		log.Printf("Lead task failed for %s: %v", payload.LeadID, err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Message: "processing failed"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{OK: true, Message: "processed"})
}

func (h LeadHandler) handleAsyncSubmit(ctx context.Context, lead Lead) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	leadID, err := saveLeadRecord(ctx, h.cfg, lead)
	if err != nil {
		return false, err
	}
	_ = markLeadQueued(ctx, h.cfg, leadID)
	queueCtx, queueCancel := context.WithTimeout(ctx, 5*time.Second)
	defer queueCancel()
	if err := enqueueLeadProcessingTask(queueCtx, h.cfg, leadID); err == nil {
		return false, nil
	}
	log.Printf("Lead %s saved; queue unavailable, admin retry required", leadID)
	_ = markLeadFailed(ctx, h.cfg, leadID, "Заявка сохранена. Очередь недоступна: повторите обработку в админке.")
	return false, nil
}

func buildLeadFromRequest(cfg Config, r *http.Request, lang string) Lead {
	return Lead{
		Site:           liikkuvaDomain,
		SubmissionID:   liikkuvaSubmissionID(),
		Name:           strings.TrimSpace(r.FormValue("name")),
		Phone:          normalizePhone(strings.TrimSpace(r.FormValue("phone"))),
		Email:          strings.TrimSpace(r.FormValue("email")),
		Address:        strings.TrimSpace(r.FormValue("address")),
		AddressHint:    strings.TrimSpace(r.FormValue("address_hint")),
		LocationSource: strings.TrimSpace(r.FormValue("location_source")),
		Latitude:       strings.TrimSpace(r.FormValue("latitude")),
		Longitude:      strings.TrimSpace(r.FormValue("longitude")),
		MapsURL:        strings.TrimSpace(r.FormValue("maps_url")),
		Car:            normalizeRegistrationNumber(strings.TrimSpace(r.FormValue("car"))),
		Message:        strings.TrimSpace(r.FormValue("message")),
		Language:       lang,
		Source:         strings.TrimSpace(r.FormValue("source")),
		PageURL:        strings.TrimSpace(r.FormValue("page_url")),
		Referer:        strings.TrimSpace(r.FormValue("referer")),
		UTMSource:      strings.TrimSpace(r.FormValue("utm_source")),
		UTMMedium:      strings.TrimSpace(r.FormValue("utm_medium")),
		UTMCampaign:    strings.TrimSpace(r.FormValue("utm_campaign")),
		UTMTerm:        strings.TrimSpace(r.FormValue("utm_term")),
		UTMContent:     strings.TrimSpace(r.FormValue("utm_content")),
		UserAgent:      r.UserAgent(),
		IP:             getClientIP(r),
		CreatedAt:      liikkuvaTimestamp(time.Now()),
		ScriptToken:    "",
	}
}

func processLeadNow(ctx context.Context, cfg Config, lead Lead) error {
	if err := processLeadCoreNow(cfg, lead); err != nil {
		return err
	}
	if strings.TrimSpace(lead.Email) != "" {
		if err := sendLeadConfirmationEmail(cfg, lead); err != nil {
			log.Printf("Lead confirmation email warning: %v", err)
		}
	}
	return nil
}

func processLeadCoreNow(cfg Config, lead Lead) error {
	if err := sendLeadToTelegram(cfg, lead); err != nil {
		return fmt.Errorf("telegram: %w", err)
	}
	if err := sendLeadToGoogleAppsScript(cfg, lead); err != nil {
		return fmt.Errorf("google apps script: %w", err)
	}
	return nil
}

func processLeadFromRecord(ctx context.Context, cfg Config, leadID string) error {
	record, err := getLeadRecord(ctx, cfg, leadID)
	if err != nil {
		return err
	}

	confirmationComplete := strings.TrimSpace(record.Lead.Email) == "" || record.ConfirmationEmailStatus == "sent"
	if record.ProcessingStatus == "done" && confirmationComplete {
		return nil
	}

	coreAlreadyDone := record.ProcessingStatus == "done"
	if !coreAlreadyDone {
		if err := markLeadProcessing(ctx, cfg, leadID); err != nil {
			log.Printf("markLeadProcessing warning for %s: %v", leadID, err)
		}
	}

	var processingErrs []string

	if !coreAlreadyDone && record.TelegramStatus != "success" {
		if err := sendLeadToTelegram(cfg, record.Lead); err != nil {
			processingErrs = append(processingErrs, "telegram: "+err.Error())
			_ = markTelegramStatus(ctx, cfg, leadID, "failed", err.Error())
		} else {
			_ = markTelegramStatus(ctx, cfg, leadID, "success", "")
		}
	}

	if !coreAlreadyDone && record.SheetsStatus != "success" {
		if err := sendLeadToGoogleAppsScript(cfg, record.Lead); err != nil {
			processingErrs = append(processingErrs, "apps_script: "+err.Error())
			_ = markSheetsStatus(ctx, cfg, leadID, "failed", err.Error())
		} else {
			_ = markSheetsStatus(ctx, cfg, leadID, "success", "")
		}
	}

	if strings.TrimSpace(record.Lead.Email) != "" && record.ConfirmationEmailStatus != "sent" {
		if err := sendLeadConfirmationEmail(cfg, record.Lead); err != nil {
			log.Printf("Confirmation email failed for %s: %v", leadID, err)
			_ = markConfirmationEmailFailed(ctx, cfg, leadID, err.Error())
		} else if err := markConfirmationEmailSent(ctx, cfg, leadID); err != nil {
			log.Printf("Confirmation email sent, but status update failed for %s: %v", leadID, err)
		}
	}

	if len(processingErrs) > 0 {
		joined := strings.Join(processingErrs, " | ")
		_ = markLeadFailed(ctx, cfg, leadID, joined)
		return errors.New(joined)
	}

	if !coreAlreadyDone {
		if err := markLeadDone(ctx, cfg, leadID); err != nil {
			return err
		}
	}

	return nil
}

func validateLead(lead Lead) map[string]string {
	errors := map[string]string{}

	if lead.Name == "" {
		if lead.Language == "fi" {
			errors["name"] = "nimi on pakollinen"
		} else {
			errors["name"] = "name is required"
		}
	}

	if lead.Phone == "" {
		if lead.Language == "fi" {
			errors["contact"] = "anna puhelinnumero"
		} else {
			errors["contact"] = "phone is required"
		}
	}

	if lead.Phone != "" && !isValidPhone(lead.Phone) {
		if lead.Language == "fi" {
			errors["phone"] = "anna kelvollinen puhelinnumero"
		} else {
			errors["phone"] = "enter a valid phone number"
		}
	}

	if lead.Email != "" && !isValidEmail(lead.Email) {
		if lead.Language == "fi" {
			errors["email"] = "anna kelvollinen sähköpostiosoite"
		} else {
			errors["email"] = "enter a valid email address"
		}
	}

	if lead.Car != "" && !isValidRegistrationNumber(lead.Car) {
		if lead.Language == "fi" {
			errors["car"] = "anna kelvollinen rekisterinumero muodossa ABC-123"
		} else {
			errors["car"] = "enter a valid registration number like ABC-123"
		}
	}

	return errors
}

func sanitizeLanguage(lang, fallback string) string {
	if lang == "fi" || lang == "en" {
		return lang
	}
	if fallback == "en" {
		return "en"
	}
	return "fi"
}

func normalizePhone(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	digits := regexp.MustCompile(`\D`).ReplaceAllString(input, "")
	if digits == "" {
		return ""
	}

	if strings.HasPrefix(digits, "358") {
		digits = digits[3:]
	} else if strings.HasPrefix(digits, "0") {
		digits = digits[1:]
	}

	if len(digits) > 9 {
		digits = digits[:9]
	}

	if digits == "" {
		return ""
	}

	parts := []string{}
	if len(digits) > 0 {
		end := minInt(len(digits), 2)
		parts = append(parts, digits[:end])
	}
	if len(digits) > 2 {
		end := minInt(len(digits), 5)
		parts = append(parts, digits[2:end])
	}
	if len(digits) > 5 {
		end := minInt(len(digits), 7)
		parts = append(parts, digits[5:end])
	}
	if len(digits) > 7 {
		end := minInt(len(digits), 9)
		parts = append(parts, digits[7:end])
	}

	return "+358 " + strings.Join(parts, " ")
}

func isValidPhone(phone string) bool {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return false
	}

	digitsOnly := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")
	if strings.HasPrefix(digitsOnly, "358") {
		return len(digitsOnly) == 12
	}
	if strings.HasPrefix(digitsOnly, "0") {
		return len(digitsOnly) == 10
	}
	return len(digitsOnly) >= 7 && len(digitsOnly) <= 15
}

func normalizeRegistrationNumber(input string) string {
	input = strings.ToUpper(strings.TrimSpace(input))
	if input == "" {
		return ""
	}

	clean := regexp.MustCompile(`[^A-Z0-9]`).ReplaceAllString(input, "")
	letters := regexp.MustCompile(`[^A-Z]`).ReplaceAllString(clean, "")
	digits := regexp.MustCompile(`[^0-9]`).ReplaceAllString(clean, "")

	if len(letters) > 3 {
		letters = letters[:3]
	}
	if len(digits) > 3 {
		digits = digits[:3]
	}
	if letters == "" {
		return digits
	}
	if digits == "" {
		return letters
	}
	return letters + "-" + digits
}

func isValidRegistrationNumber(value string) bool {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return true
	}
	return regexp.MustCompile(`^[A-Z]{3}-[0-9]{1,3}$`).MatchString(value)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func saveUploadedPhotos(cfg Config, r *http.Request) ([]string, []string, error) {
	if strings.TrimSpace(cfg.GCSBucketName) == "" {
		return saveUploadedPhotosLocally(cfg, r)
	}
	return saveUploadedPhotosToGCS(cfg, r)
}

func saveUploadedPhotosLocally(cfg Config, r *http.Request) ([]string, []string, error) {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, nil, nil
	}

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		return nil, nil, nil
	}

	if len(files) > maxPhotoCount {
		return nil, nil, fmt.Errorf("please upload up to %d photos", maxPhotoCount)
	}

	now := time.Now()
	targetDir := filepath.Join("static", "uploads", now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to create upload dir: %w", err)
	}

	var urls []string
	var paths []string

	for i, header := range files {
		if header == nil || header.Filename == "" {
			continue
		}
		if header.Size > maxPhotoSize {
			return nil, nil, fmt.Errorf("each photo must be smaller than 10 MB")
		}

		file, err := header.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("open uploaded file: %w", err)
		}

		sniff := make([]byte, 512)
		n, _ := file.Read(sniff)
		if n == 0 {
			_ = file.Close()
			return nil, nil, fmt.Errorf("empty file is not allowed")
		}

		detectedType := http.DetectContentType(sniff[:n])
		if seeker, ok := file.(io.Seeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
		}
		if !strings.HasPrefix(detectedType, "image/") {
			_ = file.Close()
			return nil, nil, fmt.Errorf("only image files are allowed")
		}

		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			if exts, _ := mime.ExtensionsByType(detectedType); len(exts) > 0 {
				ext = exts[0]
			}
		}
		if ext == "" {
			ext = ".jpg"
		}

		name := fmt.Sprintf("%s_%02d%s", time.Now().UTC().Format("20060102T150405.000000000Z"), i+1, ext)
		localPath := filepath.Join(targetDir, name)

		dst, err := os.Create(localPath)
		if err != nil {
			_ = file.Close()
			return nil, nil, fmt.Errorf("create local upload file: %w", err)
		}

		if _, err := io.Copy(dst, file); err != nil {
			_ = dst.Close()
			_ = file.Close()
			return nil, nil, fmt.Errorf("save local upload file: %w", err)
		}

		_ = dst.Close()
		_ = file.Close()

		urls = append(urls, "/"+filepath.ToSlash(localPath))
		paths = append(paths, localPath)
	}

	return urls, paths, nil
}

func validationMessageByLanguage(lang string) string {
	if lang == "fi" {
		return "lomakkeessa on virhe"
	}
	return "validation error"
}

func successMessageByLanguage(lang string) string {
	if lang == "fi" {
		return fmt.Sprintf("Kiitos pyynnöstä. Asentajan käynti maksaa %d €. Odota viestiämme — vahvistamme, milloin asentaja voi lähteä luoksesi.", visitPriceEUR)
	}
	return fmt.Sprintf("Thank you for your request. The mechanic visit costs €%d. Please wait for our message confirming when the mechanic can leave for your location.", visitPriceEUR)
}

func saveErrorMessageByLanguage(lang string) string {
	if lang == "fi" {
		return "pyynnön tallennus epäonnistui"
	}
	return "failed to save lead"
}

func thankYouRedirectByLanguage(lang string) string {
	if lang == "fi" {
		return "/thank-you-fi"
	}
	return "/thank-you-en"
}
