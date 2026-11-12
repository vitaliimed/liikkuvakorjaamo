package crm

import (
	"bytes"
	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) photos(w http.ResponseWriter, r *http.Request, u User, j Job, parts []string) {
	if s.Bucket == "" {
		fail(w, 503, "Не настроен приватный CRM_PHOTO_BUCKET")
		return
	}
	client, e := storage.NewClient(r.Context())
	if e != nil {
		fail(w, 503, "Хранилище недоступно")
		return
	}
	defer client.Close()
	if r.Method == "GET" && len(parts) == 1 {
		for _, p := range j.Photos {
			if p.ID != parts[0] {
				continue
			}
			object := p.Object
			if p.PreviewObject != "" && r.URL.Query().Get("original") != "1" {
				object = p.PreviewObject
			}
			rd, e := client.Bucket(s.Bucket).Object(object).NewReader(r.Context())
			if e != nil {
				fail(w, 404, "Фото не найдено")
				return
			}
			defer rd.Close()
			w.Header().Set("Content-Type", rd.Attrs.ContentType)
			w.Header().Set("Content-Disposition", "inline")
			_, _ = io.Copy(w, rd)
			return
		}
		fail(w, 404, "Фото не найдено")
		return
	}
	if r.Method != "POST" || len(parts) != 0 {
		fail(w, 405, "Method not allowed")
		return
	}
	select {
	case photoProcessing <- struct{}{}:
		defer func() { <-photoProcessing }()
	default:
		fail(w, 429, "Обрабатывается другое фото; повторите позже")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 21<<20)
	if e = r.ParseMultipartForm(21 << 20); e != nil {
		fail(w, 400, "Фото до 20 МБ")
		return
	}
	defer r.MultipartForm.RemoveAll()
	rev, e := strconv.ParseInt(r.FormValue("revision"), 10, 64)
	if e != nil || rev != j.Revision {
		fail(w, 409, ErrConflict.Error())
		return
	}
	stage := r.FormValue("stage")
	caption := strings.TrimSpace(r.FormValue("caption"))
	if stage != "before" && stage != "during" && stage != "after" {
		fail(w, 400, "Выберите этап фото")
		return
	}
	if len(caption) > 1000 {
		fail(w, 400, "Описание до 1000 байт")
		return
	}
	f, _, e := r.FormFile("photo")
	if e != nil {
		fail(w, 400, "Выберите фото")
		return
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (20<<20)+1))
	if e != nil || len(b) > 20<<20 {
		fail(w, 400, "Фото до 20 МБ")
		return
	}
	cfg, format, e := image.DecodeConfig(bytes.NewReader(b))
	if e != nil || (format != "jpeg" && format != "png") || cfg.Width*cfg.Height > 60000000 {
		fail(w, 400, "Допустимы JPEG / PNG до 60 мегапикселей")
		return
	}
	if len(j.Photos) >= 10 {
		fail(w, 400, "Не более 10 фотографий на обращение")
		return
	}
	p := Photo{ID: randomID(), Caption: caption, Stage: stage, Author: u.ID, SHA256: digest(string(b)), Created: time.Now().UTC()}
	p.Object = "crm/" + j.ID + "/" + p.ID
	preview, e := thumbnail(b)
	if e != nil {
		fail(w, 400, "Повреждённое изображение")
		return
	}
	p.PreviewObject = p.Object + "-preview.jpg"
	obj := client.Bucket(s.Bucket).Object(p.Object).If(storage.Conditions{DoesNotExist: true})
	wr := obj.NewWriter(r.Context())
	wr.ContentType = "image/" + format
	wr.Metadata = map[string]string{"sha256": p.SHA256, "author": u.ID, "job": j.ID}
	if _, e = wr.Write(b); e != nil {
		_ = wr.Close()
		fail(w, 503, "Фото не сохранено")
		return
	}
	if e = wr.Close(); e != nil {
		fail(w, 503, "Фото не сохранено")
		return
	}
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = client.Bucket(s.Bucket).Object(p.Object).Delete(ctx)
		_ = client.Bucket(s.Bucket).Object(p.PreviewObject).Delete(ctx)
	}
	pw := client.Bucket(s.Bucket).Object(p.PreviewObject).If(storage.Conditions{DoesNotExist: true}).NewWriter(r.Context())
	pw.ContentType = "image/jpeg"
	if _, e = pw.Write(preview); e != nil {
		_ = pw.Close()
		cleanup()
		fail(w, 503, "Ошибка копии фото")
		return
	}
	if e = pw.Close(); e != nil {
		cleanup()
		fail(w, 503, "Ошибка копии фото")
		return
	}
	ref := s.Collection("jobs").Doc(j.ID)
	e = s.DB.RunTransaction(r.Context(), func(ctx context.Context, tx *firestore.Transaction) error {
		d, e := tx.Get(ref)
		if e != nil {
			return e
		}
		var live Job
		if e = d.DataTo(&live); e != nil {
			return e
		}
		if !u.CanRead(live) {
			return ErrForbidden
		}
		if live.Revision != rev {
			return ErrConflict
		}
		if len(live.Photos) >= 10 {
			return fmt.Errorf("Не более 10 фотографий")
		}
		live.Photos = append(live.Photos, p)
		live.Revision++
		live.Updated = p.Created
		if e = tx.Set(ref, live); e != nil {
			return e
		}
		return tx.Create(ref.Collection("events").NewDoc(), Event{Actor: u.ID, Type: "photo", Text: stage + ": " + caption, At: p.Created, Revision: live.Revision, Snapshot: &live})
	})
	if e != nil {
		cleanup()
		fail(w, 409, e.Error())
		return
	}
	write(w, 201, p)
}
func icsEscape(s string) string {
	return strings.NewReplacer("\\", "\\\\", "\r", "", "\n", "\\n", ",", "\\,", ";", "\\;").Replace(s)
}
func foldICS(s string) string {
	var out strings.Builder
	count := 0
	for _, r := range s {
		n := len(string(r))
		if count+n > 73 {
			out.WriteString("\r\n ")
			count = 1
		}
		out.WriteRune(r)
		count += n
	}
	return out.String()
}
func (s *Server) calendar(w http.ResponseWriter, r *http.Request, j Job) {
	if j.Status != "scheduled" && j.Status != "in_progress" && j.Status != "review" && j.Status != "closed" {
		fail(w, 400, "Выезд ещё не подтверждён")
		return
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="visit.ics"`)
	lines := []string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//Liikkuva//CRM//RU", "BEGIN:VEVENT", "UID:" + j.ID + "@liikkuva-crm", "SEQUENCE:" + strconv.FormatInt(j.Revision, 10), "DTSTAMP:" + j.Updated.UTC().Format("20060102T150405Z"), "DTSTART:" + j.Proposal.Start.UTC().Format("20060102T150405Z"), "DTEND:" + j.Proposal.End.UTC().Format("20060102T150405Z"), "SUMMARY:" + icsEscape(j.Plate+" · "+j.Name), "LOCATION:" + icsEscape(j.Address), "DESCRIPTION:" + icsEscape(j.Phone+"\n"+j.Description+"\n"+s.URL+"/crm/?job="+j.ID), "BEGIN:VALARM", "TRIGGER:-PT60M", "ACTION:DISPLAY", "DESCRIPTION:Выезд механика", "END:VALARM", "END:VEVENT", "END:VCALENDAR"}
	for _, l := range lines {
		fmt.Fprint(w, foldICS(l)+"\r\n")
	}
}
