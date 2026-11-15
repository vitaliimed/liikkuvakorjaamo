package main

import (
    "os"
    "strings"
    "testing"
)

func TestConfirmationExplainsVisitFeeAndDiagnosticsSeparately(t *testing.T) {
    cfg := Config{SiteURL: "https://liikkuvakorjaamo.fi"}
    for _, lang := range []string{"fi", "en"} {
        lead := Lead{
            Name: "Client", Car: "ABC-123", Language: lang,
            VisitPriceEUR: 92, DiagnosticFromEUR: 78,
        }
        plain := buildLeadConfirmationText(cfg, lead)
        branded := buildLeadConfirmationHTML(cfg, lead)
        for _, body := range []string{plain, branded} {
            if !strings.Contains(body, "92 €") && !strings.Contains(body, "€92") {
                t.Errorf("%s: missing frozen visit quote 92: %s", lang, body)
            }
            if !strings.Contains(body, "78 €") && !strings.Contains(body, "€78") {
                t.Errorf("%s: missing frozen diagnostic quote 78: %s", lang, body)
            }
            if strings.Contains(body, "75 €") || strings.Contains(body, "60 €") {
                t.Errorf("%s: old quote leaked into client email", lang)
            }
        }
        if lang == "fi" {
            for _, label := range []string{"Automekaanikon käynti alkaen", "vianmäärityksen", "sovittuna ajankohtana"} {
                if !strings.Contains(strings.ToLower(plain), strings.ToLower(label)) {
                    t.Errorf("%s: missing %q in confirmation", lang, label)
                }
            }
        } else {
            for _, label := range []string{"Mechanic visit from", "Diagnostics and minor repairs", "at the agreed time"} {
                if !strings.Contains(plain, label) {
                    t.Errorf("%s: missing %q in confirmation", lang, label)
                }
            }
        }
    }
}

func TestStartingPriceChangeUpdatesWebsiteWithoutOldFee(t *testing.T) {
    configured := SiteSettings{ProVisitEUR: 95, ProDiagnosticEUR: 83}
    for _, lang := range []string{"fi", "en"} {
        content := getLandingContent(lang)
        for _, raw := range []string{
            content.Form.VisitPriceTitle,
            content.Form.VisitPriceFootnote,
            content.FAQ.Items[0].Answer,
        } {
            updated := liikkuvaPriceText(raw, configured)
            if strings.Contains(updated, "75 €") || strings.Contains(updated, "{diagnostic}") {
                t.Errorf("%s: stale price or unrendered placeholder: %s", lang, updated)
            }
        }
    }
    if got := liikkuvaPriceText("75 € / €75 / 50 € / €50 / {diagnostic}", configured); got != "95 € / €95 / 83 € / €83 / 83" {
        t.Fatalf("dynamic pricing mismatch: %s", got)
    }
}

func TestPriceCopyDoesNotPromiseRepairAsPartOfVisit(t *testing.T) {
    for _, lang := range []string{"fi", "en"} {
        content := getLandingContent(lang)
        footnote := strings.ToLower(content.Form.VisitPriceFootnote)
        if !strings.Contains(footnote, "erikseen") && !strings.Contains(footnote, "in addition") {
            t.Errorf("%s: diagnostic must be explicitly charged separately: %s", lang, footnote)
        }
    }
}

func TestSiteDisplaysMeetAtLocationOnlyAfterTimeConfirmed(t *testing.T) {
    for _, path := range []string{"landing.html", "templates/landing.html"} {
        data, err := os.ReadFile(path)
        if err != nil {
            t.Fatal(err)
        }
        body := string(data)
        for _, want := range []string{
            "visit-arrival-note", "footer-arrival-reminder",
            "Once we confirm your appointment", "Kun olemme vahvistaneet käyntiajan",
            "vianmääritys", "Diagnostics and minor repairs",
        } {
            if !strings.Contains(strings.ToLower(body), strings.ToLower(want)) {
                t.Errorf("%s: missing %q", path, want)
            }
        }
    }
}

func TestDiagnosticFeeSnapshotsOnLead(t *testing.T) {
    if got := liikkuvaLeadDiagnosticFee(Lead{DiagnosticFromEUR: 92}); got != 92 {
        t.Fatalf("stored diagnostic fee changed: %d", got)
    }
    if got := liikkuvaLeadDiagnosticFee(Lead{}); got != 60 {
        t.Fatalf("fallback diagnostic fee must be 60: %d", got)
    }
}
