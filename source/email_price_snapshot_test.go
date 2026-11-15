package main

import (
	"strings"
	"testing"
)

// The frontend, Firestore record and both MIME alternatives must refer to
// the same quoted prices. The deployed default fee must never override a
// snapshot saved when the customer submitted the form.
func TestConfirmationUsesFrozenPrices(t *testing.T) {
	cfg := Config{SiteURL: "https://liikkuvakorjaamo.fi"}
	for _, language := range []string{"fi", "en"} {
		lead := Lead{
			Name:              "Test",
			Email:             "customer@example.com",
			Language:          language,
			VisitPriceEUR:     50,
			DiagnosticFromEUR: 50,
		}
		for _, rendered := range []string{
			buildLeadConfirmationText(cfg, lead),
			buildLeadConfirmationHTML(cfg, lead),
		} {
			if strings.Contains(rendered, "75 €") || strings.Contains(rendered, "€75") ||
				strings.Contains(rendered, "60 €") || strings.Contains(rendered, "€60") {
				t.Errorf("%s: stale default price leaked into email: %s", language, rendered)
			}
			if !strings.Contains(rendered, "50 €") && !strings.Contains(rendered, "€50") {
				t.Errorf("%s: saved 50-euro price missing: %s", language, rendered)
			}
		}
	}
}

func TestConfirmationRefusesMissingPriceSnapshot(t *testing.T) {
	// SMTP settings are nonempty so the missing-snapshot check runs, but no
	// network call is permitted or needed for this test.
	cfg := Config{
		SMTPHost: "invalid.example", SMTPPort: "465",
		SMTPUsername: "test", SMTPPassword: "test",
		MailFrom: "service@example.com",
	}
	for _, lead := range []Lead{
		{Email: "customer@example.com", VisitPriceEUR: 0, DiagnosticFromEUR: 50},
		{Email: "customer@example.com", VisitPriceEUR: 50, DiagnosticFromEUR: 0},
	} {
		err := sendLeadConfirmationEmail(cfg, lead)
		if err == nil || !strings.Contains(err.Error(), "price is missing") {
			t.Fatalf("missing price snapshot was not rejected: lead=%+v, err=%v", lead, err)
		}
	}
}

func TestHistoricalQuotedPricesAreNotSilentlyRewritten(t *testing.T) {
	// Re-sending an earlier lead must use its original quote. Current admin
	// prices apply to new leads, not retroactively to past commitments.
	oldLead := Lead{
		Name: "Test", Language: "fi",
		VisitPriceEUR: 75, DiagnosticFromEUR: 60,
	}
	body := buildLeadConfirmationText(Config{}, oldLead)
	if !strings.Contains(body, "75 €") || !strings.Contains(body, "60 €") {
		t.Fatalf("historic quote changed unexpectedly: %s", body)
	}
}
