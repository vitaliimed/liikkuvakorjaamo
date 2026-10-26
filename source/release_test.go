package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestLeadDoesNotRequireVisitPriceConsent(t *testing.T) {
	valid := Lead{
		Name:     "Test Client",
		Phone:    "+358401234567",
		Email:    "client@example.com",
		Language: "en",
	}
	if errors := validateLead(valid); len(errors) != 0 {
		t.Fatalf("valid lead rejected: %#v", errors)
	}
}

func TestReviewTokenAndValidation(t *testing.T) {
	cfg := Config{ReviewTokenSecret: "test-only-secret", SiteURL: "https://example.test"}
	token, err := buildReviewToken(cfg, "lead-123", "Client@Example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyReviewToken(cfg, "lead-123", "client@example.com", token) {
		t.Fatal("valid review token rejected")
	}
	if verifyReviewToken(cfg, "lead-124", "client@example.com", token) {
		t.Fatal("modified lead ID accepted")
	}
	if errors := validateReview("Client", "Excellent mobile repair service.", 5, true, "en"); len(errors) != 0 {
		t.Fatalf("valid review rejected: %#v", errors)
	}
	if errors := validateReview("Client", "Excellent mobile repair service.", 5, false, "en"); errors["publish_consent"] == "" {
		t.Fatal("review without publication consent accepted")
	}
}

func TestReviewLinkUsesCurrentTaggedRevision(t *testing.T) {
	cfg := Config{ReviewTokenSecret: "test-only-secret", SiteURL: "https://liikkuvakorjaamo.fi"}
	req := httptest.NewRequest(http.MethodPost, "https://rel2---liikkuva-landing-example.a.run.app/admin/leads/send-review-email", nil)
	baseURL := publicBaseURLForRequest(cfg, req)
	if baseURL != "https://rel2---liikkuva-landing-example.a.run.app" {
		t.Fatalf("unexpected tagged revision base URL: %s", baseURL)
	}

	reviewURL, err := buildReviewURLWithBase(cfg, baseURL, "lead-123", Lead{Email: "client@example.com", Language: "fi"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(reviewURL, baseURL+"/review?") {
		t.Fatalf("review URL points to the wrong revision: %s", reviewURL)
	}
}

func TestReviewLinkRejectsUnknownHost(t *testing.T) {
	cfg := Config{SiteURL: "https://liikkuvakorjaamo.fi"}
	req := httptest.NewRequest(http.MethodPost, "https://attacker.example/admin", nil)
	if got := publicBaseURLForRequest(cfg, req); got != cfg.SiteURL {
		t.Fatalf("unknown host was accepted: %s", got)
	}
}

func TestReviewAliasAndClientStageHelpers(t *testing.T) {
	if got := normalizeReviewNameMode("alias"); got != "alias" {
		t.Fatalf("alias mode normalized to %q", got)
	}
	if got := normalizeReviewNameMode("unknown"); got != "real" {
		t.Fatalf("unknown name mode normalized to %q", got)
	}

	records := []LeadRecord{{ClientStatus: "new"}, {ClientStatus: "scheduled"}, {ClientStatus: "new"}}
	filtered := filterLeadRecordsByClientStatus(records, "new")
	if len(filtered) != 2 {
		t.Fatalf("client stage filter returned %d records", len(filtered))
	}
}

func TestConfirmationEmailContainsRequiredMessage(t *testing.T) {
	body := buildLeadConfirmationText(Config{SiteURL: "https://liikkuvakorjaamo.fi"}, Lead{Name: "Matti", Language: "fi"})
	for _, required := range []string{"Kiitos pyynnöstäsi", strconv.Itoa(visitPriceEUR) + " €", "Odota viestiämme"} {
		if !strings.Contains(body, required) {
			t.Fatalf("confirmation email is missing %q", required)
		}
	}
}

func TestYellowPriceCardsWereRemoved(t *testing.T) {
	landing, err := os.ReadFile("landing.html")
	if err != nil {
		t.Fatal(err)
	}
	thankYou, err := os.ReadFile("thankyou.html")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(landing), "visit-price-card") {
		t.Fatal("large yellow price card is still present in the lead form")
	}
	for _, removed := range []string{"visit-price-compact", "visitPriceAccepted", "visit_price_accepted"} {
		if strings.Contains(string(landing), removed) {
			t.Fatalf("lead form still contains removed visit-price control %q", removed)
		}
	}
	if strings.Contains(string(thankYou), "thankyou-price-notice") {
		t.Fatal("yellow price notice is still present on the thank-you page")
	}
}

// The original visual layout test is obsolete: the user requested a different layout.

func TestReviewAndMobileAdminFormsRender(t *testing.T) {
	renderer, err := newTemplateRenderer()
	if err != nil {
		t.Fatal(err)
	}

	reviewRecorder := httptest.NewRecorder()
	err = renderer.renderReview(reviewRecorder, ReviewPageData{
		Lang:     "fi",
		LeadID:   "lead-123",
		Token:    "token-123",
		Name:     "Matti",
		NameMode: "real",
		Rating:   5,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"name_mode", "reviewAlias", "publish_consent", "Lähetä arvostelu"} {
		if !strings.Contains(reviewRecorder.Body.String(), required) {
			t.Fatalf("review form is missing %q", required)
		}
	}

	adminRecorder := httptest.NewRecorder()
	err = renderer.renderAdmin(adminRecorder, AdminPageData{
		PageTitle:          "Admin",
		MailEnabled:        true,
		ActiveTab:          "leads",
		ClientStatusFilter: "all",
		SelectedLead: &AdminLeadDetails{
			ID:                      "lead-123",
			Name:                    "Matti",
			Phone:                   "+358401234567",
			Email:                   "matti@example.com",
			Car:                     "ABC-123",
			Address:                 "Helsinki",
			MapsURL:                 "https://maps.example.test",
			Message:                 "Brake repair",
			ClientStatus:            "new",
			ConfirmationEmailStatus: "pending",
			ReadyEmailStatus:        "not_sent",
			ReviewInviteEmailStatus: "not_sent",
			ConfirmationEmailError:  "—",
			ReviewInviteEmailError:  "—",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"lead-workspace", "update-status", "send-confirmation-email", "send-custom-email", "Текст письма клиенту", "mobile-lead-list"} {
		if !strings.Contains(adminRecorder.Body.String(), required) {
			t.Fatalf("mobile admin workflow is missing %q", required)
		}
	}
}

func TestOnlyApprovedConsentedReviewsBecomeCards(t *testing.T) {
	records := []ReviewRecord{
		{Name: "Approved", Message: "Excellent service", Rating: 5, Language: "en", Status: reviewStatusApproved, PublishConsent: true},
		{Name: "Pending", Message: "Pending review", Rating: 5, Language: "en", Status: reviewStatusPending, PublishConsent: true},
		{Name: "No consent", Message: "Private review", Rating: 5, Language: "en", Status: reviewStatusApproved, PublishConsent: false},
		{Name: "Finnish", Message: "Hyvä palvelu", Rating: 5, Language: "fi", Status: reviewStatusApproved, PublishConsent: true},
	}

	cards := approvedReviewCards(records, "en", 12)
	if len(cards) != 1 || cards[0].Author != "Approved" {
		t.Fatalf("unexpected cards: %#v", cards)
	}
}

func TestTemplatesAndRequiredStaticFiles(t *testing.T) {
	if _, err := newTemplateRenderer(); err != nil {
		t.Fatalf("template parsing failed: %v", err)
	}

	required := []string{
		"static/css/style.css",
		"static/js/app.js",
		"static/js/consent.js",
		"static/img/logo_liikkuva.svg",
		"static/img/hero2.webp",
		"static/img/pic/uusimaa1.webp",
	}
	for _, path := range required {
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			t.Errorf("required file %s is missing or empty", path)
		}
	}
}

func TestPublicPagesRenderWithoutCloudConfiguration(t *testing.T) {
	renderer, err := newTemplateRenderer()
	if err != nil {
		t.Fatal(err)
	}
	router := newRouter(Config{SiteURL: "https://example.test", DefaultLanguage: "fi"}, renderer)

	for _, path := range []string{"/", "/en", "/thank-you-fi", "/thank-you-en", "/privacy?lang=fi", "/robots.txt", "/sitemap.xml", "/review?submitted=1&lang=en"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Errorf("GET %s returned %d", path, recorder.Code)
		}
		if recorder.Body.Len() == 0 {
			t.Errorf("GET %s returned an empty body", path)
		}
	}
}
