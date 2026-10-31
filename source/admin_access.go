package main

// Administrative credentials live in a private Firestore document, separately
// from public site settings. Raw passwords, session IDs and email tokens are
// never persisted. A transaction consumes each recovery token exactly once.
import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"cloud.google.com/go/firestore"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const adminSessionLifetime = 12 * time.Hour
const adminRecoveryLifetime = 30 * time.Minute

// no-referrer makes browser form navigations send Origin: null, which our
// request-origin check correctly rejects. strict-origin preserves that proof
// while keeping recovery tokens and other URL paths out of the Referer header.
const adminReferrerPolicy = "strict-origin"

type adminSession struct {
	Expires time.Time `firestore:"expires"`
}
type adminRateLimit struct {
	Count int       `firestore:"count"`
	Until time.Time `firestore:"until"`
}
type adminAuditEvent struct {
	At     time.Time `firestore:"at"`
	Action string    `firestore:"action"`
}
type adminAuthState struct {
	PasswordHash  string                    `firestore:"password_hash"`
	RecoveryEmail string                    `firestore:"recovery_email"`
	ResetHash     string                    `firestore:"reset_hash"`
	ResetExpires  time.Time                 `firestore:"reset_expires"`
	VerifyHash    string                    `firestore:"verify_hash"`
	VerifyExpires time.Time                 `firestore:"verify_expires"`
	PendingEmail  string                    `firestore:"pending_email"`
	Sessions      map[string]adminSession   `firestore:"sessions"`
	Limits        map[string]adminRateLimit `firestore:"limits"`
	Audit         []adminAuditEvent         `firestore:"audit"`
}

type adminAuthStore interface {
	Load(context.Context) (adminAuthState, error)
	Update(context.Context, func(*adminAuthState) error) error
}
type firestoreAdminAuthStore struct {
	cfg           Config
	operatorToken string
}

func (s firestoreAdminAuthStore) client(ctx context.Context) (*firestore.Client, error) {
	if s.operatorToken != "" {
		return firestore.NewClientWithDatabase(ctx, s.cfg.FirestoreProjectID, "liikkuvakorjaamo", option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: s.operatorToken})))
	}
	return newFirestoreClient(ctx, s.cfg)
}

func (s firestoreAdminAuthStore) Load(ctx context.Context) (adminAuthState, error) {
	client, err := s.client(ctx)
	if err != nil {
		return adminAuthState{}, err
	}
	defer client.Close()
	doc, err := client.Collection(s.cfg.SiteSettingsCollection + "_private").Doc("admin_access").Get(ctx)
	if status.Code(err) == codes.NotFound {
		return adminAuthState{}, nil
	}
	if err != nil {
		return adminAuthState{}, err
	}
	var state adminAuthState
	err = doc.DataTo(&state)
	return state, err
}
func (s firestoreAdminAuthStore) Update(ctx context.Context, fn func(*adminAuthState) error) error {
	client, err := s.client(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	ref := client.Collection(s.cfg.SiteSettingsCollection + "_private").Doc("admin_access")
	return client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(ref)
		var state adminAuthState
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if err == nil {
			if err = doc.DataTo(&state); err != nil {
				return err
			}
		}
		if err := fn(&state); err != nil {
			return err
		}
		return tx.Set(ref, state)
	})
}
func accessStore(cfg Config) adminAuthStore {
	if cfg.AdminStore != nil {
		return cfg.AdminStore
	}
	return firestoreAdminAuthStore{cfg: cfg}
}
func prepareAccessState(state *adminAuthState, cfg Config) error {
	if state.Sessions == nil {
		state.Sessions = map[string]adminSession{}
	}
	if state.Limits == nil {
		state.Limits = map[string]adminRateLimit{}
	}
	if state.RecoveryEmail == "" {
		email, err := normalizeRecoveryEmail(cfg.AdminRecoveryEmail)
		if err != nil {
			return errors.New("Не настроен адрес восстановления владельца.")
		}
		state.RecoveryEmail = email
		// Legacy ADMIN_PASSWORD is imported once. Later deployments cannot undo a
		// password changed by the owner or through a recovery link.
		if cfg.AdminPassword != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), 12)
			if err != nil {
				return err
			}
			state.PasswordHash = string(hash)
		}
	}
	now := time.Now()
	for key, value := range state.Sessions {
		if !value.Expires.After(now) {
			delete(state.Sessions, key)
		}
	}
	for key, value := range state.Limits {
		if !value.Until.After(now) {
			delete(state.Limits, key)
		}
	}
	return nil
}
func updateAccess(ctx context.Context, cfg Config, fn func(*adminAuthState) error) error {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	return accessStore(cfg).Update(ctx, func(state *adminAuthState) error {
		if err := prepareAccessState(state, cfg); err != nil {
			return err
		}
		return fn(state)
	})
}
func loadAccess(ctx context.Context, cfg Config) (adminAuthState, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	state, err := accessStore(cfg).Load(ctx)
	if err != nil {
		return state, err
	}
	if state.RecoveryEmail == "" {
		err = updateAccess(ctx, cfg, func(s *adminAuthState) error { state = *s; return nil })
	}
	return state, err
}
func normalizeRecoveryEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || len(value) > 254 || strings.ContainsAny(value, "\r\n") {
		return "", errors.New("Укажите корректный email без имени и дополнительных адресов.")
	}
	return value, nil
}
func validateAdminPassword(value string) error {
	if !utf8.ValidString(value) || strings.ContainsAny(value, "\r\n") || utf8.RuneCountInString(value) < 12 || len(value) > 72 {
		return errors.New("Пароль должен содержать не менее 12 символов и не более 72 байт. Можно использовать длинную фразу.")
	}
	return nil
}
func accessToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
func accessDigest(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:])
}
func addAccessAudit(state *adminAuthState, action string) {
	state.Audit = append([]adminAuditEvent{{At: time.Now().UTC(), Action: action}}, state.Audit...)
	if len(state.Audit) > 30 {
		state.Audit = state.Audit[:30]
	}
}
func takeAccessLimit(state *adminAuthState, key string, max int, window time.Duration) bool {
	key = accessDigest(key)
	limit := state.Limits[key]
	if !limit.Until.After(time.Now()) {
		limit = adminRateLimit{Until: time.Now().Add(window)}
	}
	if limit.Count >= max {
		return false
	}
	// Bound the document even if hostile clients rotate spoofed IP addresses.
	if _, ok := state.Limits[key]; !ok && len(state.Limits) >= 128 {
		return false
	}
	limit.Count++
	state.Limits[key] = limit
	return true
}
func accessRequestIP(r *http.Request) string {
	// The per-account/global quota remains effective even behind proxies.
	ip := getClientIP(r)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}
func sessionToken(r *http.Request) string {
	cookie, err := r.Cookie(adminCookieName)
	if err != nil || len(cookie.Value) != 43 {
		return ""
	}
	return cookie.Value
}
func accessCSRF(r *http.Request) string {
	token := sessionToken(r)
	if token == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(token))
	mac.Write([]byte("liikkuva-admin-form-v1"))
	return hex.EncodeToString(mac.Sum(nil))
}

type adminAccessContextKey struct{}

func authenticateAdmin(cfg Config, r *http.Request) bool {
	if authenticated, ok := r.Context().Value(adminAccessContextKey{}).(bool); ok {
		return authenticated
	}
	token := sessionToken(r)
	if token == "" {
		return false
	}
	state, err := loadAccess(r.Context(), cfg)
	if err != nil {
		return false
	}
	session, ok := state.Sessions[accessDigest(token)]
	return ok && session.Expires.After(time.Now())
}
func adminPublicPath(path string) bool {
	switch path {
	case "/admin", "/admin/login", "/admin/recovery", "/admin/reset-password", "/admin/confirm-recovery":
		return true
	}
	return false
}
func adminAccessGuard(cfg Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/admin") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store, private")
		w.Header().Set("Referrer-Policy", adminReferrerPolicy)
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		w.Header().Set("X-Frame-Options", "DENY")
		authenticated := authenticateAdmin(cfg, r)
		r = r.WithContext(context.WithValue(r.Context(), adminAccessContextKey{}, authenticated))
		if !adminPublicPath(r.URL.Path) && !authenticated {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if r.Method != http.MethodPost {
				http.Error(w, "Метод не поддерживается.", 405)
				return
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				if ref, err := url.Parse(r.Referer()); err == nil && ref.Host != "" {
					origin = ref.Scheme + "://" + ref.Host
				}
			}
			u, err := url.Parse(origin)
			if origin == "" || err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || (u.Host != r.Host && strings.TrimRight(origin, "/") != strings.TrimRight(cfg.SiteURL, "/")) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				rejectAdminForm(w, r, "Не удалось подтвердить страницу отправки. Откройте форму заново и повторите действие.")
				return
			}
			limit := int64(2 << 20)
			if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
				limit = 25 << 20
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
				err = r.ParseMultipartForm(2 << 20)
			} else {
				err = r.ParseForm()
			}
			if r.MultipartForm != nil {
				defer r.MultipartForm.RemoveAll()
			}
			if err != nil {
				http.Error(w, "Не удалось прочитать форму. Проверьте размер вложений.", 400)
				return
			}
			if !adminPublicPath(r.URL.Path) && (accessCSRF(r) == "" || subtle.ConstantTimeCompare([]byte(r.PostFormValue("access_csrf")), []byte(accessCSRF(r))) != 1) {
				rejectAdminForm(w, r, "Срок действия формы истёк. Обновите страницу и повторите действие.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func rejectAdminForm(w http.ResponseWriter, r *http.Request, message string) {
	mode := "request-error"
	if r.URL.Path == "/admin" || r.URL.Path == "/admin/login" {
		mode = "login"
	}
	renderAccess(w, http.StatusForbidden, accessPageData{Mode: mode, Error: message})
}

//go:embed admin-access.html
var adminAccessHTML string
var adminAccessTemplate = template.Must(template.New("access").Funcs(template.FuncMap{"localTime": adminAuditTime}).Parse(adminAccessHTML))

type accessPageData struct {
	Mode, Error, Message, Token, CSRF, RecoveryEmail, PendingEmail string
	ReleaseVersion                                                 string
	MailEnabled, HasPassword                                       bool
	Audit                                                          []adminAuditEvent
}

func renderAccess(w http.ResponseWriter, code int, data accessPageData) {
	data.ReleaseVersion = releaseVersion
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Referrer-Policy", adminReferrerPolicy)
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; img-src 'self'; script-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	w.WriteHeader(code)
	if err := adminAccessTemplate.Execute(w, data); err != nil {
		log.Printf("admin access render failed: %v", err)
	}
}
func (h *AdminHandler) renderAdmin(w http.ResponseWriter, r *http.Request, data AdminPageData) error {
	data.CSRF = accessCSRF(r)
	data.MailEnabled = mailNotificationsEnabled(h.cfg)
	data.TelegramConfigured = strings.TrimSpace(h.cfg.TelegramBotToken) != "" && strings.TrimSpace(h.cfg.TelegramChatID) != ""
	data.ReleaseVersion = releaseVersion
	data.IntakePaused = getEnv("LIIKKUVA_BOOTSTRAP", "") == "1"
	return h.renderer.renderAdmin(w, data)
}
func (h *AdminHandler) accessLogin(w http.ResponseWriter, r *http.Request) {
	data := accessPageData{Mode: "login", MailEnabled: mailNotificationsEnabled(h.cfg)}
	switch r.URL.Query().Get("notice") {
	case "password":
		data.Message = "Пароль изменён. Войдите с новым паролем."
	case "email":
		data.Message = "Новый адрес восстановления подтверждён. Войдите в кабинет."
	}
	if r.Method != http.MethodPost {
		renderAccess(w, 200, data)
		return
	}
	allowed := false
	err := updateAccess(r.Context(), h.cfg, func(state *adminAuthState) error {
		allowed = takeAccessLimit(state, "login-global", 60, 15*time.Minute) && takeAccessLimit(state, "login-ip:"+accessRequestIP(r), 10, 15*time.Minute)
		return nil
	})
	if err != nil {
		data.Error = "Хранилище доступа недоступно. Запустите команду обновления или обратитесь к владельцу проекта."
		renderAccess(w, 503, data)
		return
	}
	if !allowed {
		data.Error = "Слишком много попыток. Повторите вход через 15 минут."
		w.Header().Set("Retry-After", "900")
		renderAccess(w, 429, data)
		return
	}
	state, err := loadAccess(r.Context(), h.cfg)
	password := r.PostFormValue("password")
	if err != nil || len(password) > 72 || state.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(state.PasswordHash), []byte(password)) != nil {
		data.Error = "Пароль не подошёл. Проверьте ввод или восстановите доступ по почте."
		renderAccess(w, 401, data)
		return
	}
	token, err := accessToken()
	if err == nil {
		err = updateAccess(r.Context(), h.cfg, func(current *adminAuthState) error {
			if current.PasswordHash != state.PasswordHash {
				return errors.New("password changed")
			}
			if len(current.Sessions) >= 20 {
				keys := make([]string, 0, len(current.Sessions))
				for key := range current.Sessions {
					keys = append(keys, key)
				}
				sort.Slice(keys, func(i, j int) bool {
					return current.Sessions[keys[i]].Expires.Before(current.Sessions[keys[j]].Expires)
				})
				delete(current.Sessions, keys[0])
			}
			current.Sessions[accessDigest(token)] = adminSession{Expires: time.Now().Add(adminSessionLifetime)}
			addAccessAudit(current, "Вход в кабинет")
			return nil
		})
	}
	if err != nil {
		data.Error = "Не удалось открыть сеанс. Повторите вход."
		renderAccess(w, 503, data)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: adminCookieName, Value: token, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(h.cfg.SiteURL, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: int(adminSessionLifetime.Seconds())})
	http.Redirect(w, r, "/admin?tab=overview", 303)
}
func clearAdminCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: adminCookieName, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}
func (h *AdminHandler) accessLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Для выхода используйте кнопку в кабинете.", http.StatusMethodNotAllowed)
		return
	}
	_ = updateAccess(r.Context(), h.cfg, func(state *adminAuthState) error { delete(state.Sessions, accessDigest(sessionToken(r))); return nil })
	clearAdminCookie(w)
	http.Redirect(w, r, "/admin", 303)
}
func sendAccessMail(cfg Config, to, subject, message string) error {
	// Recovery links must never be copied to the client-mail BCC recipient.
	cfg.MailCopyTo = ""
	if cfg.AdminSend != nil {
		return cfg.AdminSend(cfg, to, subject, message, standaloneEmailHTML(message))
	}
	return sendSMTPMail(cfg, to, subject, message, standaloneEmailHTML(message))
}
func accessURL(cfg Config, path, token string) (string, error) {
	u, err := url.Parse(cfg.SiteURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return "", errors.New("Для восстановления нужен корректный HTTPS-адрес сайта.")
	}
	u.Path = path
	u.RawPath = ""
	u.RawQuery = url.Values{"token": {token}}.Encode()
	u.Fragment = ""
	return u.String(), nil
}
func (h *AdminHandler) Recovery(w http.ResponseWriter, r *http.Request) {
	data := accessPageData{Mode: "recovery", MailEnabled: mailNotificationsEnabled(h.cfg)}
	if r.Method != http.MethodPost {
		renderAccess(w, 200, data)
		return
	}
	if !data.MailEnabled {
		data.Error = "Отправка писем пока не подключена. Владелец может задать новый пароль командой python3 update.py --password-only в Cloud Shell."
		renderAccess(w, 503, data)
		return
	}
	token, err := accessToken()
	var link, email string
	if err == nil {
		link, err = accessURL(h.cfg, "/admin/reset-password", token)
	}
	allowed := false
	if err == nil {
		err = updateAccess(r.Context(), h.cfg, func(state *adminAuthState) error {
			allowed = takeAccessLimit(state, "recovery-minute", 1, time.Minute) && takeAccessLimit(state, "recovery-hour", 3, time.Hour)
			if !allowed {
				return nil
			}
			state.ResetHash = accessDigest(token)
			state.ResetExpires = time.Now().Add(adminRecoveryLifetime)
			email = state.RecoveryEmail
			return nil
		})
	}
	if err != nil {
		data.Error = "Не удалось подготовить восстановление. Повторите позже."
		renderAccess(w, 503, data)
		return
	}
	if allowed {
		err = sendAccessMail(h.cfg, email, "Liikkuva — восстановление доступа", "Вы запросили новый пароль для кабинета Liikkuva.\n\nОткройте ссылку в течение 30 минут:\n"+link+"\n\nСсылка действует один раз. Если вы не запрашивали восстановление, просто проигнорируйте письмо: текущий пароль продолжает работать.")
		if err != nil {
			_ = updateAccess(r.Context(), h.cfg, func(state *adminAuthState) error {
				if state.ResetHash == accessDigest(token) {
					state.ResetHash = ""
				}
				return nil
			})
			log.Print("admin recovery email delivery failed")
			data.Error = "Почтовый сервис не принял письмо. Проверьте настройки отправки или задайте пароль через Cloud Shell."
			renderAccess(w, 503, data)
			return
		}
	}
	data.Message = "Запрос принят. Проверьте почту восстановления и папку «Спам». Повторная отправка доступна через минуту, не более трёх раз в час."
	renderAccess(w, 200, data)
}
func validAccessProof(hash string, expires time.Time, token string) bool {
	return len(token) == 43 && hash != "" && expires.After(time.Now()) && subtle.ConstantTimeCompare([]byte(hash), []byte(accessDigest(token))) == 1
}
func changeStoredPassword(state *adminAuthState, hash string, action string) {
	state.PasswordHash = hash
	state.Sessions = map[string]adminSession{}
	state.ResetHash = ""
	state.ResetExpires = time.Time{}
	state.VerifyHash = ""
	state.VerifyExpires = time.Time{}
	state.PendingEmail = ""
	state.Limits = map[string]adminRateLimit{}
	addAccessAudit(state, action)
}
func (h *AdminHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if r.Method == http.MethodPost {
		token = r.PostFormValue("token")
	}
	data := accessPageData{Mode: "reset", Token: token}
	state, err := loadAccess(r.Context(), h.cfg)
	if err != nil || !validAccessProof(state.ResetHash, state.ResetExpires, token) {
		data.Mode = "invalid"
		data.Error = "Ссылка недействительна, уже использована или истекли 30 минут. Запросите новую ссылку."
		renderAccess(w, 400, data)
		return
	}
	if r.Method != http.MethodPost {
		renderAccess(w, 200, data)
		return
	}
	password := r.PostFormValue("password")
	if err := validateAdminPassword(password); err != nil {
		data.Error = err.Error()
		renderAccess(w, 400, data)
		return
	}
	if password != r.PostFormValue("password_confirm") {
		data.Error = "Пароли не совпадают."
		renderAccess(w, 400, data)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err == nil {
		err = updateAccess(r.Context(), h.cfg, func(current *adminAuthState) error {
			if !validAccessProof(current.ResetHash, current.ResetExpires, token) {
				return errors.New("invalid proof")
			}
			changeStoredPassword(current, string(hash), "Пароль восстановлен по почте")
			return nil
		})
	}
	if err != nil {
		data.Error = "Не удалось применить ссылку. Возможно, она уже использована. Запросите восстановление ещё раз."
		renderAccess(w, 400, data)
		return
	}
	_ = sendAccessMail(h.cfg, state.RecoveryEmail, "Liikkuva — пароль изменён", "Пароль кабинета изменён. Все предыдущие сеансы завершены. Если это были не вы, восстановите доступ на "+h.cfg.SiteURL+"/admin/recovery")
	clearAdminCookie(w)
	http.Redirect(w, r, "/admin?notice=password", 303)
}
func (h *AdminHandler) Access(w http.ResponseWriter, r *http.Request) {
	if !isAdminAuthenticated(h.cfg, r) {
		http.Redirect(w, r, "/admin", 303)
		return
	}
	state, err := loadAccess(r.Context(), h.cfg)
	data := accessPageData{Mode: "access", CSRF: accessCSRF(r), MailEnabled: mailNotificationsEnabled(h.cfg), RecoveryEmail: state.RecoveryEmail, PendingEmail: state.PendingEmail, HasPassword: state.PasswordHash != "", Audit: state.Audit}
	if err != nil {
		data.Error = "Не удалось загрузить настройки доступа."
		renderAccess(w, 503, data)
		return
	}
	if r.Method != http.MethodPost {
		renderAccess(w, 200, data)
		return
	}
	allowedChange := false
	if err := updateAccess(r.Context(), h.cfg, func(current *adminAuthState) error {
		allowedChange = takeAccessLimit(current, "access-reauth", 10, 15*time.Minute)
		return nil
	}); err != nil {
		data.Error = "Не удалось проверить действие. Повторите позже."
		renderAccess(w, 503, data)
		return
	}
	if !allowedChange {
		data.Error = "Слишком много попыток изменения доступа. Повторите через 15 минут."
		renderAccess(w, 429, data)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(state.PasswordHash), []byte(r.PostFormValue("current_password"))) != nil {
		data.Error = "Текущий пароль не подошёл."
		renderAccess(w, 400, data)
		return
	}
	switch r.PostFormValue("action") {
	case "password":
		password := r.PostFormValue("password")
		if err := validateAdminPassword(password); err != nil {
			data.Error = err.Error()
			renderAccess(w, 400, data)
			return
		}
		if password != r.PostFormValue("password_confirm") {
			data.Error = "Пароли не совпадают."
			renderAccess(w, 400, data)
			return
		}
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), 12)
		if hashErr != nil {
			data.Error = "Не удалось обработать пароль."
			renderAccess(w, 500, data)
			return
		}
		err = updateAccess(r.Context(), h.cfg, func(current *adminAuthState) error {
			if current.PasswordHash != state.PasswordHash {
				return errors.New("password changed")
			}
			changeStoredPassword(current, string(hash), "Пароль изменён в кабинете")
			return nil
		})
		if err == nil {
			if data.MailEnabled {
				_ = sendAccessMail(h.cfg, state.RecoveryEmail, "Liikkuva — пароль изменён", "В кабинете изменён пароль. Все предыдущие сеансы завершены. Если это были не вы, откройте "+h.cfg.SiteURL+"/admin/recovery")
			}
			clearAdminCookie(w)
			http.Redirect(w, r, "/admin?notice=password", 303)
			return
		}
	case "email":
		if !data.MailEnabled {
			data.Error = "Сначала подключите отправку писем. Новый адрес активируется только после подтверждения."
			renderAccess(w, 400, data)
			return
		}
		email, emailErr := normalizeRecoveryEmail(r.PostFormValue("recovery_email"))
		if emailErr != nil {
			data.Error = emailErr.Error()
			renderAccess(w, 400, data)
			return
		}
		if email == state.RecoveryEmail {
			data.Message = "Этот адрес уже используется для восстановления."
			renderAccess(w, 200, data)
			return
		}
		token, tokenErr := accessToken()
		if tokenErr != nil {
			data.Error = "Не удалось создать ссылку."
			renderAccess(w, 500, data)
			return
		}
		link, linkErr := accessURL(h.cfg, "/admin/confirm-recovery", token)
		if linkErr != nil {
			data.Error = linkErr.Error()
			renderAccess(w, 500, data)
			return
		}
		allowed := false
		err = updateAccess(r.Context(), h.cfg, func(current *adminAuthState) error {
			if current.PasswordHash != state.PasswordHash {
				return errors.New("password changed")
			}
			allowed = takeAccessLimit(current, "change-email", 5, time.Hour)
			if !allowed {
				return nil
			}
			current.PendingEmail = email
			current.VerifyHash = accessDigest(token)
			current.VerifyExpires = time.Now().Add(adminRecoveryLifetime)
			return nil
		})
		if err == nil && !allowed {
			data.Error = "Лимит подтверждений исчерпан. Повторите через час."
			renderAccess(w, 429, data)
			return
		}
		if err == nil {
			err = sendAccessMail(h.cfg, email, "Liikkuva — подтвердите адрес восстановления", "Чтобы использовать этот адрес для восстановления доступа к кабинету, откройте ссылку в течение 30 минут и нажмите «Подтвердить адрес»:\n\n"+link+"\n\nДо подтверждения продолжает действовать прежний адрес.")
		}
		if err == nil {
			data.PendingEmail = email
			data.Message = "Письмо отправлено на новый адрес. До подтверждения восстановление приходит на прежнюю почту."
			renderAccess(w, 200, data)
			return
		}
		_ = updateAccess(r.Context(), h.cfg, func(current *adminAuthState) error {
			if current.VerifyHash == accessDigest(token) {
				current.VerifyHash = ""
				current.PendingEmail = ""
			}
			return nil
		})
	case "sessions":
		err = updateAccess(r.Context(), h.cfg, func(current *adminAuthState) error {
			if current.PasswordHash != state.PasswordHash {
				return errors.New("password changed")
			}
			current.Sessions = map[string]adminSession{}
			addAccessAudit(current, "Все сеансы завершены владельцем")
			return nil
		})
		if err == nil {
			clearAdminCookie(w)
			http.Redirect(w, r, "/admin", 303)
			return
		}
	default:
		data.Error = "Неизвестное действие."
		renderAccess(w, 400, data)
		return
	}
	data.Error = "Изменение не выполнено. Проверьте подключение к хранилищу и почтовому сервису."
	renderAccess(w, 503, data)
}
func (h *AdminHandler) ConfirmRecovery(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if r.Method == http.MethodPost {
		token = r.PostFormValue("token")
	}
	state, err := loadAccess(r.Context(), h.cfg)
	data := accessPageData{Mode: "confirm", Token: token, PendingEmail: state.PendingEmail}
	if err != nil || !validAccessProof(state.VerifyHash, state.VerifyExpires, token) {
		data.Mode = "invalid"
		data.Error = "Ссылка подтверждения недействительна или истекла."
		renderAccess(w, 400, data)
		return
	}
	// GET is intentionally read-only: mail scanners cannot activate a new address.
	if r.Method != http.MethodPost {
		renderAccess(w, 200, data)
		return
	}
	err = updateAccess(r.Context(), h.cfg, func(current *adminAuthState) error {
		if !validAccessProof(current.VerifyHash, current.VerifyExpires, token) {
			return errors.New("invalid proof")
		}
		current.RecoveryEmail = current.PendingEmail
		current.PendingEmail = ""
		current.VerifyHash = ""
		current.ResetHash = ""
		current.Sessions = map[string]adminSession{}
		addAccessAudit(current, "Подтверждён новый адрес восстановления")
		return nil
	})
	if err != nil {
		data.Error = "Не удалось подтвердить адрес. Запросите новую ссылку в кабинете."
		renderAccess(w, 400, data)
		return
	}
	_ = sendAccessMail(h.cfg, state.RecoveryEmail, "Liikkuva — изменён адрес восстановления", "В кабинете подтверждён новый адрес восстановления: "+state.PendingEmail+". Если это были не вы, восстановите доступ через владельца Google Cloud проекта.")
	clearAdminCookie(w)
	http.Redirect(w, r, "/admin?notice=email", 303)
}

// Called only by an authenticated Cloud Shell operator, never an HTTP route.
func resetAdminPasswordCLI(cfg Config, input io.Reader) error {
	value, err := io.ReadAll(io.LimitReader(input, 1024))
	if err != nil {
		return err
	}
	password := strings.TrimSuffix(strings.TrimSuffix(string(value), "\n"), "\r")
	if err := validateAdminPassword(password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	err = updateAccess(context.Background(), cfg, func(state *adminAuthState) error {
		changeStoredPassword(state, string(hash), "Пароль задан владельцем через Cloud Shell")
		return nil
	})
	if err != nil {
		return fmt.Errorf("Не удалось сохранить пароль в Firestore: %w", err)
	}
	return nil
}

func adminAuditTime(value time.Time) string {
	location, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		location = time.UTC
	}
	return value.In(location).Format("02.01.2006 15:04")
}
