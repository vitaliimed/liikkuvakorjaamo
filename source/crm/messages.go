package crm

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strings"
	"time"
)

// A job is a conversation; every message also references the persistent vehicle.
type Message struct {
	ID          string    `json:"id"`
	JobID       string    `json:"job_id"`
	VehicleID   string    `json:"vehicle_id"`
	Source      string    `json:"source"`
	Author      string    `json:"author"`
	At          time.Time `json:"at"`
	Original    string    `json:"original"`
	Language    string    `json:"language"`
	Translation string    `json:"translation"`
	Attachments []string  `json:"attachments"`
	Direction   string    `json:"direction"`
	Status      string    `json:"status"`
}
type Translator interface {
	Translate(context.Context, string, string, string) (string, error)
}

var ErrTranslatorUnavailable = errors.New("Переводчик не подключён")

func TranslateToRussian(ctx context.Context, t Translator, text, language string) (string, error) {
	if language == "ru" {
		return text, nil
	}
	if t == nil {
		return "", ErrTranslatorUnavailable
	}
	if language != "fi" && language != "en" {
		return "", errors.New("Неподдерживаемый язык")
	}
	return t.Translate(ctx, text, language, "ru")
}

// Internal adapter boundary: callers authenticate the provider and resolve the
// verified customer/job relationship. This is deliberately not a public webhook.
func (s *Server) RecordInboundMessage(ctx context.Context, jobID, source, externalID, author, original, language string, attachments []string) (Message, error) {
	if len(jobID) != 64 || externalID == "" || len(externalID) > 512 || len(original) > 40000 || len(attachments) > 10 {
		return Message{}, errors.New("Некорректное сообщение")
	}
	switch source {
	case "whatsapp", "telegram", "email", "site":
	default:
		return Message{}, errors.New("Неизвестный канал")
	}
	ref := s.Collection("jobs").Doc(jobID)
	id := digest(source + ":" + externalID)
	var m Message
	e := s.DB.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		d, e := tx.Get(ref)
		if e != nil {
			return e
		}
		var j Job
		if e = d.DataTo(&j); e != nil {
			return e
		}
		mr := ref.Collection("messages").Doc(id)
		d, e = tx.Get(mr)
		if e == nil {
			return d.DataTo(&m)
		}
		if status.Code(e) != codes.NotFound {
			return e
		}
		m = Message{ID: id, JobID: jobID, VehicleID: vehicleID(j), Source: source, Author: author, At: time.Now().UTC(), Original: original, Language: language, Attachments: attachments, Direction: "incoming", Status: "received"}
		return tx.Create(mr, m)
	})
	return m, e
}
func (s *Server) messages(w http.ResponseWriter, r *http.Request, u User, j Job) {
	if !u.Manager() {
		fail(w, 403, "Переписка доступна только владельцу")
		return
	}
	if r.Method != "GET" {
		fail(w, 405, "Method not allowed")
		return
	}
	q := s.Collection("jobs").Doc(j.ID).Collection("messages").OrderBy("At", firestore.Asc).OrderBy(firestore.DocumentID, firestore.Asc)
	if c := r.URL.Query().Get("cursor"); c != "" {
		d, e := s.Collection("jobs").Doc(j.ID).Collection("messages").Doc(c).Get(r.Context())
		if e != nil {
			fail(w, 400, "Некорректный курсор")
			return
		}
		q = q.StartAfter(d)
	}
	ds, e := q.Limit(101).Documents(r.Context()).GetAll()
	if e != nil {
		fail(w, 503, "Ошибка переписки")
		return
	}
	next := ""
	if len(ds) > 100 {
		ds = ds[:100]
		next = ds[99].Ref.ID
	}
	out := []Message{}
	for _, d := range ds {
		var m Message
		if d.DataTo(&m) != nil {
			fail(w, 503, "Ошибка сообщения")
			return
		}
		out = append(out, m)
	}
	write(w, 200, map[string]any{"messages": out, "cursor": next})
}

func emailMessage(j Job, m Email) Message {
	return Message{ID: "email-" + m.ID, JobID: j.ID, VehicleID: vehicleID(j), Source: "email", Author: m.Actor, At: m.Created, Original: m.Body, Direction: m.Direction, Status: m.Status}
}

// TranslateMessage centralizes translation and stores the result without changing
// the original. Provider/network calls happen outside Firestore transactions.
func (s *Server) TranslateMessage(ctx context.Context, u User, jobID, messageID string, t Translator) (Message, error) {
	if !u.Manager() {
		return Message{}, ErrForbidden
	}
	j, e := s.readJob(ctx, jobID, u)
	if e != nil {
		return Message{}, e
	}
	_ = j
	if messageID == "" || strings.Contains(messageID, "/") {
		return Message{}, errors.New("Некорректный ID сообщения")
	}
	ref := s.Collection("jobs").Doc(jobID).Collection("messages").Doc(messageID)
	d, e := ref.Get(ctx)
	if e != nil {
		return Message{}, e
	}
	var original Message
	if e = d.DataTo(&original); e != nil {
		return Message{}, e
	}
	translated, e := TranslateToRussian(ctx, t, original.Original, original.Language)
	if e != nil {
		return original, e
	}
	var result Message
	e = s.DB.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		d, e := tx.Get(ref)
		if e != nil {
			return e
		}
		if e = d.DataTo(&result); e != nil {
			return e
		}
		if result.Original != original.Original || result.Language != original.Language {
			return ErrConflict
		}
		result.Translation = translated
		return tx.Update(ref, []firestore.Update{{Path: "Translation", Value: translated}})
	})
	return result, e
}
