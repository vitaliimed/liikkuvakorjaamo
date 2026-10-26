package main

import "time"

const visitPriceEUR = 70

type Lead struct {
	DiagnosticFromEUR  int      `json:"diagnostic_from_eur" firestore:"diagnostic_from_eur"`
	Site               string   `json:"site" firestore:"site"`
	SubmissionID       string   `json:"submission_id" firestore:"submission_id"`
	Name               string   `json:"name" firestore:"name"`
	Phone              string   `json:"phone" firestore:"phone"`
	Email              string   `json:"email" firestore:"email"`
	Address            string   `json:"address" firestore:"address"`
	AddressHint        string   `json:"address_hint" firestore:"address_hint"`
	LocationSource     string   `json:"location_source" firestore:"location_source"`
	Latitude           string   `json:"latitude" firestore:"latitude"`
	Longitude          string   `json:"longitude" firestore:"longitude"`
	MapsURL            string   `json:"maps_url" firestore:"maps_url"`
	Car                string   `json:"car" firestore:"car"`
	Message            string   `json:"message" firestore:"message"`
	VisitPriceEUR      int      `json:"visit_price_eur" firestore:"visit_price_eur"`
	VisitPriceAccepted bool     `json:"visit_price_accepted" firestore:"visit_price_accepted"`
	PhotoURLs          []string `json:"photo_urls,omitempty" firestore:"photo_urls,omitempty"`
	PhotoPaths         []string `json:"-" firestore:"-"`
	Language           string   `json:"language" firestore:"language"`
	Source             string   `json:"source" firestore:"source"`
	PageURL            string   `json:"page_url" firestore:"page_url"`
	Referer            string   `json:"referer" firestore:"referer"`
	UTMSource          string   `json:"utm_source" firestore:"utm_source"`
	UTMMedium          string   `json:"utm_medium" firestore:"utm_medium"`
	UTMCampaign        string   `json:"utm_campaign" firestore:"utm_campaign"`
	UTMTerm            string   `json:"utm_term" firestore:"utm_term"`
	UTMContent         string   `json:"utm_content" firestore:"utm_content"`
	UserAgent          string   `json:"user_agent" firestore:"user_agent"`
	IP                 string   `json:"ip" firestore:"ip"`
	CreatedAt          string   `json:"created_at" firestore:"created_at"`
	ScriptToken        string   `json:"script_token,omitempty" firestore:"script_token,omitempty"`
}

type LeadRecord struct {
	ID                      string    `json:"id,omitempty" firestore:"-"`
	Lead                    Lead      `json:"lead" firestore:"lead"`
	ProcessingStatus        string    `json:"processing_status" firestore:"processing_status"`
	TelegramStatus          string    `json:"telegram_status" firestore:"telegram_status"`
	SheetsStatus            string    `json:"sheets_status" firestore:"sheets_status"`
	ClientStatus            string    `json:"client_status" firestore:"client_status"`
	ReadyEmailStatus        string    `json:"ready_email_status" firestore:"ready_email_status"`
	ReadyEmailError         string    `json:"ready_email_error,omitempty" firestore:"ready_email_error,omitempty"`
	ConfirmationEmailStatus string    `json:"confirmation_email_status" firestore:"confirmation_email_status"`
	ConfirmationEmailError  string    `json:"confirmation_email_error,omitempty" firestore:"confirmation_email_error,omitempty"`
	ReviewInviteEmailStatus string    `json:"review_invite_email_status" firestore:"review_invite_email_status"`
	ReviewInviteEmailError  string    `json:"review_invite_email_error,omitempty" firestore:"review_invite_email_error,omitempty"`
	LastError               string    `json:"last_error,omitempty" firestore:"last_error,omitempty"`
	CreatedAt               time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt               time.Time `json:"updated_at" firestore:"updated_at"`
	TaskEnqueuedAt          time.Time `json:"task_enqueued_at,omitempty" firestore:"task_enqueued_at,omitempty"`
	ProcessedAt             time.Time `json:"processed_at,omitempty" firestore:"processed_at,omitempty"`
	ReadyEmailSentAt        time.Time `json:"ready_email_sent_at,omitempty" firestore:"ready_email_sent_at,omitempty"`
	ConfirmationEmailSentAt time.Time `json:"confirmation_email_sent_at,omitempty" firestore:"confirmation_email_sent_at,omitempty"`
	ReviewInviteEmailSentAt time.Time `json:"review_invite_email_sent_at,omitempty" firestore:"review_invite_email_sent_at,omitempty"`
}

type ReviewRecord struct {
	ID             string    `json:"id,omitempty" firestore:"-"`
	LeadID         string    `json:"lead_id" firestore:"lead_id"`
	Name           string    `json:"name" firestore:"name"`
	Rating         int       `json:"rating" firestore:"rating"`
	Message        string    `json:"message" firestore:"message"`
	Language       string    `json:"language" firestore:"language"`
	PublishConsent bool      `json:"publish_consent" firestore:"publish_consent"`
	Status         string    `json:"status" firestore:"status"`
	CreatedAt      time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" firestore:"updated_at"`
	ModeratedAt    time.Time `json:"moderated_at,omitempty" firestore:"moderated_at,omitempty"`
}

type LeadTaskPayload struct {
	LeadID string `json:"lead_id"`
}

type APIResponse struct {
	SubmissionID string            `json:"submission_id,omitempty"`
	OK           bool              `json:"ok"`
	Message      string            `json:"message"`
	RedirectURL  string            `json:"redirect_url,omitempty"`
	Errors       map[string]string `json:"errors,omitempty"`
}

type AppsScriptResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Sheet   string `json:"sheet,omitempty"`
	Message string `json:"message,omitempty"`
}
