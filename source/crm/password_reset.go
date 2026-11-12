package crm

import (
    "context"
    "errors"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "strings"
    "time"

    "cloud.google.com/go/firestore"
    "golang.org/x/crypto/bcrypt"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// Password recovery is only available to ACTIVE, REGISTERED CRM users.
// No account is created by a reset request and password hashes are never emailed.
// A reset token is 256 bits from crypto/rand, stored only as SHA-256 in Firestore.
type passwordResetRecord struct {
    User    string
    Auth    string
    Expires time.Time
}

type passwordResetLimit struct {
    Count int
    Until time.Time
}

const passwordResetLifetime = 30 * time.Minute
const passwordResetLimitWindow = 15 * time.Minute
const passwordResetLimitCount = 3

const passwordResetGeneric = "Если этот email зарегистрирован и активен в CRM, письмо со ссылкой для восстановления будет отправлено."

func validCRMPassword(password string) bool {
    // Bcrypt supports passwords with 1–72 bytes. Exclude accidental whitespace-only inputs.
    return strings.TrimSpace(password) != "" && len([]byte(password)) <= 72
}

func normalizedCRMEmail(email string) string {
    email = strings.ToLower(strings.TrimSpace(email))
    if len(email) > 254 || !validAddress(email) {
        return ""
    }
    return email
}

func passwordResetLink(baseURL, token string) (string, error) {
    u, err := url.Parse(baseURL)
    if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
        u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
        return "", errors.New("CRM_PUBLIC_URL должен быть HTTPS-адресом без пути")
    }
    if len(token) != 64 {
        return "", errors.New("invalid token")
    }
    return strings.TrimRight(baseURL, "/") + "/crm/#reset=" + token, nil
}

// Owner/mechanic reset requests always return the same answer, including for
// unknown, inactive and recently rate-limited accounts.
func (s *Server) passwordForgot(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email string `json:"email"`
    }
    if err := decode(w, r, &req); err != nil {
        fail(w, 400, err.Error())
        return
    }
    generic := map[string]string{"message": passwordResetGeneric}
    email := normalizedCRMEmail(req.Email)
    if email == "" {
        write(w, 200, generic)
        return
    }
    id := digest(email)
    now := time.Now().UTC()
    allowed := false
    err := s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
        allowed = false
        ref := s.Collection("password_reset_limits").Doc(id)
        d, err := tx.Get(ref)
        var limit passwordResetLimit
        if status.Code(err) != codes.NotFound && err != nil {
            return err
        }
        if err == nil {
            if err = d.DataTo(&limit); err != nil {
                return err
            }
        }
        if !limit.Until.After(now) {
            limit.Count = 0
            limit.Until = now.Add(passwordResetLimitWindow)
        }
        if limit.Count >= passwordResetLimitCount {
            return nil
        }
        limit.Count++
        allowed = true
        return tx.Set(ref, limit)
    })
    if err != nil {
        // Avoid disclosing existence of specific CRM accounts on temporary DB errors.
        log.Printf("CRM password reset rate-limit storage unavailable: %v", err)
        fail(w, 503, "Восстановление временно недоступно. Повторите позже")
        return
    }
    if !allowed {
        write(w, 200, generic)
        return
    }

    d, err := s.Collection("users").Doc(id).Get(r.Context())
    if err != nil {
        if status.Code(err) != codes.NotFound {
            log.Printf("CRM password reset account lookup unavailable: %v", err)
        }
        write(w, 200, generic)
        return
    }
    var user User
    if d.DataTo(&user) != nil || !user.Active ||
        (user.Role != "owner" && user.Role != "mechanic") ||
        normalizedCRMEmail(user.Email) != email ||
        user.Hash == "" {
        write(w, 200, generic)
        return
    }
    token := randomID()
    link, err := passwordResetLink(s.URL, token)
    if err != nil {
        log.Print("CRM password reset URL misconfigured")
        fail(w, 503, "Восстановление временно недоступно. Повторите позже")
        return
    }

    record := passwordResetRecord{User: id, Auth: digest(user.Hash), Expires: now.Add(passwordResetLifetime)}
    ref := s.Collection("password_resets").Doc(digest(token))
    if _, err = ref.Create(r.Context(), record); err != nil {
        log.Printf("CRM password reset token storage unavailable: %v", err)
        fail(w, 503, "Восстановление временно недоступно. Повторите позже")
        return
    }

    from, _, _, _, err := mailConfig()
    if err != nil {
        log.Print("CRM password reset email unavailable: SMTP configuration missing")
        _, _ = ref.Delete(r.Context())
        write(w, 200, generic)
        return
    }
    subject := "Liikkuva CRM — восстановление пароля"
    body := fmt.Sprintf("Здравствуйте!\n\nДля установки нового пароля в Liikkuva CRM откройте ссылку:\n%s\n\nСсылка действует 30 минут и может быть использована только один раз.\n\nЕсли вы не запрашивали восстановление, просто проигнорируйте письмо.\n\nLiikkuva auto korjaamo", link)
    mailMessage := Email{
        ID: digest(token), MessageID: "<" + randomID() + "@liikkuvakorjaamo.fi>",
        From: from, To: user.Email, Language: "ru", Subject: subject,
        Body: body, ResetURL: link, Direction: "outgoing",
        Created: now,
    }
    send := s.resetMailer
    if send == nil {
        send = smtpSend
    }
    if _, err = send(mailMessage); err != nil {
        // Never reveal existence of the account or the reset token to the requester.
        log.Printf("CRM password reset mail was not accepted: %v", err)
        _, _ = ref.Delete(r.Context())
    }
    write(w, 200, generic)
}

func (s *Server) validPasswordReset(ctx context.Context, token string) (passwordResetRecord, User, error) {
    if len(token) != 64 {
        return passwordResetRecord{}, User{}, ErrForbidden
    }
    for _, ch := range token {
        if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
            return passwordResetRecord{}, User{}, ErrForbidden
        }
    }
    d, err := s.Collection("password_resets").Doc(digest(token)).Get(ctx)
    if err != nil {
        return passwordResetRecord{}, User{}, err
    }
    var record passwordResetRecord
    if err = d.DataTo(&record); err != nil || !record.Expires.After(time.Now()) || record.User == "" {
        return passwordResetRecord{}, User{}, ErrForbidden
    }
    ud, err := s.Collection("users").Doc(record.User).Get(ctx)
    if err != nil {
        return passwordResetRecord{}, User{}, err
    }
    var user User
    if ud.DataTo(&user) != nil || !user.Active || (user.Role != "owner" && user.Role != "mechanic") ||
        digest(user.Hash) != record.Auth || normalizedCRMEmail(user.Email) == "" ||
        digest(normalizedCRMEmail(user.Email)) != record.User {
        return passwordResetRecord{}, User{}, ErrForbidden
    }
    return record, user, nil
}

func (s *Server) passwordVerify(w http.ResponseWriter, r *http.Request) {
    var req struct { Token string `json:"token"` }
    if err := decode(w, r, &req); err != nil {
        fail(w, 400, err.Error())
        return
    }
    _, user, err := s.validPasswordReset(r.Context(), req.Token)
    if err != nil {
        fail(w, 400, "Ссылка недействительна или срок её действия истёк. Запросите новую")
        return
    }
    // Only possession of a valid single-use token discloses the registered email.
    write(w, 200, map[string]string{"email": user.Email})
}

func (s *Server) passwordReset(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Token string `json:"token"`
        Password string `json:"password"`
    }
    if err := decode(w, r, &req); err != nil {
        fail(w, 400, err.Error())
        return
    }
    if !validCRMPassword(req.Password) {
        fail(w, 400, "Пароль не может быть пустым; максимум 72 байта. Рекомендуем генератор надёжного пароля")
        return
    }
    // Hash once, before the Firestore transaction to avoid repeated bcrypt work.
    hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
    if err != nil {
        fail(w, 400, "Не удалось обработать пароль")
        return
    }
    _, _, err = s.validPasswordReset(r.Context(), req.Token)
    if err != nil {
        fail(w, 400, "Ссылка недействительна или срок её действия истёк")
        return
    }
    tokenRef := s.Collection("password_resets").Doc(digest(req.Token))
    err = s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
        d, err := tx.Get(tokenRef)
        if err != nil {
            return err
        }
        var record passwordResetRecord
        if d.DataTo(&record) != nil || !record.Expires.After(time.Now()) {
            return ErrForbidden
        }
        userRef := s.Collection("users").Doc(record.User)
        ud, err := tx.Get(userRef)
        if err != nil {
            return err
        }
        var user User
        if ud.DataTo(&user) != nil || !user.Active || digest(user.Hash) != record.Auth {
            return ErrForbidden
        }
        // Session authentication uses the hash fingerprint. All old sessions
        // automatically stop being authorized as soon as this update commits.
        if err = tx.Update(userRef, []firestore.Update{{Path:"Hash", Value:string(hash)}}); err != nil {
            return err
        }
        if err = tx.Delete(s.Collection("login_limits").Doc(record.User)); err != nil {
            return err
        }
        return tx.Delete(tokenRef)
    })
    if err != nil {
        fail(w, 400, "Ссылка недействительна или уже использована. Запросите новую")
        return
    }
    // Never log or return the token, password or password hash.
    write(w, 200, map[string]bool{"ok": true})
}
