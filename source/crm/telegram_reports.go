package crm

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type MechanicReport struct {
	ID            string        `json:"id"`
	JobID         string        `json:"job_id"`
	Actor         string        `json:"actor"`
	TelegramID    string        `json:"telegram_id"`
	Source        string        `json:"source"`
	Work          string        `json:"work"`
	Problem       string        `json:"problem"`
	Payment       LedgerEntry   `json:"payment"`
	Expenses      []LedgerEntry `json:"expenses"`
	Status        string        `json:"status"`
	Created       time.Time     `json:"created"`
	Reviewed      time.Time     `json:"reviewed"`
	Reviewer      string        `json:"reviewer"`
	LinkedPayment string        `json:"linked_payment"`
	Comment       string        `json:"comment"`
}
type BotSession struct {
	Step         string
	LastUpdate   int64
	Choices      map[string]string
	Draft        MechanicReport
	PendingTitle string
}

var botMenu = [][]string{{"Мои заявки", "Сегодня"}, {"Отчитаться о работе", "Мои отправленные отчёты"}, {"Помощь", "Отмена"}}

func botMoney(s string) (int64, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	parts := strings.Split(s, ".")
	if len(parts) > 2 || len(parts) == 0 || parts[0] == "" {
		return 0, errors.New("Введите сумму, например 150 или 150,50")
	}
	digits := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
		if len(frac) < 1 || len(frac) > 2 {
			return 0, errors.New("Не более двух знаков после запятой")
		}
	}
	for _, c := range digits + frac {
		if c < '0' || c > '9' {
			return 0, errors.New("Введите неотрицательную сумму")
		}
	}
	for len(frac) < 2 {
		frac += "0"
	}
	n, e := strconv.ParseInt(digits+frac, 10, 64)
	if e != nil || n > 100000000 {
		return 0, errors.New("Сумма слишком велика")
	}
	return n, nil
}
func reportSummary(d MechanicReport) string {
	text := fmt.Sprintf("Проблема: %s\nВыполнено: %s\nПолучено: %.2f €\nСпособ: %s", d.Problem, d.Work, float64(d.Payment.Cents)/100, d.Payment.Method)
	var sum int64
	for _, x := range d.Payment.Items {
		sum += x.Cents
		text += fmt.Sprintf("\n%s: %.2f €", x.Title, float64(x.Cents)/100)
	}
	text += fmt.Sprintf("\nНе распределено: %.2f €", float64(d.Payment.Cents-sum)/100)
	for _, x := range d.Expenses {
		text += fmt.Sprintf("\nРасход: %s — %.2f € (%s)", x.Note, float64(x.Cents)/100, x.Participant)
	}
	return text
}
func advanceReport(b BotSession, text string) (BotSession, string, [][]string, bool, error) {
	bad := func(msg string) (BotSession, string, [][]string, bool, error) {
		return b, "", nil, false, errors.New(msg)
	}
	money := func() (int64, error) { return botMoney(text) }
	switch b.Step {
	case "problem":
		if text != "Оставить проблему" {
			b.Draft.Problem = text
		}
		b.Step = "work"
		return b, "Что выполнено?", nil, false, nil
	case "work":
		if strings.TrimSpace(text) == "" || len(text) > 4000 {
			return bad("Опишите выполненную работу до 4000 символов")
		}
		b.Draft.Work = text
		b.Step = "amount"
		return b, "Сколько денег получено? Если не оплачено — 0.", nil, false, nil
	case "amount":
		n, e := money()
		if e != nil {
			return bad(e.Error())
		}
		b.Draft.Payment.Cents = n
		b.Step = "method"
		return b, "Как получены деньги?", [][]string{{"Наличные", "Банковский перевод"}, {"Карта", "Другое"}}, false, nil
	case "method":
		m := map[string]string{"Наличные": "cash", "Банковский перевод": "transfer", "Карта": "card", "Другое": "other"}[text]
		if m == "" {
			return bad("Выберите способ оплаты кнопкой")
		}
		b.Draft.Payment.Method = m
		b.Step = "items"
		return b, "Распределить сумму по работам?", [][]string{{"Добавить позицию", "Без детализации"}}, false, nil
	case "items":
		if text == "Добавить позицию" {
			b.Step = "item_title"
			return b, "Название работы или услуги", nil, false, nil
		}
		if text != "Без детализации" && text != "Далее к расходам" {
			return bad("Выберите действие кнопкой")
		}
		b.Step = "expenses"
		return b, "Были расходы?", [][]string{{"Добавить расход", "Нет расходов"}}, false, nil
	case "item_title":
		if strings.TrimSpace(text) == "" || len(text) > 200 {
			return bad("Название до 200 символов")
		}
		b.PendingTitle = text
		b.Step = "item_amount"
		return b, "Сумма этой позиции, €", nil, false, nil
	case "item_amount":
		n, e := money()
		if e != nil || n <= 0 {
			return bad("Укажите положительную сумму")
		}
		sum := n
		for _, x := range b.Draft.Payment.Items {
			sum += x.Cents
		}
		if sum > b.Draft.Payment.Cents {
			return bad("Позиции превышают полученную сумму")
		}
		b.Draft.Payment.Items = append(b.Draft.Payment.Items, PaymentItem{Title: b.PendingTitle, Cents: n})
		b.Step = "items"
		return b, "Позиция добавлена", [][]string{{"Добавить позицию", "Далее к расходам"}}, false, nil
	case "expenses":
		if text == "Добавить расход" {
			b.Step = "expense_title"
			return b, "На что потрачено?", nil, false, nil
		}
		if text != "Нет расходов" && text != "К итогу" {
			return bad("Выберите действие кнопкой")
		}
		b.Step = "summary"
		return b, reportSummary(b.Draft), [][]string{{"Отправить отчёт", "Изменить"}, {"Отмена"}}, false, nil
	case "expense_title":
		if strings.TrimSpace(text) == "" || len(text) > 200 {
			return bad("Название до 200 символов")
		}
		b.PendingTitle = text
		b.Step = "expense_amount"
		return b, "Сумма расхода, €", nil, false, nil
	case "expense_amount":
		n, e := money()
		if e != nil || n <= 0 {
			return bad("Укажите положительную сумму")
		}
		b.Draft.Expenses = append(b.Draft.Expenses, LedgerEntry{Kind: "expense", Cents: n, Note: b.PendingTitle, Method: "other", Date: b.Draft.Payment.Date})
		b.Step = "expense_payer"
		return b, "Кто оплатил расход?", [][]string{{"Я оплатил", "Владелец оплатил"}}, false, nil
	case "expense_payer":
		p := b.Draft.Payment.Participant
		if text == "Владелец оплатил" {
			p = "owner"
		} else if text != "Я оплатил" {
			return bad("Выберите плательщика кнопкой")
		}
		b.Draft.Expenses[len(b.Draft.Expenses)-1].Participant = p
		b.Step = "expenses"
		return b, "Расход добавлен", [][]string{{"Добавить расход", "К итогу"}}, false, nil
	case "summary":
		if text == "Изменить" {
			b.Draft.Payment.Items = nil
			b.Draft.Expenses = nil
			b.Step = "problem"
			return b, "Проблема: " + b.Draft.Problem, [][]string{{"Оставить проблему"}}, false, nil
		}
		if text != "Отправить отчёт" {
			return bad("Проверьте итог и выберите действие")
		}
		return b, "Отчёт отправлен владельцу. Деньги будут учтены после подтверждения.", botMenu, true, nil
	}
	return bad("Выберите заявку")
}

func (s *Server) handleBotReport(ctx context.Context, u User, updateID int64, text string) error {
	receipt := s.Collection("telegram_updates").Doc(strconv.FormatInt(updateID, 10))
	session := s.Collection("telegram_sessions").Doc(u.ID)
	return s.DB.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, e := tx.Get(receipt); e == nil {
			return nil
		} else if status.Code(e) != codes.NotFound {
			return e
		}
		ud, e := tx.Get(s.Collection("users").Doc(u.ID))
		if e != nil {
			return e
		}
		var active User
		if e = ud.DataTo(&active); e != nil {
			return e
		}
		if !active.Active || active.TelegramID != u.TelegramID || active.Role != u.Role {
			return ErrForbidden
		}
		var b BotSession
		d, e := tx.Get(session)
		if e == nil {
			if e = d.DataTo(&b); e != nil {
				return e
			}
		} else if status.Code(e) != codes.NotFound {
			return e
		}

		b.LastUpdate = updateID
		reply := ""
		keyboard := botMenu
		submit := false
		now := time.Now().UTC()
		menu := text == "/start" || text == "Помощь" || text == "Отмена"
		listing := text == "Мои заявки" || text == "Сегодня" || text == "Отчитаться о работе"
		plate, plateErr := Plate(text)
		if menu {
			b.Step = ""
			b.Choices = nil
			reply = "Выберите заявку, затем заполните отчёт. Можно ввести номер ABC-123 или 123-ABC. Отмена сбрасывает черновик."
		} else if text == "Мои отправленные отчёты" {
			q := s.Collection("mechanic_reports").Where("Actor", "==", u.ID)
			docs, e := tx.Documents(q).GetAll()
			if e != nil {
				return e
			}
			reply = "Ваши отчёты:"
			for _, d := range docs {
				var x MechanicReport
				if d.DataTo(&x) != nil {
					return errors.New("Ошибка отчёта")
				}
				reply += "\n" + x.Created.In(Helsinki).Format("02.01 15:04") + " · " + x.Status + " · " + x.Work
			}
		} else if listing || plateErr == nil {
			q := s.Collection("jobs").Query
			if !u.Manager() {
				q = q.Where("Mechanic", "==", u.ID)
			}
			docs, e := tx.Documents(q).GetAll()
			if e != nil {
				return e
			}
			b.Choices = map[string]string{}
			keyboard = nil
			reply = "Выберите конкретную заявку:"
			b.Step = "choose"
			for _, d := range docs {
				var j Job
				if d.DataTo(&j) != nil {
					return errors.New("Ошибка заявки")
				}
				if !u.CanRead(j) {
					continue
				}
				if plateErr == nil && j.Plate != plate {
					continue
				}
				if listing && (j.Status == "closed" || j.Status == "cancelled") {
					continue
				}
				if text == "Сегодня" && j.Proposal.Start.In(Helsinki).Format("2006-01-02") != now.In(Helsinki).Format("2006-01-02") {
					continue
				}
				label := fmt.Sprintf("%s · %s · %d", j.Plate, j.Created.In(Helsinki).Format("02.01"), len(b.Choices)+1)
				b.Choices[label] = j.ID
				keyboard = append(keyboard, []string{label})
				if len(b.Choices) >= 50 {
					break
				}
			}
			if len(b.Choices) == 0 {
				reply = "Доступных заявок нет. Можно найти по номеру автомобиля."
			}
			keyboard = append(keyboard, []string{"Отмена"})
		} else if b.Step == "choose" {
			id := b.Choices[text]
			if id == "" {
				reply = "Выберите заявку из списка или введите номер автомобиля."
			} else {
				d, e := tx.Get(s.Collection("jobs").Doc(id))
				if e != nil {
					return e
				}
				var j Job
				if e = d.DataTo(&j); e != nil {
					return e
				}
				if !u.CanRead(j) {
					return ErrForbidden
				}
				b.Draft = MechanicReport{ID: "tg-" + strconv.FormatInt(updateID, 10), JobID: j.ID, Actor: u.ID, TelegramID: u.TelegramID, Source: "telegram", Problem: j.Description, Status: "pending", Created: now, Payment: LedgerEntry{Kind: "payment", Participant: u.ID, Date: now.In(Helsinki).Format("2006-01-02")}}
				if u.Manager() {
					b.Draft.Payment.Participant = "owner"
				}
				b.Step = "problem"
				reply = j.Plate + "\nКакая была проблема?\n" + j.Description
				keyboard = [][]string{{"Оставить проблему"}, {"Отмена"}}
			}
		} else if b.Step != "" {
			next, msg, k, done, e := advanceReport(b, text)
			if e != nil {
				reply = e.Error() + "\nПовторите ввод или нажмите Отмена."
				keyboard = [][]string{{"Отмена"}}
			} else {
				b, reply, keyboard, submit = next, msg, k, done
				if keyboard == nil {
					keyboard = [][]string{{"Отмена"}}
				}
			}
		} else {
			reply = "Выберите «Отчитаться о работе» или введите номер автомобиля."
		}
		if submit {
			d, e := tx.Get(s.Collection("jobs").Doc(b.Draft.JobID))
			if e != nil {
				return e
			}
			var j Job
			if e = d.DataTo(&j); e != nil {
				return e
			}
			if !u.CanRead(j) {
				return ErrForbidden
			}
			b.Draft.Payment.Note = "По отчёту механика"
			if e = tx.Create(s.Collection("mechanic_reports").Doc(b.Draft.ID), b.Draft); e != nil {
				return e
			}
			if e = tx.Create(s.Collection("outbox").Doc(b.Draft.ID+"-owner"), Notice{Job: j.ID, Target: "owner", Text: "Новый отчёт механика · " + j.Plate + "\n" + reportSummary(b.Draft) + "\n" + s.URL + "/crm/?job=" + j.ID, Status: "pending", Created: now}); e != nil {
				return e
			}
			b.Step = ""
		}
		encoded, _ := json.Marshal(b)
		if len(encoded) > 500000 {
			return errors.New("Черновик слишком большой")
		}
		if e = tx.Set(session, b); e != nil {
			return e
		}
		if e = tx.Create(receipt, map[string]any{"Actor": u.ID, "At": now}); e != nil {
			return e
		}
		return tx.Create(s.Collection("outbox").Doc("tg-reply-"+strconv.FormatInt(updateID, 10)), Notice{Target: u.ID, Text: reply, Keyboard: encodeKeyboard(keyboard), Status: "pending", Created: now})
	})
}

type ReviewReportRequest struct {
	ID             string `json:"id"`
	Decision       string `json:"decision"`
	PaymentID      string `json:"payment_id"`
	Comment        string `json:"comment"`
	LedgerRevision int64  `json:"ledger_revision"`
}

func (s *Server) mechanicReports(w http.ResponseWriter, r *http.Request, u User, j Job) {
	if !u.Manager() {
		fail(w, 403, "Отчёты подтверждает владелец")
		return
	}
	if r.Method == "GET" {
		docs, e := s.Collection("mechanic_reports").Where("JobID", "==", j.ID).Documents(r.Context()).GetAll()
		if e != nil {
			fail(w, 503, "Ошибка загрузки отчётов")
			return
		}
		out := []MechanicReport{}
		for _, d := range docs {
			var x MechanicReport
			if d.DataTo(&x) != nil {
				fail(w, 503, "Ошибка отчёта")
				return
			}
			out = append(out, x)
		}
		write(w, 200, out)
		return
	}
	if r.Method != "POST" {
		fail(w, 405, "Method not allowed")
		return
	}
	var q ReviewReportRequest
	if decode(w, r, &q) != nil || !ledgerKey.MatchString(q.ID) || (q.Decision != "approve" && q.Decision != "revise") {
		fail(w, 400, "Некорректное подтверждение")
		return
	}
	err := s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
		ref := s.Collection("mechanic_reports").Doc(q.ID)
		d, e := tx.Get(ref)
		if e != nil {
			return e
		}
		var x MechanicReport
		if e = d.DataTo(&x); e != nil {
			return e
		}
		if x.JobID != j.ID {
			return ErrForbidden
		}
		if x.Status != "pending" {
			if x.Comment == q.Comment && ((q.Decision == "approve" && x.Status == "approved" && x.LinkedPayment == q.PaymentID) || (q.Decision == "revise" && x.Status == "revision_requested")) {
				return nil
			}
			return ErrConflict
		}
		jr := s.Collection("jobs").Doc(j.ID)
		d, e = tx.Get(jr)
		if e != nil {
			return e
		}
		var current Job
		if e = d.DataTo(&current); e != nil {
			return e
		}
		lr := s.Collection("ledgers").Doc(j.ID)
		l := newLedger(current)
		d, e = tx.Get(lr)
		if e == nil {
			if e = d.DataTo(&l); e != nil {
				return e
			}
		} else if status.Code(e) != codes.NotFound {
			return e
		}
		if l.Revision != q.LedgerRevision {
			return ErrConflict
		}
		now := time.Now().UTC()
		x.Reviewer = u.ID
		x.Reviewed = now
		x.Comment = q.Comment
		if q.Decision == "revise" {
			if strings.TrimSpace(q.Comment) == "" {
				return errors.New("Укажите что исправить")
			}
			x.Status = "revision_requested"
		} else {
			x.Status = "approved"
			x.LinkedPayment = q.PaymentID
			if q.PaymentID != "" {
				found := false
				for _, entry := range l.Entries {
					if entry.ID == q.PaymentID && (entry.Kind == "payment" || entry.Kind == "legacy_payment") && entry.Cents == x.Payment.Cents && entry.Participant == x.Payment.Participant {
						found = true
					}
					if entry.Kind == "reverse" && entry.Reverses == q.PaymentID {
						return errors.New("Этот платёж отменён")
					}
				}
				if !found {
					return errors.New("Выберите платёж с той же суммой и получателем")
				}
				docs, e := tx.Documents(s.Collection("mechanic_reports").Where("JobID", "==", j.ID)).GetAll()
				if e != nil {
					return e
				}
				for _, d := range docs {
					var old MechanicReport
					if e = d.DataTo(&old); e != nil {
						return e
					}
					if old.Status == "approved" && (old.LinkedPayment == q.PaymentID || old.ID+"-payment" == q.PaymentID) {
						return errors.New("Этот платёж уже связан с другим отчётом")
					}
				}
			} else if x.Payment.Cents > 0 {
				payment := x.Payment
				payment.ID = x.ID + "-payment"
				l, e = applyLedger(l, LedgerRequest{Revision: l.Revision, Entry: payment}, u, current)
				if e != nil {
					return e
				}
			}
			for i, expense := range x.Expenses {
				expense.ID = x.ID + "-expense-" + stringID(i)
				l, e = applyLedger(l, LedgerRequest{Revision: l.Revision, Entry: expense}, u, current)
				if e != nil {
					return e
				}
			}
			if e = tx.Set(lr, l); e != nil {
				return e
			}
		}
		if e = tx.Set(ref, x); e != nil {
			return e
		}
		if e = tx.Create(jr.Collection("events").NewDoc(), Event{Actor: u.ID, Type: "mechanic_report_" + x.Status, Text: x.Work + "\n" + q.Comment, At: now, Revision: current.Revision}); e != nil {
			return e
		}
		return tx.Create(s.Collection("outbox").Doc(x.ID+"-"+x.Status), Notice{Target: x.Actor, Text: "Отчёт: " + x.Status + "\n" + q.Comment, Status: "pending", Created: now})
	})
	if err != nil {
		fail(w, 409, err.Error())
		return
	}
	write(w, 200, map[string]bool{"ok": true})
}

func encodeKeyboard(rows [][]string) string { b, _ := json.Marshal(rows); return string(b) }

// Deliver only this conversation reply immediately. Failed delivery remains in
// the existing outbox for retry by the internal tick; finance is already durable.
func (s *Server) deliverBotReply(ctx context.Context, u User, updateID int64) {
	ref := s.Collection("outbox").Doc("tg-reply-" + strconv.FormatInt(updateID, 10))
	var n Notice
	claimed := false
	err := s.DB.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		claimed = false
		d, e := tx.Get(ref)
		if e != nil {
			return e
		}
		if e = d.DataTo(&n); e != nil {
			return e
		}
		if n.Status != "pending" || n.Lease.After(time.Now()) {
			return nil
		}
		n.Lease = time.Now().Add(2 * time.Minute)
		n.Attempts++
		claimed = true
		return tx.Set(ref, n)
	})
	if err != nil || !claimed {
		return
	}
	var keyboard [][]string
	if n.Keyboard != "" && json.Unmarshal([]byte(n.Keyboard), &keyboard) != nil {
		return
	}
	err = telegramKeyboard(ctx, u.TelegramID, n.Text, keyboard)
	if err == nil {
		n.Status = "sent"
		n.Sent = time.Now()
		n.Error = ""
	} else {
		n.Error = err.Error()
		n.Lease = time.Now().Add(time.Minute)
	}
	_, _ = ref.Set(ctx, n)
}
