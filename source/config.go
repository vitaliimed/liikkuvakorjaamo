package main

import (
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppPort               string
	SiteURL               string
	TelegramBotToken      string
	TelegramChatID        string
	GoogleAppsScriptURL   string
	GoogleAppsScriptToken string
	DefaultLanguage       string

	GCSBucketName  string
	GCSPhotoPrefix string
	GCSMaxPhotos   int

	FirestoreProjectID         string
	FirestoreLeadsCollection   string
	FirestoreReviewsCollection string
	CloudTasksProjectID        string
	CloudTasksLocation         string
	CloudTasksQueue            string
	InternalTaskToken          string
	InternalProcessLeadURL     string
	TasksServiceAccountEmail   string
	AdminPassword              string
	ReviewTokenSecret          string
	SiteSettingsCollection     string

	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	MailFrom     string
	MailReplyTo  string
	MailCopyTo   string
}

func loadConfig() Config {
	siteURL := getEnv("SITE_URL", "http://localhost:8080")
	processLeadURL := getEnv("INTERNAL_PROCESS_LEAD_URL", strings.TrimRight(siteURL, "/")+"/internal/process-lead")
	firestoreProjectID := getEnv("FIRESTORE_PROJECT_ID", getEnv("GOOGLE_CLOUD_PROJECT", ""))
	cloudTasksProjectID := getEnv("CLOUD_TASKS_PROJECT_ID", getEnv("GOOGLE_CLOUD_PROJECT", ""))

	return Config{
		AppPort:               getEnv("APP_PORT", getEnv("PORT", "8080")),
		SiteURL:               siteURL,
		TelegramBotToken:      getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:        getEnv("TELEGRAM_CHAT_ID", ""),
		GoogleAppsScriptURL:   getEnv("GOOGLE_APPS_SCRIPT_URL", ""),
		GoogleAppsScriptToken: getEnv("GOOGLE_APPS_SCRIPT_TOKEN", ""),
		DefaultLanguage:       getEnv("DEFAULT_LANGUAGE", "fi"),

		GCSBucketName:  getEnv("GCS_BUCKET_NAME", ""),
		GCSPhotoPrefix: getEnv("GCS_PHOTO_PREFIX", "lead-photos"),
		GCSMaxPhotos:   getEnvAsInt("GCS_MAX_PHOTOS", 500),

		FirestoreProjectID:         firestoreProjectID,
		FirestoreLeadsCollection:   getEnv("FIRESTORE_LEADS_COLLECTION", "leads"),
		FirestoreReviewsCollection: getEnv("FIRESTORE_REVIEWS_COLLECTION", "reviews"),
		CloudTasksProjectID:        cloudTasksProjectID,
		CloudTasksLocation:         getEnv("CLOUD_TASKS_LOCATION", "europe-north1"),
		CloudTasksQueue:            getEnv("CLOUD_TASKS_QUEUE", "lead-processing"),
		InternalTaskToken:          getEnv("INTERNAL_TASK_TOKEN", ""),
		InternalProcessLeadURL:     processLeadURL,
		TasksServiceAccountEmail:   getEnv("TASKS_SERVICE_ACCOUNT_EMAIL", ""),
		AdminPassword:              getEnv("ADMIN_PASSWORD", ""),
		ReviewTokenSecret:          getEnv("REVIEW_TOKEN_SECRET", ""),
		SiteSettingsCollection:     getEnv("SITE_SETTINGS_COLLECTION", "site_settings"),

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		MailFrom:     getEnv("MAIL_FROM", ""),
		MailReplyTo:  getEnv("MAIL_REPLY_TO", ""),
		MailCopyTo:   getEnv("MAIL_COPY_TO", ""),
	}
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func newServer(cfg Config, renderer *TemplateRenderer) *http.Server {
	return &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: liikkuvaPro(cfg, liikkuvaSecurity(cfg, loggingMiddleware(newRouter(cfg, renderer)))),
	}
}
