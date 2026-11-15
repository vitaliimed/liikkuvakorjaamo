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
    if got := liikkuvaPriceText("175 € / €750 / 150 € / €500 / 75 € / €75 / 50 € / €50", configured); got != "175 € / €750 / 150 € / €500 / 95 € / €95 / 83 € / €83" {
        t.Fatalf("partial price substitution corrupts unrelated amounts: %s", got)
    }
    if got := liikkuvaPriceText("75 € / €75 / 50 € / €50 / {diagnostic}", configured); got != "95 € / €95 / 83 € / €83 / 83" {
        t.Fatalf("dynamic pricing mismatch: %s", got)
    }
}

func TestPriceCopyDoesNotPromiseRepairAsPartOfVisit(t *testing.T) {
    // getLandingContent loads the embedded liikkuva-content.json and overwrites
    // some Go default strings. Test the actual customer-facing content.
    for _, lang := range []string{"fi", "en"} {
        content := getLandingContent(lang)
        if len(content.Services.Items) < 2 || len(content.FAQ.Items) == 0 || len(content.TrustPanel.Items) < 2 {
            t.Fatalf("%s: missing price-related content sections", lang)
        }
        for _, raw := range []string{
            content.Form.VisitPriceText,
            content.Form.VisitPriceFootnote,
            content.FAQ.Items[0].Answer,
        } {
            copy := strings.ToLower(raw)
            if !strings.Contains(copy, "erikseen") && !strings.Contains(copy, "in addition") && !strings.Contains(copy, "separately") {
                t.Errorf("%s: diagnostic fee not explicitly separate from visit: %s", lang, raw)
            }
        }
        if !strings.Contains(content.Services.Items[1].Price, "{diagnostic}") ||
            !strings.Contains(content.TrustPanel.Items[1], "{diagnostic}") {
            t.Errorf("%s: embedded JSON has hard-coded diagnostic price", lang)
        }
        if !strings.Contains(content.FAQ.Items[0].Question, "{visit}") {
            t.Errorf("%s: embedded FAQ has hard-coded mechanic visit price", lang)
        }
        if len(content.Process.Items) < 2 || !strings.Contains(strings.ToLower(content.Process.Items[1].Text), "sovittuna") &&
            !strings.Contains(strings.ToLower(content.Process.Items[1].Text), "agreed time") {
            t.Errorf("%s: missing confirmed-time appointment instructions", lang)
        }
        for _, raw := range []string{content.Form.VisitPriceText, content.Form.VisitPriceFootnote, content.FAQ.Items[0].Answer} {
            rendered := liikkuvaPriceText(raw, SiteSettings{ProVisitEUR: 97, ProDiagnosticEUR: 82})
            if strings.Contains(rendered, "{visit}") || strings.Contains(rendered, "{diagnostic}") {
                t.Errorf("%s: pricing placeholder not resolved: %s", lang, rendered)
            }
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
        if strings.Contains(body, "</li><p class=\"footer-arrival-reminder\">") {
            t.Errorf("%s: footer arrival notice must be outside the unordered list", path)
        }
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
