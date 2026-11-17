package main

import (
	"liikkuvakorjaamo.fi/landing/crm"
	"strings"
	"testing"
)

func TestSiteAndCRMRegistrationPolicy(t *testing.T) {
	for _, input := range []string{"abc123", "ABC-123", "123abc", "123-abc", "123 abc"} {
		car := normalizeRegistrationNumber(input)
		key, err := crm.Plate(input)
		if !isValidRegistrationNumber(car) || err != nil || strings.ReplaceAll(car, "-", "") != key {
			t.Fatalf("site/CRM disagree: %q %q %q %v", input, car, key, err)
		}
		for _, lang := range []string{"fi", "en"} {
			if msg := validateLead(Lead{Name: "Test", Phone: "+358401234567", Car: car, Language: lang})["car"]; msg != "" {
				t.Fatal(msg)
			}
		}
	}
	for _, input := range []string{"ABC1", "ABC12", "ABC1234", "1234ABC", "A1B2C3", "АВС123", "ABC-123!"} {
		if isValidRegistrationNumber(normalizeRegistrationNumber(input)) {
			t.Fatal(input)
		}
		if _, err := crm.Plate(input); err == nil {
			t.Fatal(input)
		}
	}
}
func TestPersistedRegistrationHints(t *testing.T) {
	for _, lang := range []string{"fi", "en"} {
		s := SiteSettings{FormCarHelpFI: "ABC-123", FormCarHelpEN: "ABC-123", FormRegistrationInvalidFI: "ABC-123", FormRegistrationInvalidEN: "ABC-123"}
		c := applySiteSettingsToContent(getLandingContent(lang), s, lang)
		if !strings.Contains(c.Form.CarHelp, "123-ABC") || !strings.Contains(c.Form.RegistrationInvalidMessage, "123-ABC") {
			t.Fatal(lang, c.Form.CarHelp)
		}
	}
}
