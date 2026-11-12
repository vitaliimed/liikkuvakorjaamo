package main

import (
    "os"
    "strings"
    "testing"
)

// The landing template contains two language branches; each must have exactly
// one fully functional form, rendered inside the hero before the lower sections.
func TestHeroFormAndRequiredFieldsInBothLanguages(t *testing.T) {
    b, err := os.ReadFile("templates/landing.html")
    if err != nil { t.Fatal(err) }
    html := string(b)
    expected := map[string]int{
        `class="hero-form" id="lead-form"`: 2,
        `id="leadForm"`: 2,
        `id="request-more"`: 2,
        `name="name" placeholder=`: 2,
        `name="phone" required`: 2,
        `name="email" required`: 2,
        `name="address" required`: 2,
        `name="car" required`: 2,
        `id="photosTrigger"`: 2,
        `data-theme-toggle`: 2,
    }
    for needle, want := range expected {
        if got := strings.Count(html, needle); got != want {
            t.Errorf("unexpected occurrences of %q: got %d, want %d", needle, got, want)
        }
    }
    if strings.Index(html, `class="hero-form"`) > strings.Index(html, `class="promise-bar"`) {
        t.Error("form must be present before the first content section")
    }
    if !strings.Contains(html, `/static/css/hero-form-ux.css`) {
        t.Error("compact UX stylesheet is not linked")
    }
    if !strings.Contains(html, `/static/js/theme.js`) {
        t.Error("theme toggle script is not linked")
    }
    if !strings.Contains(html, `property="og:image" content="https://liikkuvakorjaamo.fi/static/img/logo-white.png`) {
        t.Error("social share image must use full logo, not hero photograph")
    }
}

func TestAutofillAndMetroMapConfiguration(t *testing.T) {
    b, err := os.ReadFile("static/js/app.js")
    if err != nil { t.Fatal(err) }
    src := string(b)
    for _, needle := range []string{
        "liikkuva_lead_autofill_v2",
        "liikkuva_problem_draft_v2",
        "localStorage.setItem(draftKey",
        "sessionStorage.setItem(problemDraftKey",
        "[60.06, 24.48]",
        "[60.40, 25.38]",
    } {
        if !strings.Contains(src, needle) { t.Errorf("missing %s", needle) }
    }
}

func TestPreserveCurrentVisitPrice(t *testing.T) {
    if visitPriceEUR != 75 {
        t.Fatalf("live visit price must remain 75 EUR, got %d", visitPriceEUR)
    }
    b, err := os.ReadFile("liikkuva-content.json")
    if err != nil { t.Fatal(err) }
    if strings.Contains(string(b), "70 €") || strings.Contains(string(b), "€70") {
        t.Fatal("old 70 EUR price remains in the default public content")
    }
}
