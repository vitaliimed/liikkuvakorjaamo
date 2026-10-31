package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type memorySiteAssets struct {
	mu    sync.Mutex
	files map[string][]byte
	kinds map[string]string
}

func (s *memorySiteAssets) Put(_ context.Context, name, kind string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.files == nil {
		s.files = map[string][]byte{}
		s.kinds = map[string]string{}
	}
	s.files[name] = append([]byte(nil), data...)
	s.kinds[name] = kind
	return nil
}
func (s *memorySiteAssets) Get(_ context.Context, name string) ([]byte, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.files[name]
	if !ok {
		return nil, "", fmt.Errorf("missing")
	}
	return data, s.kinds[name], nil
}
func testAssetPNG(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestSiteImageUploadUsesAuthenticatedCSRFAndOnlyPublishesAssetPrefix(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	assets := &memorySiteAssets{}
	f.cfg.AssetStore = assets
	renderer, _ := newTemplateRenderer()
	f.handler = newServer(f.cfg, renderer).Handler
	cookie := f.login(t, fixtureAdminPassword)
	imageData := testAssetPNG(t)
	upload := func(csrf, origin string, data []byte) *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		writer.WriteField("access_csrf", csrf)
		file, _ := writer.CreateFormFile("asset", "../../private-client.png")
		file.Write(data)
		writer.Close()
		r := httptest.NewRequest("POST", f.cfg.SiteURL+"/admin/assets/upload", &body)
		r.Header.Set("Content-Type", writer.FormDataContentType())
		r.Header.Set("Origin", origin)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		return w
	}
	for _, attempt := range []struct{ csrf, origin string }{{"", f.cfg.SiteURL}, {fixtureCSRF(cookie), "https://foreign.example"}} {
		if w := upload(attempt.csrf, attempt.origin, imageData); w.Code != 403 {
			t.Fatalf("unsafe upload accepted: %d", w.Code)
		}
	}
	if w := upload(fixtureCSRF(cookie), f.cfg.SiteURL, []byte(`<script>alert('not an image')</script>`)); w.Code != 400 {
		t.Fatalf("HTML accepted: %d", w.Code)
	}
	w := upload(fixtureCSRF(cookie), f.cfg.SiteURL, imageData)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var response struct{ URL string }
	json.Unmarshal(w.Body.Bytes(), &response)
	hash := sha256.Sum256(imageData)
	expected := "/media/site/" + hex.EncodeToString(hash[:]) + ".png"
	if response.URL != expected {
		t.Fatalf("unsafe or unstable image path: %s", response.URL)
	}
	public := f.request("GET", expected, nil, nil)
	if public.Code != 200 || !bytes.Equal(public.Body.Bytes(), imageData) || !strings.Contains(public.Header().Get("Cache-Control"), "immutable") {
		t.Fatal("public asset did not survive an unauthenticated read")
	}
	for _, path := range []string{"/media/site/lead-photos/client.png", "/media/site/client.png", "/media/site/" + strings.Repeat("a", 64) + ".html"} {
		if f.request("GET", path, nil, nil).Code == 200 {
			t.Fatal("private or executable path exposed")
		}
	}
	if f.request("GET", expected, nil, nil).Header().Get("Content-Type") != "image/png" {
		t.Fatal("unsafe content type")
	}
}
func TestStaticSVGRejectsScriptsExternalResourcesCSSAndEntities(t *testing.T) {
	good := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#2458d6" d="M0 0h24v24H0z"/></svg>`)
	if _, kind, err := validateSiteAsset(good); err != nil || kind != "image/svg+xml" {
		t.Fatal(err)
	}
	for _, svg := range []string{`<svg><script>alert(1)</script></svg>`, `<svg onload="bad()"/>`, `<svg><use href="https://external.example/a.svg#x"/></svg>`, `<svg><foreignObject><div>HTML</div></foreignObject></svg>`, `<svg><style>body{display:none}</style></svg>`, `<!DOCTYPE svg [<!ENTITY x SYSTEM "file:///etc/passwd">]><svg>&x;</svg>`, `<svg><path fill="url(https://evil.example)"/></svg>`} {
		if validateStaticSVG([]byte(svg)) == nil {
			t.Fatalf("active SVG accepted: %s", svg)
		}
	}
}
func TestBrandEmailsUseSavedTextsFeeSnapshotAndEscapeCustomerContent(t *testing.T) {
	s := SiteSettings{ProAccent: "#2458d6", EmailLogoURL: "/media/site/brand.png", EmailConfirmationTextEN: "Thank you, {name}. Please wait for our manager's call. Visit {visit}.", EmailReadyTextFI: "Autosi {car} on valmis.", EmailSignatureEN: "Liikkuva team"}
	cfg := Config{SiteURL: "https://liikkuvakorjaamo.fi", MailFrom: "service@liikkuvakorjaamo.fi", EmailSettings: &s}
	lead := Lead{Name: `<script>customer</script>`, Car: "ABC-123", Language: "en", VisitPriceEUR: 95}
	text := buildLeadConfirmationText(cfg, lead)
	if !strings.Contains(text, "Visit 95") || !strings.Contains(text, "€95") || !strings.Contains(text, "manager's call") {
		t.Fatal("wrong fee snapshot or configured message")
	}
	for _, body := range []string{buildLeadConfirmationHTML(cfg, lead), buildReadyEmailHTMLBody(cfg, lead), buildCustomEmailHTMLBody(cfg, lead, "Status", `<script>custom</script>`), brandedPlainEmailHTML(cfg, "ru", "Доступ", `<script>system</script>`)} {
		for _, required := range []string{"#2458d6", "AUTO KORJAAMO", "https://liikkuvakorjaamo.fi/media/site/brand.png", "service@liikkuvakorjaamo.fi"} {
			if !strings.Contains(body, required) {
				t.Fatalf("brand missing: %s", required)
			}
		}
		if strings.Contains(body, "<script>") {
			t.Fatal("unescaped customer content")
		}
	}
	lead.Language = "fi"
	if !strings.Contains(buildCustomEmailTextBody(cfg, lead, "Viesti"), "Hei ") || !strings.Contains(buildReadyEmailHTMLBody(cfg, lead), "Autosi ABC-123 on valmis") {
		t.Fatal("wrong Finnish email language")
	}
}
func TestAdminCanSaveWithoutPhoneAndShowsActualGA4Defaults(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	settings := &memorySettingsStore{}
	f.cfg.SettingsStore = settings
	renderer, _ := newTemplateRenderer()
	f.handler = newServer(f.cfg, renderer).Handler
	cookie := f.login(t, fixtureAdminPassword)
	values := url.Values{"access_csrf": {fixtureCSRF(cookie)}, "settings_revision": {"0"}, "active_tab": {"analytics"}, "analytics_events_present": {"1"}, "analytics_event": {"generate_lead", "phone_click"}, "phone": {""}, "email_ready_text_fi": {"Valmis {car}"}, "logo_dark_url": {"/static/img/favicon.svg"}}
	if w := f.request("POST", "/admin/save", values, cookie); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	stored, _ := settings.Load(context.Background())
	if stored.Phone != "" || stored.EmailReadyTextFI != "Valmis {car}" || stored.LogoDarkURL == "" {
		t.Fatal("settings did not save without phone")
	}
	w := f.request("GET", "/admin?tab=analytics", nil, cookie)
	if !strings.Contains(w.Body.String(), `value="G-MXX9XCNCRB"`) || !strings.Contains(w.Body.String(), `name="analytics_event"`) {
		t.Fatal("actual analytics settings missing")
	}
	if strings.Contains(w.Body.String(), `name="phone" required`) {
		t.Fatal("admin settings still require a contact phone")
	}
	var config map[string]any
	json.Unmarshal([]byte(strings.TrimPrefix(f.request("GET", "/tracking-config.js", nil, nil).Body.String(), "window.LIIKKUVA_TRACKING=")), &config)
	for _, event := range analyticsEventCatalog(stored) {
		if event.Name == "theme_change" && event.Enabled {
			t.Fatal("disabled event still enabled")
		}
		if event.Name == "generate_lead" && !event.Enabled {
			t.Fatal("saved lead event unexpectedly disabled")
		}
	}
	if f.request("GET", "/admin/email-preview?kind=ready&lang=fi", nil, cookie).Code != 200 {
		t.Fatal("preview unavailable")
	}
}

func TestTwoFormsUseUniqueMarkupAndReviewsStayVisible(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	f.cfg.SettingsStore = &memorySettingsStore{state: SiteSettings{Testimonials: []AdminTestimonialItem{
		{Name: "Testiasiakas 1", QuoteFI: "Testiarvostelu: selkeä yhteydenpito ja sovittu aika.", QuoteEN: "Sample review: clear communication and an agreed appointment.", Rating: 5},
		{Name: "Testiasiakas 2", QuoteFI: "Testiarvostelu: auton tilanne käytiin ymmärrettävästi läpi.", QuoteEN: "Sample review: the condition of the car was explained clearly.", Rating: 4},
		{Name: "Testiasiakas 3", QuoteFI: "Testiarvostelu: palvelu tuli auton luokse.", QuoteEN: "Sample review: the service came to the vehicle.", Rating: 5},
	}}}
	renderer, err := newTemplateRenderer()
	if err != nil {
		t.Fatal(err)
	}
	f.handler = newServer(f.cfg, renderer).Handler
	for _, language := range []string{"fi", "en"} {
		target := "/"
		if language == "en" {
			target = "/en"
		}
		response := f.request("GET", target, nil, nil)
		if response.Code != 200 {
			t.Fatal(response.Code)
		}
		body := response.Body.String()
		if strings.Count(body, "data-lead-form") != 2 || strings.Contains(body, "request-details") {
			t.Fatal("forms must both render without collapsed fields")
		}
		for _, id := range []string{"leadForm", "leadFormBottom", "phone", "bottom-phone", "car", "bottom-car", "mapPickerModal", "bottom-mapPickerModal"} {
			if strings.Count(body, `id="`+id+`"`) != 1 {
				t.Fatal("missing or repeated form ID: " + id)
			}
		}
		start := strings.Index(body, `<section class="section reviews">`)
		if start < 0 {
			t.Fatal("approved review cards are missing")
		}
		end := strings.Index(body[start:], "</section>") + start
		reviews := body[start:end]
		if strings.Count(reviews, `class="testimonial-card"`) != 3 || strings.Contains(reviews, "<details") || strings.Contains(reviews, "data-testimonial-slider") {
			t.Fatal("reviews should be directly visible cards")
		}
		if dir := os.Getenv("LIIKKUVA_QA_DIR"); dir != "" {
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "landing-reviews-"+language+".html"), []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, plate := range []string{"ABC-1", "ABC-12", "AB-123", "ABCD-123"} {
		if isValidRegistrationNumber(plate) {
			t.Fatal("registration must contain three letters and three digits: " + plate)
		}
	}
	if !isValidRegistrationNumber("ABC-123") {
		t.Fatal("valid masked registration rejected")
	}
}
