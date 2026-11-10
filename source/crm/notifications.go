package crm

import (
	"bytes"
	"cloud.google.com/go/firestore"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Notice struct {
	Fingerprint string    `json:"-"`
	Job         string    `json:"job"`
	Target      string    `json:"target"`
	Text        string    `json:"text"`
	Status      string    `json:"status"`
	Error       string    `json:"error"`
	Attempts    int       `json:"attempts"`
	Created     time.Time `json:"created"`
	Lease       time.Time `json:"lease"`
	Sent        time.Time `json:"sent"`
}

func (s *Server) notifications(w http.ResponseWriter, r *http.Request) {
	ds, e := s.Collection("outbox").OrderBy("Created", firestore.Desc).Limit(100).Documents(r.Context()).GetAll()
	if e != nil {
		fail(w, 503, "Ошибка чтения уведомлений")
		return
	}
	out := []Notice{}
	for _, d := range ds {
		var n Notice
		if d.DataTo(&n) == nil {
			out = append(out, n)
		}
	}
	write(w, 200, out)
}
func telegram(ctx context.Context, chat, text string) error {
	token := os.Getenv("CRM_TELEGRAM_BOT_TOKEN")
	if token == "" || chat == "" {
		return errors.New("Не настроен бот или личный Telegram ID")
	}
	if len([]rune(text)) > 4000 {
		text = string([]rune(text)[:3950]) + "\nПолный текст в CRM"
	}
	b, _ := json.Marshal(map[string]any{"chat_id": chat, "text": text, "disable_web_page_preview": true})
	req, e := http.NewRequestWithContext(ctx, "POST", "https://api.telegram.org/bot"+token+"/sendMessage", bytes.NewReader(b))
	if e != nil {
		return errors.New("Ошибка настройки Telegram")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, e := (&http.Client{Timeout: 12 * time.Second}).Do(req)
	if e != nil {
		return errors.New("Telegram недоступен")
	}
	defer resp.Body.Close()
	var result struct {
		OK bool `json:"ok"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result) != nil || resp.StatusCode != 200 || !result.OK {
		return errors.New("Telegram отклонил сообщение; проверьте ID и запуск бота пользователем")
	}
	return nil
}
func (s *Server) dispatch(w http.ResponseWriter, r *http.Request) {
	sent, failed, e := s.sendPending(r.Context())
	if e != nil {
		fail(w, 503, "Не удалось обработать очередь")
		return
	}
	write(w, 200, map[string]int{"sent": sent, "failed": failed})
}
func (s *Server) sendPending(ctx context.Context) (int, int, error) {
	ds, e := s.Collection("outbox").Where("Status", "==", "pending").Where("Lease", "<=", time.Now()).OrderBy("Lease", firestore.Asc).Limit(50).Documents(ctx).GetAll()
	if e != nil {
		return 0, 0, e
	}
	sent, failed := 0, 0
	for _, d := range ds {
		var n Notice
		claimed := false
		e = s.DB.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			claimed = false
			x, e := tx.Get(d.Ref)
			if e != nil {
				return e
			}
			if e = x.DataTo(&n); e != nil {
				return e
			}
			if n.Status != "pending" || n.Lease.After(time.Now()) {
				return nil
			}
			n.Lease = time.Now().Add(2 * time.Minute)
			n.Attempts++
			claimed = true
			return tx.Set(d.Ref, n)
		})
		if e != nil || !claimed {
			continue
		}
		if n.Job != "" && n.Fingerprint != "" {
			jobDoc, readErr := s.Collection("jobs").Doc(n.Job).Get(ctx)
			var current Job
			if readErr != nil || jobDoc.DataTo(&current) != nil {
				failed++
				continue
			}
			if noticeFingerprint(current) != n.Fingerprint {
				n.Status = "superseded"
				_, err := d.Ref.Set(ctx, n)
				if err != nil {
					return sent, failed, err
				}
				continue
			}
		}
		chat := ""
		if n.Target == "owner" {
			chat = os.Getenv("CRM_OWNER_TELEGRAM_ID")
		} else {
			ud, err := s.Collection("users").Doc(n.Target).Get(ctx)
			var u User
			if err == nil && ud.DataTo(&u) == nil && u.Active {
				chat = u.TelegramID
			}
		}
		e = telegram(ctx, chat, n.Text)
		if e == nil {
			n.Status = "sent"
			n.Sent = time.Now()
			n.Error = ""
			sent++
		} else {
			n.Error = e.Error()
			n.Lease = time.Now().Add(time.Duration(min(n.Attempts, 30)) * time.Minute)
			failed++
		}
		if _, err := d.Ref.Set(ctx, n); err != nil {
			return sent, failed, err
		}
	}
	return sent, failed, nil
}
func (s *Server) tickHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" || !internalAuthorized(r) {
		fail(w, 403, "Нет доступа")
		return
	}
	// One bounded indexed query per tick. Only future scheduled jobs in the next hour.
	now := time.Now().UTC()
	ds, e := s.Collection("jobs").Where("Status", "==", "scheduled").Where("Proposal.Start", ">", now).Where("Proposal.Start", "<=", now.Add(time.Hour)).Limit(100).Documents(r.Context()).GetAll()
	if e != nil {
		fail(w, 503, "Проверьте индекс Status / Proposal.Start")
		return
	}
	for _, d := range ds {
		var j Job
		if d.DataTo(&j) != nil {
			continue
		}
		for _, target := range []string{"owner", j.Mechanic} {
			if target == "" {
				continue
			}
			id := digest(j.ID + "-" + strconv.FormatInt(j.Revision, 10) + "-" + target + "-reminder")
			_, e = s.Collection("outbox").Doc(id).Create(r.Context(), Notice{Job: j.ID, Target: target, Text: fmt.Sprintf("Выезд в течение часа · %s\n%s\n%s\n%s\n%s/crm/?job=%s", j.Plate, j.Proposal.Start.In(Helsinki).Format("02.01 15:04"), j.Address, j.Phone, s.URL, j.ID), Status: "pending", Created: now, Fingerprint: noticeFingerprint(j)})
			if e != nil && status.Code(e) != codes.AlreadyExists {
				fail(w, 503, "Ошибка сохранения напоминания")
				return
			}
		}
	}
	s.dispatch(w, r)
}
func (s *Server) bot(w http.ResponseWriter, r *http.Request) {
	secret := os.Getenv("CRM_TELEGRAM_WEBHOOK_SECRET")
	if r.Method != "POST" || len(secret) < 32 || subtle.ConstantTimeCompare([]byte(secret), []byte(r.Header.Get("X-Telegram-Bot-Api-Secret-Token"))) != 1 {
		fail(w, 403, "Нет доступа")
		return
	}
	var update struct {
		ID      int64 `json:"update_id"`
		Message struct {
			From struct {
				ID int64 `json:"id"`
			} `json:"from"`
			Chat struct {
				ID   int64  `json:"id"`
				Type string `json:"type"`
			} `json:"chat"`
			Text string `json:"text"`
		} `json:"message"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if json.NewDecoder(r.Body).Decode(&update) != nil {
		fail(w, 400, "Invalid update")
		return
	}
	if update.Message.Chat.Type != "private" {
		write(w, 200, map[string]bool{"ok": true})
		return
	}
	actor := strconv.FormatInt(update.Message.From.ID, 10)
	ds, e := s.Collection("users").Where("TelegramID", "==", actor).Limit(2).Documents(r.Context()).GetAll()
	if e != nil {
		fail(w, 503, "Временно недоступно")
		return
	}
	if len(ds) != 1 {
		write(w, 200, map[string]bool{"ok": true})
		return
	}
	var u User
	if ds[0].DataTo(&u) != nil || !u.Active {
		write(w, 200, map[string]bool{"ok": true})
		return
	}
	// A stable outbox ID deduplicates Telegram retries. Mechanic opens the authenticated mobile form.
	text := "Введите номер автомобиля, например ABC-123. Я покажу доступные вам обращения. Отчёт, сумма оплаты и до 10 фото сохраняются в форме CRM."
	p, err := Plate(update.Message.Text)
	if err == nil {
		q := s.Collection("jobs").Where("Plate", "==", p)
		alias, lookupErr := s.Collection("vehicle_aliases").Doc(p).Get(r.Context())
		if lookupErr == nil {
			var a VehicleAlias
			if alias.DataTo(&a) != nil {
				fail(w, 503, "Ошибка номера автомобиля")
				return
			}
			q = s.Collection("jobs").Where("Vehicle", "==", a.VehicleID)
		} else if status.Code(lookupErr) != codes.NotFound {
			fail(w, 503, "Ошибка поиска автомобиля")
			return
		}
		if !u.Manager() {
			q = q.Where("Mechanic", "==", u.ID)
		}
		jobs, err := q.Limit(20).Documents(r.Context()).GetAll()
		if err != nil {
			fail(w, 503, "Не удалось найти заявки")
			return
		}
		text = "По номеру " + p + " доступно обращений: " + strconv.Itoa(len(jobs))
		for _, d := range jobs {
			var j Job
			if d.DataTo(&j) == nil {
				text += "\n\n" + j.Created.In(Helsinki).Format("02.01.2006") + " · " + j.Status + "\n" + j.Address + "\n" + s.URL + "/crm/?job=" + j.ID
			}
		}
	}
	_, e = s.Collection("outbox").Doc("tg-"+strconv.FormatInt(update.ID, 10)).Create(r.Context(), Notice{Target: u.ID, Text: strings.TrimSpace(text), Status: "pending", Created: time.Now().UTC()})
	if e != nil && status.Code(e) != codes.AlreadyExists {
		fail(w, 503, "Ошибка сохранения ответа")
		return
	}
	write(w, 200, map[string]bool{"ok": true})
}

func noticeFingerprint(j Job) string {
	b, _ := json.Marshal(struct {
		Status, Mechanic string
		Proposal         Proposal
		Report           Report
	}{j.Status, j.Mechanic, j.Proposal, j.Report})
	return digest(string(b))
}
