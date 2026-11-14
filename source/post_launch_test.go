package main

import (
    "os"
    "strings"
    "testing"
)

func TestPostLaunchThemeScriptSingleRegistration(t *testing.T) {
    old := `<html><head><script src="/static/js/theme.js?v=already-present" defer></script></head><body></body></html>`
    output := liikkuvaThemeHTML(old, SiteSettings{})
    if count := strings.Count(output, "/static/js/theme.js"); count != 1 {
        t.Fatalf("landing theme loaded %d times, expected one", count)
    }
    noScript := liikkuvaThemeHTML("<html><head></head><body></body></html>", SiteSettings{})
    if count := strings.Count(noScript, "/static/js/theme.js"); count != 1 {
        t.Fatalf("other pages need one theme controller, got %d", count)
    }
}

func TestPostLaunchHeroBrokenLegacyUploadFallsBack(t *testing.T) {
    source := getLandingContent("fi")
    if !strings.HasPrefix(source.Hero.BackgroundURL, "/static/img/") {
        t.Fatalf("expected bundled default hero image, got %q", source.Hero.BackgroundURL)
    }
    updated := applySiteSettingsToContent(source, SiteSettings{
        HeroBackgroundURL:"/media/site/16944e9d2e322b1b0b9abe3cbfd4d4c5efff469df750fedc32dcf6a175da43.png",
    }, "fi")
    if updated.Hero.BackgroundURL != source.Hero.BackgroundURL {
        t.Fatalf("unsupported legacy media path broke public hero: %q", updated.Hero.BackgroundURL)
    }
    custom := applySiteSettingsToContent(source, SiteSettings{
        HeroBackgroundURL:"/static/img/hero2.webp",
    }, "fi")
    if custom.Hero.BackgroundURL != "/static/img/hero2.webp" {
        t.Fatalf("valid admin hero image incorrectly rejected: %q", custom.Hero.BackgroundURL)
    }
}

func TestPostLaunchTwoFormsAndFooterJS(t *testing.T) {
    for _, file := range []string{"templates/landing.html", "landing.html"} {
        html, err := os.ReadFile(file)
        if err != nil {t.Fatal(err)}
        source := string(html)
        for _, marker := range []string{
            `id="leadForm"`, `id="footerLeadFormMount"`,
            "/static/js/footer-form.js", "/static/css/footer-form.css",
            "Mekaanikoillamme vähintään 15 vuoden työkokemus",
            "Mechanics with at least 15 years of experience",
        } {
            if !strings.Contains(source, marker) {
                t.Fatalf("%s missing %s", file, marker)
            }
        }
        if n := strings.Count(source, `id="footerLeadFormMount"`); n != 2 {
            t.Fatalf("%s needs a mount in each language branch: %d", file, n)
        }
    }
}

func TestPostLaunchAdminSettingsPhoneNotMandatory(t *testing.T) {
    for _, file := range []string{"templates/admin.html", "admin.html"} {
        html, err := os.ReadFile(file)
        if err != nil {t.Fatal(err)}
        if strings.Contains(string(html), `name="phone" required`) {
            t.Fatalf("%s reintroduced browser-blocking phone requirement", file)
        }
    }
}
