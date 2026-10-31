package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The in-memory implementation has the same atomic commit/rollback boundary as
// a Firestore transaction. HTTP tests exercise the actual middleware/routes.
type memoryAdminStore struct {
	mu    sync.Mutex
	state adminAuthState
}

func cloneAdminState(state adminAuthState) adminAuthState {
	b, _ := json.Marshal(state)
	var copy adminAuthState
	_ = json.Unmarshal(b, &copy)
	return copy
}
func (s *memoryAdminStore) Load(context.Context) (adminAuthState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneAdminState(s.state), nil
}
func (s *memoryAdminStore) Update(_ context.Context, fn func(*adminAuthState) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := cloneAdminState(s.state)
	if err := fn(&copy); err != nil {
		return err
	}
	s.state = copy
	return nil
}

type capturedAccessMail struct{ To, Subject, Body, Copy string }
type accessFixture struct {
	cfg     Config
	store   *memoryAdminStore
	handler http.Handler
	mailMu  sync.Mutex
	mails   []capturedAccessMail
}

const fixtureAdminPassword = "test-password-long-enough"

func newAccessFixture(t *testing.T, initialPassword string) *accessFixture {
	t.Helper()
	f := &accessFixture{store: &memoryAdminStore{}}
	f.cfg = Config{SiteURL: "https://liikkuvakorjaamo.fi", AdminPassword: initialPassword, AdminRecoveryEmail: "owner@example.test", SiteSettingsCollection: "liikkuva_settings", AdminStore: f.store, SMTPHost: "smtp.example.test", SMTPPort: "587", SMTPUsername: "user", SMTPPassword: "fake-test-key", MailFrom: "service@example.test", MailCopyTo: "copy-must-not-receive-recovery@example.test"}
	f.cfg.AdminSend = func(cfg Config, to, subject, body, html string) error {
		f.mailMu.Lock()
		defer f.mailMu.Unlock()
		f.mails = append(f.mails, capturedAccessMail{To: to, Subject: subject, Body: body, Copy: cfg.MailCopyTo})
		return nil
	}
	renderer, err := newTemplateRenderer()
	if err != nil {
		t.Fatal(err)
	}
	f.handler = newServer(f.cfg, renderer).Handler
	return f
}
func (f *accessFixture) request(method, target string, values url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
	var body io.Reader
	if values != nil {
		body = strings.NewReader(values.Encode())
	}
	r := httptest.NewRequest(method, f.cfg.SiteURL+target, body)
	if method == "POST" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", f.cfg.SiteURL)
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}
func (f *accessFixture) login(t *testing.T, password string) *http.Cookie {
	t.Helper()
	w := f.request("POST", "/admin/login", url.Values{"password": {password}}, nil)
	if w.Code != 303 {
		t.Fatalf("login failed (%d): %s", w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == adminCookieName {
			return c
		}
	}
	t.Fatal("session cookie missing")
	return nil
}
func (f *accessFixture) lastMailToken(t *testing.T) string {
	t.Helper()
	f.mailMu.Lock()
	defer f.mailMu.Unlock()
	if len(f.mails) == 0 {
		t.Fatal("mail not sent")
	}
	body := f.mails[len(f.mails)-1].Body
	token := regexp.MustCompile(`token=([A-Za-z0-9_-]{43})`).FindStringSubmatch(body)
	if len(token) != 2 {
		t.Fatalf("token missing from mail: %s", body)
	}
	return token[1]
}
func fixtureCSRF(cookie *http.Cookie) string {
	r := httptest.NewRequest("GET", "/admin", nil)
	r.AddCookie(cookie)
	return accessCSRF(r)
}

func TestAdminFirstAccessRecoveryAndPasswordSurviveDeployment(t *testing.T) {
	f := newAccessFixture(t, "")
	w := f.request("GET", "/admin", nil, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Забыли пароль") || strings.Contains(w.Body.String(), "ADMIN_PASSWORD is not configured") {
		t.Fatal("first login does not render recovery UI")
	}
	w = f.request("POST", "/admin/recovery", url.Values{}, nil)
	if w.Code != 200 {
		t.Fatalf("recovery failed: %d", w.Code)
	}
	token := f.lastMailToken(t)
	state, _ := f.store.Load(context.Background())
	b, _ := json.Marshal(state)
	if strings.Contains(string(b), token) || strings.Contains(string(b), fixtureAdminPassword) {
		t.Fatal("raw credential/token was persisted")
	}
	if f.mails[0].To != "owner@example.test" || f.mails[0].Copy != "" {
		t.Fatal("recovery went to an unapproved recipient")
	}
	// Opening a link (including an automated email scanner) does not consume it.
	w = f.request("GET", "/admin/reset-password?token="+token, nil, nil)
	if w.Code != 200 {
		t.Fatal("valid token rejected")
	}
	values := url.Values{"token": {token}, "password": {fixtureAdminPassword}, "password_confirm": {fixtureAdminPassword}}
	w = f.request("POST", "/admin/reset-password", values, nil)
	if w.Code != 303 {
		t.Fatalf("reset failed: %d", w.Code)
	}
	if w = f.request("POST", "/admin/reset-password", values, nil); w.Code != 400 {
		t.Fatal("reused token accepted")
	}
	cookie := f.login(t, fixtureAdminPassword)
	// A new instance importing a different/old secret must honor stored access.
	cfg := f.cfg
	cfg.AdminPassword = "old-env-password-cannot-restore-access"
	renderer, _ := newTemplateRenderer()
	f.handler = newServer(cfg, renderer).Handler
	if w = f.request("GET", "/admin/access", nil, cookie); w.Code != 200 {
		t.Fatal("session lost on instance restart")
	}
	if w = f.request("POST", "/admin/login", url.Values{"password": {cfg.AdminPassword}}, nil); w.Code != 401 {
		t.Fatal("environment password overwrote persisted password")
	}
	f.login(t, fixtureAdminPassword)
}
func TestAdminSessionsAreRandomExpireAndRevokeOnPasswordChange(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	a, b := f.login(t, fixtureAdminPassword), f.login(t, fixtureAdminPassword)
	if a.Value == b.Value || len(a.Value) != 43 || !a.HttpOnly || !a.Secure || a.SameSite != http.SameSiteLaxMode {
		t.Fatal("unsafe/reused session cookie")
	}
	state, _ := f.store.Load(context.Background())
	if _, ok := state.Sessions[a.Value]; ok {
		t.Fatal("raw session stored")
	}
	legacy := &http.Cookie{Name: adminCookieName, Value: accessDigest(fixtureAdminPassword)}
	if w := f.request("GET", "/admin/access", nil, legacy); w.Code != 303 {
		t.Fatal("legacy password fingerprint accepted as session")
	}
	newPassword := "a-new-safe-password-123"
	w := f.request("POST", "/admin/access", url.Values{"access_csrf": {fixtureCSRF(a)}, "action": {"password"}, "current_password": {fixtureAdminPassword}, "password": {newPassword}, "password_confirm": {newPassword}}, a)
	if w.Code != 303 {
		t.Fatalf("password change failed: %d", w.Code)
	}
	if w = f.request("GET", "/admin/access", nil, b); w.Code != 303 {
		t.Fatal("old session survived password change")
	}
	c := f.login(t, newPassword)
	_ = f.store.Update(context.Background(), func(s *adminAuthState) error {
		s.Sessions[accessDigest(c.Value)] = adminSession{Expires: time.Now().Add(-time.Minute)}
		return nil
	})
	if w = f.request("GET", "/admin/access", nil, c); w.Code != 303 {
		t.Fatal("expired session accepted")
	}
}
func TestAdminRecoveryQuotaPersistsBetweenInstancesAndExpiredTokenFails(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	for i := 0; i < 4; i++ {
		_ = f.store.Update(context.Background(), func(s *adminAuthState) error { delete(s.Limits, accessDigest("recovery-minute")); return nil })
		renderer, _ := newTemplateRenderer()
		f.handler = newServer(f.cfg, renderer).Handler
		w := f.request("POST", "/admin/recovery", url.Values{}, nil)
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	if len(f.mails) != 3 {
		t.Fatalf("expected three recovery mails, got %d", len(f.mails))
	}
	token := f.lastMailToken(t)
	_ = f.store.Update(context.Background(), func(s *adminAuthState) error { s.ResetExpires = time.Now().Add(-time.Second); return nil })
	if w := f.request("GET", "/admin/reset-password?token="+token, nil, nil); w.Code != 400 {
		t.Fatal("expired link accepted")
	}
}
func TestAdminResetTokenConsumedAtomically(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	f.request("POST", "/admin/recovery", url.Values{}, nil)
	token := f.lastMailToken(t)
	values := url.Values{"token": {token}, "password": {"atomic-new-password-123"}, "password_confirm": {"atomic-new-password-123"}}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if f.request("POST", "/admin/reset-password", values, nil).Code == 303 {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("one-time token accepted %d times", successes.Load())
	}
}
func TestAdminRecoveryEmailRequiresReauthenticationAndNewMailboxProof(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	cookie := f.login(t, fixtureAdminPassword)
	values := url.Values{"action": {"email"}, "access_csrf": {fixtureCSRF(cookie)}, "current_password": {"wrong"}, "recovery_email": {"new@example.test"}}
	if w := f.request("POST", "/admin/access", values, cookie); w.Code != 400 {
		t.Fatal("changed email without current password")
	}
	values.Set("current_password", fixtureAdminPassword)
	if w := f.request("POST", "/admin/access", values, cookie); w.Code != 200 {
		t.Fatalf("email request failed: %d", w.Code)
	}
	token := f.lastMailToken(t)
	if w := f.request("GET", "/admin/confirm-recovery?token="+token, nil, nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	state, _ := f.store.Load(context.Background())
	if state.RecoveryEmail != "owner@example.test" {
		t.Fatal("email scanner GET changed recovery email")
	}
	if f.mails[0].To != "new@example.test" {
		t.Fatal("verification not sent to new mailbox")
	}
	if w := f.request("POST", "/admin/confirm-recovery", url.Values{"token": {token}}, nil); w.Code != 303 {
		t.Fatal(w.Code)
	}
	state, _ = f.store.Load(context.Background())
	if state.RecoveryEmail != "new@example.test" || state.PendingEmail != "" {
		t.Fatal("new address not activated")
	}
	if w := f.request("GET", "/admin/access", nil, cookie); w.Code != 303 {
		t.Fatal("email change left old session active")
	}
	if w := f.request("POST", "/admin/confirm-recovery", url.Values{"token": {token}}, nil); w.Code != 400 {
		t.Fatal("verification token reused")
	}
}
func TestAdminCSRFOriginAndLoginThrottle(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	cookie := f.login(t, fixtureAdminPassword)
	if w := f.request("POST", "/admin/save", url.Values{}, cookie); w.Code != 403 {
		t.Fatal("authenticated mutation without CSRF accepted")
	}
	for _, origin := range []string{"", "https://attacker.example"} {
		r := httptest.NewRequest("POST", f.cfg.SiteURL+"/admin/login", strings.NewReader("password="+fixtureAdminPassword))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("untrusted origin accepted")
		}
	}
	for i := 0; i < 12; i++ {
		w := f.request("POST", "/admin/login", url.Values{"password": {"wrong"}}, nil)
		if i == 11 && w.Code != 429 {
			t.Fatal("login rate limit missing")
		}
	}
}
func TestAdminBootstrapAllowsRecoveryButStillBlocksPublicSubmissions(t *testing.T) {
	t.Setenv("LIIKKUVA_BOOTSTRAP", "1")
	f := newAccessFixture(t, fixtureAdminPassword)
	if w := f.request("POST", "/admin/recovery", url.Values{}, nil); w.Code != 200 {
		t.Fatal("bootstrap blocks recovery")
	}
	if w := f.request("POST", "/lead", url.Values{}, nil); w.Code != 503 {
		t.Fatal("bootstrap must keep lead intake paused")
	}
}
func TestAdminNoSMTPAndSafeRecoveryURL(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	f.cfg.SMTPHost = ""
	renderer, _ := newTemplateRenderer()
	f.handler = newServer(f.cfg, renderer).Handler
	w := f.request("GET", "/admin/recovery", nil, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "--password-only") {
		t.Fatal("offline recovery instruction missing")
	}
	if _, err := accessURL(Config{SiteURL: "http://insecure.test"}, "/admin/reset-password", "token"); err == nil {
		t.Fatal("HTTP recovery URL accepted")
	}
	if w := f.request("POST", "/admin/recovery", url.Values{}, nil); w.Code != 503 || len(f.mails) != 0 {
		t.Fatal("pretended to send email without SMTP")
	}
}

func TestAdminLogoutCannotBeTriggeredByGET(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	cookie := f.login(t, fixtureAdminPassword)
	if w := f.request("GET", "/admin/logout", nil, cookie); w.Code != 405 {
		t.Fatal("GET logged out an active session")
	}
	if w := f.request("GET", "/admin/access", nil, cookie); w.Code != 200 {
		t.Fatal("session invalidated by GET")
	}
	if w := f.request("POST", "/admin/logout", url.Values{"access_csrf": {fixtureCSRF(cookie)}}, cookie); w.Code != 303 {
		t.Fatal("logout POST failed")
	}
	if w := f.request("GET", "/admin/access", nil, cookie); w.Code != 303 {
		t.Fatal("logged-out session still accepted")
	}
	if err := validateAdminPassword("password\nwith\nnewlines"); err == nil {
		t.Fatal("multiline password accepted")
	}
}
func TestAdminCLIPasswordInvalidatesSessionsWithoutExposingPassword(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	cookie := f.login(t, fixtureAdminPassword)
	if err := resetAdminPasswordCLI(f.cfg, strings.NewReader("console-password-12345\n")); err != nil {
		t.Fatal(err)
	}
	if w := f.request("GET", "/admin/access", nil, cookie); w.Code != 303 {
		t.Fatal("CLI reset retained session")
	}
	f.login(t, "console-password-12345")
}

type memorySettingsStore struct {
	mu                       sync.Mutex
	state                    SiteSettings
	loadError                error
	retryWithConcurrentWrite bool
}

func (s *memorySettingsStore) Load(context.Context) (SiteSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state, s.loadError
}
func (s *memorySettingsStore) Update(_ context.Context, fn func(SiteSettings) (SiteSettings, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.retryWithConcurrentWrite {
		if _, err := fn(s.state); err != nil {
			return err
		}
		s.state.Revision++
		s.state.Phone = "concurrent-change"
	}
	next, err := fn(s.state)
	if err != nil {
		return err
	}
	s.state = next
	return nil
}
func TestSettingsTwoWindowsCannotOverwriteEachOther(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	store := &memorySettingsStore{}
	f.cfg.SettingsStore = store
	renderer, _ := newTemplateRenderer()
	f.handler = newServer(f.cfg, renderer).Handler
	cookie := f.login(t, fixtureAdminPassword)
	values := url.Values{"access_csrf": {fixtureCSRF(cookie)}, "active_tab": {"general"}, "settings_revision": {"0"}, "phone": {"first-window"}}
	if w := f.request("POST", "/admin/save", values, cookie); w.Code != 303 {
		t.Fatalf("first save: %d", w.Code)
	}
	values.Set("phone", "stale-second-window")
	w := f.request("POST", "/admin/save", values, cookie)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "уже изменены") {
		t.Fatal("stale form overwrote current settings")
	}
	current, _ := store.Load(context.Background())
	if current.Phone != "first-window" || current.Revision != 1 {
		t.Fatal("committed settings were replaced")
	}
}
func TestSettingsTransactionRetryStillRejectsStaleForm(t *testing.T) {
	store := &memorySettingsStore{retryWithConcurrentWrite: true}
	err := saveSiteSettings(context.Background(), Config{SettingsStore: store}, SiteSettings{Phone: "stale", Revision: 0})
	if !errors.Is(err, errSiteSettingsConflict) {
		t.Fatal("transaction retry accepted a stale revision")
	}
	if store.state.Phone != "concurrent-change" || store.state.Revision != 1 {
		t.Fatal("concurrent write was overwritten")
	}
}
func TestUnavailableSettingsNeverRenderBlankEditor(t *testing.T) {
	f := newAccessFixture(t, fixtureAdminPassword)
	f.cfg.SettingsStore = &memorySettingsStore{loadError: errors.New("offline")}
	renderer, _ := newTemplateRenderer()
	f.handler = newServer(f.cfg, renderer).Handler
	cookie := f.login(t, fixtureAdminPassword)
	w := f.request("GET", "/admin?tab=general", nil, cookie)
	if w.Code != 503 || strings.Contains(w.Body.String(), "adminSettingsForm") {
		t.Fatal("failed read exposed a blank settings form")
	}
}
func TestEveryAdminPostFormHasCSRFField(t *testing.T) {
	renderer, err := newTemplateRenderer()
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	err = renderer.renderAdmin(w, AdminPageData{
		Settings: normalizeAdminSettingsForTemplate(SiteSettings{}), ActiveTab: "general", CSRF: "test-proof", MailEnabled: true,
		SelectedLead: &AdminLeadDetails{ID: "synthetic-lead", Email: "client@example.test", ClientStatus: "new"},
		Leads:        []AdminLeadListItem{{ID: "synthetic-lead", ClientStatus: "new", Email: "client@example.test"}},
		Reviews:      []AdminReviewListItem{{ID: "synthetic-review", Status: "pending"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	forms := regexp.MustCompile(`(?s)<form\b[^>]*method="post"[^>]*>(.*?)</form>`).FindAllStringSubmatch(w.Body.String(), -1)
	if len(forms) < 3 {
		t.Fatal("admin forms missing")
	}
	for _, form := range forms {
		if !strings.Contains(form[1], `name="access_csrf" value="test-proof"`) {
			t.Fatal("POST form has no CSRF proof")
		}
	}
}

// Opt-in local preview fixtures contain only synthetic data. They let the
// responsive browser checks use the same templates as production.
func TestWriteAdminPreview(t *testing.T) {
	dir := os.Getenv("LIIKKUVA_QA_DIR")
	if dir == "" {
		t.Skip("local preview not requested")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	renderer, err := newTemplateRenderer()
	if err != nil {
		t.Fatal(err)
	}
	settings := normalizeAdminSettingsForTemplate(SiteSettings{BusinessName: "Liikkuva", Phone: "+358 40 123 4567", ServiceEmail: "service@example.test"})
	for _, tab := range []string{"overview", "general", "assets", "analytics", "emails", "leads", "help", "seo"} {
		file, err := os.Create(filepath.Join(dir, "admin-"+tab+".html"))
		if err != nil {
			t.Fatal(err)
		}
		err = renderer.admin.Execute(file, AdminPageData{PageTitle: "Кабинет Liikkuva", Settings: settings, AnalyticsEvents: analyticsEventCatalog(settings), AssetsConfigured: true, ActiveTab: tab, CSRF: "synthetic-preview-token", ReleaseVersion: releaseVersion, MailEnabled: true, OverviewTotalLeads: 12, OverviewFailedLeads: 1, OverviewPendingReviews: 2})
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []string{"login", "recovery", "reset", "access"} {
		file, err := os.Create(filepath.Join(dir, "access-"+mode+".html"))
		if err != nil {
			t.Fatal(err)
		}
		err = adminAccessTemplate.Execute(file, accessPageData{Mode: mode, MailEnabled: true, RecoveryEmail: "owner@example.test", CSRF: "preview", Token: "preview"})
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}
