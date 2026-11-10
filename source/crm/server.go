package crm

import (
	"cloud.google.com/go/firestore"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"sort"
	"strings"
	"time"
)

//go:embed web/*
var web embed.FS

type Server struct {
	DB          *firestore.Client
	Bucket, URL string
}

func Open(ctx context.Context, project, database, bucket, url string) (*Server, error) {
	db, e := firestore.NewClientWithDatabase(ctx, project, database)
	if e != nil {
		return nil, e
	}
	s := &Server{DB: db, Bucket: bucket, URL: strings.TrimRight(url, "/")}
	return s, nil
}
func (s *Server) Collection(n string) *firestore.CollectionRef { return s.DB.Collection("crm_" + n) }
func randomID() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func digest(s string) string { v := sha256.Sum256([]byte(s)); return hex.EncodeToString(v[:]) }
func write(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, code int, msg string) {
	write(w, code, map[string]string{"error": msg})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return errors.New("Некорректные поля запроса")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("Ожидается один JSON объект")
	}
	return nil
}
func (s *Server) Bootstrap(ctx context.Context) error {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("CRM_OWNER_EMAIL")))
	pass := os.Getenv("CRM_OWNER_PASSWORD")
	if email == "" || pass == "" {
		return nil
	}
	if len(pass) < 12 || len(pass) > 72 {
		return errors.New("CRM_OWNER_PASSWORD: 12–72 байта")
	}
	ref := s.Collection("users").Doc(digest(email))
	_, e := ref.Get(ctx)
	if e == nil {
		return nil
	}
	if status.Code(e) != codes.NotFound {
		return e
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(pass), 12)
	if e != nil {
		return e
	}
	_, e = ref.Create(ctx, User{ID: ref.ID, Name: "Владелец", Email: email, Role: "owner", Hash: string(hash), Active: true, TelegramID: os.Getenv("CRM_OWNER_TELEGRAM_ID")})
	if status.Code(e) == codes.AlreadyExists {
		return nil
	}
	return e
}

type session struct {
	User    string
	Auth    string
	Expires time.Time
}

func (s *Server) auth(r *http.Request) (User, error) {
	c, e := r.Cookie("crm_session")
	if e != nil {
		return User{}, ErrForbidden
	}
	d, e := s.Collection("sessions").Doc(digest(c.Value)).Get(r.Context())
	if e != nil {
		return User{}, ErrForbidden
	}
	var x session
	if d.DataTo(&x) != nil || x.Expires.Before(time.Now()) {
		return User{}, ErrForbidden
	}
	d, e = s.Collection("users").Doc(x.User).Get(r.Context())
	if e != nil {
		return User{}, ErrForbidden
	}
	var u User
	if d.DataTo(&u) != nil || !u.Active || (u.Role != "owner" && u.Role != "mechanic") || x.Auth != digest(u.Hash) {
		return User{}, ErrForbidden
	}
	return u, nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' blob:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	path := strings.TrimPrefix(r.URL.Path, "/crm")
	if path == "/api/v1/leads" {
		token := os.Getenv("CRM_INTAKE_TOKEN")
		if r.Method != "POST" || len(token) < 32 || subtle.ConstantTimeCompare([]byte(token), []byte(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))) != 1 {
			fail(w, 403, "Нет доступа")
			return
		}
		s.jobs(w, r, User{ID: "intake-api", Role: "owner", Active: true})
		return
	}
	if path == "/webhooks/whatsapp" {
		s.whatsappWebhook(w, r)
		return
	}
	if path == "/mail/inbound" {
		s.inboundMail(w, r)
		return
	}
	if path == "/bot/telegram" {
		s.bot(w, r)
		return
	}
	if path == "/internal/tick" {
		s.tickHTTP(w, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if r.Header.Get("X-CRM-Request") != "1" {
			fail(w, 403, "Недопустимый запрос")
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != s.URL {
			fail(w, 403, "Недопустимый источник запроса")
			return
		}
	}
	if path == "/api/login" && r.Method == "POST" {
		s.login(w, r)
		return
	}
	if !strings.HasPrefix(path, "/api/") {
		if r.Method != "GET" {
			fail(w, 405, "Method not allowed")
			return
		}
		file := "index.html"
		if path == "/app.js" {
			file = "app.js"
		}
		if path == "/style.css" {
			file = "style.css"
		}
		if path != "" && path != "/" && path != "/app.js" && path != "/style.css" {
			http.NotFound(w, r)
			return
		}
		b, _ := web.ReadFile("web/" + file)
		w.Header().Set("Content-Type", map[string]string{"index.html": "text/html; charset=utf-8", "app.js": "text/javascript; charset=utf-8", "style.css": "text/css; charset=utf-8"}[file])
		_, _ = w.Write(b)
		return
	}
	u, e := s.auth(r)
	if e != nil {
		fail(w, 401, "Войдите в CRM")
		return
	}
	switch {
	case path == "/api/me" && r.Method == "GET":
		write(w, 200, u)
	case path == "/api/logout" && r.Method == "POST":
		c, _ := r.Cookie("crm_session")
		_, e := s.Collection("sessions").Doc(digest(c.Value)).Delete(r.Context())
		if e != nil {
			fail(w, 503, "Не удалось завершить сеанс")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "crm_session", Path: "/crm", MaxAge: -1, Secure: strings.HasPrefix(s.URL, "https://"), HttpOnly: true, SameSite: http.SameSiteStrictMode})
		write(w, 200, map[string]bool{"ok": true})
	case path == "/api/users":
		s.users(w, r, u)
	case path == "/api/jobs":
		s.jobs(w, r, u)
	case strings.HasPrefix(path, "/api/jobs/"):
		s.job(w, r, u, strings.TrimPrefix(path, "/api/jobs/"))
	case path == "/api/calendar":
		s.calendarList(w, r, u)
	case path == "/api/finance":
		s.finance(w, r, u)
	case path == "/api/tariffs":
		s.tariffs(w, r, u)
	case path == "/api/whatsapp-inbox":
		s.whatsappInbox(w, r, u)
	case path == "/api/notifications" && r.Method == "GET" && u.Manager():
		s.notifications(w, r)
	case path == "/api/retry-notifications" && r.Method == "POST" && u.Manager():
		s.dispatch(w, r)
	default:
		http.NotFound(w, r)
	}
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if e := decode(w, r, &in); e != nil {
		fail(w, 400, e.Error())
		return
	}
	id := digest(strings.ToLower(strings.TrimSpace(in.Email)))
	// Persistent account rate limit, including nonexistent users. No password is logged.
	ref := s.Collection("login_limits").Doc(id)
	now := time.Now()
	e := s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
		var rate struct {
			Count int
			Until time.Time
		}
		d, e := tx.Get(ref)
		if e != nil && status.Code(e) != codes.NotFound {
			return e
		}
		if e == nil {
			if e = d.DataTo(&rate); e != nil {
				return e
			}
		}
		if rate.Until.Before(now) {
			rate.Count = 0
			rate.Until = now.Add(15 * time.Minute)
		}
		if rate.Count >= 10 {
			return errors.New("Слишком много попыток. Повторите через 15 минут")
		}
		rate.Count++
		return tx.Set(ref, rate)
	})
	if e != nil {
		fail(w, 429, "Слишком много попыток или вход временно недоступен. Повторите через 15 минут")
		return
	}
	d, e := s.Collection("users").Doc(id).Get(r.Context())
	var u User
	if e == nil {
		e = d.DataTo(&u)
	}
	if e != nil || !u.Active || bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(in.Password)) != nil {
		fail(w, 401, "Неверный email или пароль")
		return
	}
	token := randomID()
	_, e = s.Collection("sessions").Doc(digest(token)).Create(r.Context(), session{User: u.ID, Auth: digest(u.Hash), Expires: now.Add(12 * time.Hour)})
	if e != nil {
		fail(w, 503, "Ошибка создания сеанса")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "crm_session", Value: token, Path: "/crm", MaxAge: 43200, Secure: strings.HasPrefix(s.URL, "https://"), HttpOnly: true, SameSite: http.SameSiteStrictMode})
	write(w, 200, u)
}
func (s *Server) users(w http.ResponseWriter, r *http.Request, u User) {
	if !u.Manager() {
		fail(w, 403, "Нет доступа")
		return
	}
	if r.Method == "GET" {
		docs, e := s.Collection("users").Limit(100).Documents(r.Context()).GetAll()
		if e != nil {
			fail(w, 503, "Ошибка чтения пользователей")
			return
		}
		out := []User{}
		for _, d := range docs {
			var x User
			if d.DataTo(&x) != nil {
				continue
			}
			out = append(out, x)
		}
		write(w, 200, out)
		return
	}
	if r.Method != "POST" || u.Role != "owner" {
		fail(w, 403, "Только владелец управляет доступом")
		return
	}
	var x struct {
		MainCity   string          `json:"main_city"`
		Phone      string          `json:"phone"`
		Exceptions []WorkException `json:"exceptions"`
		Email      string          `json:"email"`
		Name       string          `json:"name"`
		Password   string          `json:"password"`
		Role       string          `json:"role"`
		Active     bool            `json:"active"`
		TelegramID string          `json:"telegram_id"`
		Areas      []string        `json:"areas"`
		Shifts     []Shift         `json:"shifts"`
		DaysOff    []string        `json:"days_off"`
	}
	if e := decode(w, r, &x); e != nil {
		fail(w, 400, e.Error())
		return
	}
	x.Email = strings.ToLower(strings.TrimSpace(x.Email))
	if !strings.Contains(x.Email, "@") || x.Name == "" || x.Role != "mechanic" {
		fail(w, 400, "Укажите имя, email и роль")
		return
	}
	if len(x.Password) > 72 || (x.Password != "" && len(x.Password) < 12) {
		fail(w, 400, "Пароль: 12–72 байта")
		return
	}
	candidate := User{MainCity: Area(x.MainCity), Phone: x.Phone, Exceptions: x.Exceptions, Areas: x.Areas, Shifts: x.Shifts, DaysOff: x.DaysOff}
	if e := ValidateAvailability(candidate); e != nil {
		fail(w, 400, e.Error())
		return
	}
	for i := range x.Areas {
		x.Areas[i] = Area(x.Areas[i])
	}
	ref := s.Collection("users").Doc(digest(x.Email))
	e := s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
		var prev User
		d, e := tx.Get(ref)
		if e != nil && status.Code(e) != codes.NotFound {
			return e
		}
		if e == nil {
			if e = d.DataTo(&prev); e != nil {
				return e
			}
			if prev.Role == "owner" {
				return ErrForbidden
			}
		}
		hash := prev.Hash
		if x.Password != "" {
			h, e := bcrypt.GenerateFromPassword([]byte(x.Password), 12)
			if e != nil {
				return e
			}
			hash = string(h)
		}
		if hash == "" {
			return errors.New("Для нового пользователя нужен пароль")
		}
		sd, err := tx.Get(s.Collection("schedules").Doc(ref.ID))
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if err == nil {
			var sc schedule
			if err = sd.DataTo(&sc); err != nil {
				return err
			}
			for _, b := range sc.Bookings {
				if b.Proposal.End.After(time.Now()) && !WithinHours(candidate, b.Proposal) {
					return errors.New("Новый график конфликтует с подтверждённым выездом. Сначала перенесите выезд")
				}
			}
		}
		return tx.Set(ref, User{MainCity: Area(x.MainCity), Phone: x.Phone, Exceptions: x.Exceptions, Areas: x.Areas, Shifts: x.Shifts, DaysOff: x.DaysOff, ID: ref.ID, Name: x.Name, Email: x.Email, Role: x.Role, Active: x.Active, Hash: hash, TelegramID: x.TelegramID})
	})
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	write(w, 200, map[string]bool{"ok": true})
}
func (s *Server) jobs(w http.ResponseWriter, r *http.Request, u User) {
	if r.Method == "POST" {
		if !u.Manager() {
			fail(w, 403, "Нет доступа")
			return
		}
		var x struct {
			Area        string `json:"area"`
			Plate       string `json:"plate"`
			Name        string `json:"name"`
			Phone       string `json:"phone"`
			Email       string `json:"email"`
			Address     string `json:"address"`
			Description string `json:"description"`
			Brand       string `json:"brand"`
			Language    string `json:"language"`
		}
		if e := decode(w, r, &x); e != nil {
			fail(w, 400, e.Error())
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if len(key) < 16 || len(key) > 128 {
			fail(w, 400, "Нужен Idempotency-Key (16–128 символов)")
			return
		}
		id := digest(u.ID + ":" + key)
		j, e := NewJob(id, x.Plate, x.Name, x.Phone, x.Address, x.Description, x.Brand, x.Language)
		if e != nil {
			fail(w, 400, e.Error())
			return
		}
		if x.Email != "" {
			a, e := mail.ParseAddress(x.Email)
			if e != nil || a.Address != x.Email {
				fail(w, 400, "Некорректный email")
				return
			}
		}
		book, err := loadPriceBook(r.Context(), s.DB)
		if err != nil {
			fail(w, 503, "Ошибка тарифов")
			return
		}
		applyPrices(&j, book)
		stable, err := resolveVehicle(r.Context(), s.DB, j.Plate)
		if err != nil {
			fail(w, 503, "Ошибка автомобиля")
			return
		}
		j.Vehicle = stable
		j.VehicleID = stable
		j.Email = x.Email
		if x.Area != "" {
			j.Area = Area(x.Area)
		}
		ref := s.Collection("jobs").Doc(id)
		b := s.DB.Batch()
		b.Create(ref, j)
		b.Create(ref.Collection("events").NewDoc(), Event{Actor: u.ID, Type: "created", At: j.Created, Revision: 1, Snapshot: &j})
		_, e = b.Commit(r.Context())
		if status.Code(e) == codes.AlreadyExists {
			d, err := ref.Get(r.Context())
			if err == nil && d.DataTo(&j) == nil {
				write(w, 200, j)
				return
			}
		}
		if e != nil {
			fail(w, 503, "Не удалось сохранить заявку")
			return
		}
		write(w, 201, j)
		return
	}
	if r.Method != "GET" {
		fail(w, 405, "Method not allowed")
		return
	}
	q := s.Collection("jobs").Query
	if !u.Manager() {
		q = q.Where("Mechanic", "==", u.ID)
	}
	if p := r.URL.Query().Get("plate"); p != "" {
		p, e := Plate(p)
		if e != nil {
			fail(w, 400, e.Error())
			return
		}
		alias, err := s.Collection("vehicle_aliases").Doc(p).Get(r.Context())
		if err == nil {
			var a VehicleAlias
			if alias.DataTo(&a) != nil {
				fail(w, 503, "Ошибка номера")
				return
			}
			q = q.Where("Vehicle", "==", a.VehicleID)
		} else if status.Code(err) == codes.NotFound {
			q = q.Where("Plate", "==", p)
		} else {
			fail(w, 503, "Ошибка поиска")
			return
		}
	}
	q = q.OrderBy("Created", firestore.Desc).OrderBy(firestore.DocumentID, firestore.Desc)
	if c := r.URL.Query().Get("cursor"); c != "" {
		if len(c) != 64 {
			fail(w, 400, "Некорректный курсор")
			return
		}
		cursorDoc, err := s.Collection("jobs").Doc(c).Get(r.Context())
		if err != nil {
			fail(w, 400, "Некорректный курсор")
			return
		}
		var cursorJob Job
		if cursorDoc.DataTo(&cursorJob) != nil || !u.CanRead(cursorJob) {
			fail(w, 400, "Некорректный курсор")
			return
		}
		q = q.StartAfter(cursorDoc)
	}
	docs, e := q.Limit(101).Documents(r.Context()).GetAll()
	if e != nil {
		log.Printf("CRM list: %v", e)
		fail(w, 503, "Не удалось прочитать заявки. Проверьте индексы Firestore")
		return
	}
	next := ""
	if len(docs) > 100 {
		docs = docs[:100]
		next = docs[99].Ref.ID
	}
	jobs := []Job{}
	for _, d := range docs {
		var j Job
		if d.DataTo(&j) != nil {
			fail(w, 503, "Ошибка данных заявки")
			return
		}
		jobs = append(jobs, j)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].Created.After(jobs[j].Created) })
	safe := []any{}
	for _, j := range jobs {
		safe = append(safe, jobView(j, u))
	}
	write(w, 200, map[string]any{"jobs": safe, "cursor": next})
}
func (s *Server) readJob(ctx context.Context, id string, u User) (Job, error) {
	if len(id) != 64 {
		return Job{}, ErrForbidden
	}
	d, e := s.Collection("jobs").Doc(id).Get(ctx)
	if e != nil {
		return Job{}, e
	}
	var j Job
	if e = d.DataTo(&j); e != nil {
		return j, e
	}
	if !u.CanRead(j) {
		return j, ErrForbidden
	}
	return j, nil
}
func (s *Server) job(w http.ResponseWriter, r *http.Request, u User, path string) {
	parts := strings.Split(path, "/")
	id := parts[0]
	j, e := s.readJob(r.Context(), id, u)
	if e != nil {
		fail(w, 404, "Заявка не найдена или недоступна")
		return
	}
	if len(parts) == 3 && parts[1] == "intake-photos" {
		s.intakePhoto(w, r, j, parts[2])
		return
	}
	if len(parts) == 2 && parts[1] == "registration" {
		s.changePlate(w, r, u, j)
		return
	}
	if len(parts) == 2 && parts[1] == "messages" {
		s.messages(w, r, u, j)
		return
	}
	if len(parts) == 2 && parts[1] == "emails" {
		s.emails(w, r, u, j)
		return
	}
	if len(parts) == 2 && parts[1] == "vehicle-history" && r.Method == "GET" {
		s.vehicleHistory(w, r, j, u)
		return
	}
	if len(parts) == 4 && parts[1] == "vehicle-photos" && r.Method == "GET" {
		previous, e := s.Collection("jobs").Doc(parts[2]).Get(r.Context())
		var old Job
		if e != nil || previous.DataTo(&old) != nil || vehicleID(old) == "" || vehicleID(old) != vehicleID(j) {
			fail(w, 404, "Фото недоступно")
			return
		}
		s.photos(w, r, u, old, parts[3:])
		return
	}
	if len(parts) > 1 && parts[1] == "photos" {
		s.photos(w, r, u, j, parts[2:])
		return
	}
	if len(parts) > 1 && parts[1] == "calendar.ics" && r.Method == "GET" {
		s.calendar(w, r, j)
		return
	}
	if len(parts) == 1 && r.Method == "GET" {
		write(w, 200, jobResponse(j, u))
		return
	}
	if len(parts) == 2 && parts[1] == "events" && r.Method == "GET" {
		q := s.Collection("jobs").Doc(id).Collection("events").OrderBy("At", firestore.Asc).OrderBy(firestore.DocumentID, firestore.Asc)
		if c := r.URL.Query().Get("cursor"); c != "" {
			d, e := s.Collection("jobs").Doc(id).Collection("events").Doc(c).Get(r.Context())
			if e != nil {
				fail(w, 400, "Некорректный курсор")
				return
			}
			q = q.StartAfter(d)
		}
		ds, e := q.Limit(101).Documents(r.Context()).GetAll()
		if e != nil {
			fail(w, 503, "Не удалось загрузить историю")
			return
		}
		next := ""
		if len(ds) > 100 {
			ds = ds[:100]
			next = ds[99].Ref.ID
		}
		out := []any{}
		for _, d := range ds {
			var x Event
			if d.DataTo(&x) != nil {
				fail(w, 503, "Ошибка истории")
				return
			}
			x.ID = d.Ref.ID
			if v, ok := eventView(x, u); ok {
				out = append(out, v)
			}
		}
		write(w, 200, map[string]any{"events": out, "cursor": next})
		return
	}
	if len(parts) == 2 && parts[1] == "actions" && r.Method == "POST" {
		var a Action
		if e := decode(w, r, &a); e != nil {
			fail(w, 400, e.Error())
			return
		}
		next, e := s.mutate(r.Context(), id, a, u)
		if e != nil {
			code := 400
			if errors.Is(e, ErrConflict) {
				code = 409
			}
			if errors.Is(e, ErrForbidden) {
				code = 403
			}
			fail(w, code, e.Error())
			return
		}
		write(w, 200, jobResponse(next, u))
		return
	}
	http.NotFound(w, r)
}

type booking struct {
	Job      string
	Proposal Proposal
}
type schedule struct{ Bookings []booking }

func (s *Server) mutate(ctx context.Context, id string, a Action, u User) (Job, error) {
	var result Job
	ref := s.Collection("jobs").Doc(id)
	now := time.Now().UTC()
	err := s.DB.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		d, e := tx.Get(ref)
		if e != nil {
			return e
		}
		var j Job
		if e = d.DataTo(&j); e != nil {
			return e
		}
		if a.Kind == "contact" && a.Email != "" {
			address, e := mail.ParseAddress(a.Email)
			if e != nil || address.Address != a.Email {
				return errors.New("Некорректный email")
			}
		}
		n, e := Apply(j, a, u, now)
		if e != nil {
			return e
		}
		if a.Kind == "assign" || a.Kind == "reassign" || a.Kind == "propose" || a.Kind == "confirm" {
			d, e := tx.Get(s.Collection("users").Doc(n.Mechanic))
			if e != nil {
				return errors.New("Механик не найден")
			}
			var m User
			if d.DataTo(&m) != nil || !m.Active || m.Role != "mechanic" {
				return errors.New("Выберите активного механика")
			}
			if e := CheckAvailability(m, n, a.Kind != "assign" && a.Kind != "reassign"); e != nil {
				return e
			}
		}
		var sched schedule
		var sr *firestore.DocumentRef
		touch := a.Kind == "reassign" || a.Kind == "confirm" || a.Kind == "reschedule" || a.Kind == "cancel"
		if touch && j.Mechanic != "" {
			sr = s.Collection("schedules").Doc(j.Mechanic)
			d, e := tx.Get(sr)
			if e != nil && status.Code(e) != codes.NotFound {
				return e
			}
			if e == nil {
				if e = d.DataTo(&sched); e != nil {
					return e
				}
			}
			keep := []booking{}
			for _, b := range sched.Bookings {
				if b.Job == id || b.Proposal.End.Before(now) {
					continue
				}
				if a.Kind == "confirm" && Overlap(n.Proposal, b.Proposal) {
					return errors.New("У механика уже есть выезд в это время. Предложите другое время")
				}
				keep = append(keep, b)
			}
			if a.Kind == "confirm" {
				if len(keep) >= 500 {
					return errors.New("Слишком много будущих выездов у механика")
				}
				keep = append(keep, booking{Job: id, Proposal: n.Proposal})
			}
			sched.Bookings = keep
		}
		if sr != nil {
			if e := tx.Set(sr, sched); e != nil {
				return e
			}
		}
		if a.Kind == "message" {
			mr := ref.Collection("messages").NewDoc()
			m := Message{ID: mr.ID, JobID: id, VehicleID: vehicleID(n), Source: a.Channel, Author: u.ID, At: now, Original: a.Text, Direction: a.Direction, Status: "recorded"}
			if e := tx.Create(mr, m); e != nil {
				return e
			}
		}
		if e := tx.Set(ref, n); e != nil {
			return e
		}
		event := Event{Actor: u.ID, Type: a.Kind, Text: a.Text, Channel: a.Channel, Direction: a.Direction, At: now, Revision: n.Revision, Snapshot: &n}
		if e := tx.Create(ref.Collection("events").NewDoc(), event); e != nil {
			return e
		}
		if a.Kind == "propose" || a.Kind == "confirm" || a.Kind == "assign" || a.Kind == "report" || a.Kind == "reschedule" || a.Kind == "cancel" {
			text := fmt.Sprintf("%s · %s\nКлиент: %s\nОтветить: %s\n%s/crm/?job=%s", n.Plate, a.Kind, n.Name, n.Phone, s.URL, id)
			if a.Kind == "propose" {
				text += "\n\n" + Draft(n)
			}
			if e := tx.Create(s.Collection("outbox").Doc(fmt.Sprintf("%s-%d-owner", id, n.Revision)), Notice{Job: id, Text: text, Target: "owner", Status: "pending", Created: now, Fingerprint: noticeFingerprint(n)}); e != nil {
				return e
			}
			if a.Kind == "assign" || a.Kind == "confirm" || a.Kind == "reschedule" || a.Kind == "cancel" {
				if e := tx.Create(s.Collection("outbox").Doc(fmt.Sprintf("%s-%d-mechanic", id, n.Revision)), Notice{Job: id, Text: text, Target: n.Mechanic, Status: "pending", Created: now, Fingerprint: noticeFingerprint(n)}); e != nil {
					return e
				}
			}
		}
		result = n
		return nil
	})
	return result, err
}
func internalAuthorized(r *http.Request) bool {
	expected := os.Getenv("CRM_INTERNAL_TOKEN")
	given := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return len(expected) >= 32 && subtle.ConstantTimeCompare([]byte(expected), []byte(given)) == 1
}
