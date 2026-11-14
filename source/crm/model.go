package crm

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"
)

var Helsinki, _ = time.LoadLocation("Europe/Helsinki")
var platePattern = regexp.MustCompile(`^[A-Z0-9]{2,10}$`)
var ErrConflict = errors.New("Запись изменена. Обновите страницу и повторите действие")
var ErrForbidden = errors.New("Нет доступа")

type User struct {
	MainCity   string          `json:"main_city"`
	Phone      string          `json:"phone"`
	Exceptions []WorkException `json:"exceptions"`
	Areas      []string        `json:"areas"`
	Shifts     []Shift         `json:"shifts"`
	DaysOff    []string        `json:"days_off"`
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Email      string          `json:"email"`
	Role       string          `json:"role"`
	Hash       string          `json:"-"`
	Active     bool            `json:"active"`
	TelegramID string          `json:"telegram_id"`
}
type Photo struct {
	PreviewObject string    `json:"-"`
	ID            string    `json:"id"`
	Object        string    `json:"-"`
	Caption       string    `json:"caption"`
	Stage         string    `json:"stage"`
	Author        string    `json:"author"`
	SHA256        string    `json:"sha256"`
	Created       time.Time `json:"created"`
}
type Proposal struct {
	WorkPriceSet *bool     `json:"work_price_set,omitempty"`
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	VisitCents   int64     `json:"visit_cents"`
	WorkCents    int64     `json:"work_cents"`
	PartsCents   int64     `json:"parts_cents"`
	Cause        string    `json:"cause"`
	Plan         string    `json:"plan"`
	Payment      string    `json:"payment"`
}
type Report struct {
	Diagnosis     string `json:"diagnosis"`
	Parts         string `json:"parts"`
	ReceivedCents int64  `json:"received_cents"`
	Work          string `json:"work"`
	Result        string `json:"result"`
	Remaining     string `json:"remaining"`
	Payment       string `json:"payment"`
}
type Agreement struct {
	Proposal Proposal  `json:"proposal"`
	At       time.Time `json:"at"`
	Actor    string    `json:"actor"`
	Revision int64     `json:"revision"`
}

type Job struct {
	Agreed          *Agreement `json:"agreed,omitempty"`
	VehicleID       string     `json:"vehicle_id"`
	PricingSnapshot []Tariff   `json:"pricing_snapshot"`
	Area            string     `json:"area"`
	ID              string     `json:"id"`
	Vehicle         string     `json:"vehicle"`
	Plate           string     `json:"plate"`
	Brand           string     `json:"brand"`
	Name            string     `json:"name"`
	Phone           string     `json:"phone"`
	Email           string     `json:"email"`
	Address         string     `json:"address"`
	Language        string     `json:"language"`
	Description     string     `json:"description"`
	Model           string     `json:"model"`
	VIN             string     `json:"vin"`
	Mileage         int64      `json:"mileage"`
	Mechanic        string     `json:"mechanic"`
	Status          string     `json:"status"`
	Proposal        Proposal   `json:"proposal"`
	Report          Report     `json:"report"`
	Photos          []Photo    `json:"photos"`
	IntakePhotos    []string   `json:"intake_photos"`
	Revision        int64      `json:"revision"`
	Created         time.Time  `json:"created"`
	Updated         time.Time  `json:"updated"`
}
type Event struct {
	ID        string    `json:"id"`
	Actor     string    `json:"actor"`
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	Channel   string    `json:"channel"`
	Direction string    `json:"direction"`
	At        time.Time `json:"at"`
	Revision  int64     `json:"revision"`
	Snapshot  *Job      `json:"snapshot,omitempty"`
}
type Action struct {
	Area      string   `json:"area"`
	Email     string   `json:"email"`
	Kind      string   `json:"kind"`
	Revision  int64    `json:"revision"`
	Mechanic  string   `json:"mechanic"`
	Text      string   `json:"text"`
	Channel   string   `json:"channel"`
	Direction string   `json:"direction"`
	Proposal  Proposal `json:"proposal"`
	Report    Report   `json:"report"`
	Model     string   `json:"model"`
	VIN       string   `json:"vin"`
	Mileage   int64    `json:"mileage"`
}

func Plate(s string) (string, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.NewReplacer("-", "", " ", "", "\t", "").Replace(s)
	if !platePattern.MatchString(s) {
		return "", errors.New("Номер: 2–10 латинских букв или цифр")
	}
	return s, nil
}
func NewJob(id, plate, name, phone, address, description, brand, language string) (Job, error) {
	p, e := Plate(plate)
	if e != nil {
		return Job{}, e
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(phone) == "" || strings.TrimSpace(address) == "" || strings.TrimSpace(description) == "" {
		return Job{}, errors.New("Укажите клиента, телефон, адрес и проблему")
	}
	if brand != "autohelp" {
		brand = "liikkuva"
	}
	if language != "en" && language != "ru" {
		language = "fi"
	}
	now := time.Now().UTC()
	j := Job{Area: InferArea(address), ID: id, Vehicle: "FI-" + p, Plate: p, Name: name, Phone: phone, Address: address, Description: description, Brand: brand, Language: language, Status: "new", Revision: 1, Created: now, Updated: now, Photos: []Photo{}, Proposal: Proposal{VisitCents: 5000, WorkCents: 7000, Payment: "cash"}}
	applyPrices(&j, DefaultPriceBook())
	return j, nil
}
func (u User) Manager() bool { return u.Active && u.Role == "owner" }
func (u User) CanRead(j Job) bool {
	return u.Active && (u.Manager() || (u.Role == "mechanic" && j.Mechanic == u.ID))
}
func Apply(j Job, a Action, u User, now time.Time) (Job, error) {
	if !u.CanRead(j) {
		return j, ErrForbidden
	}
	if a.Revision != j.Revision {
		return j, ErrConflict
	}
	bad := func(s string) (Job, error) { return j, errors.New(s) }
	switch a.Kind {
	case "contact":
		if !u.Manager() {
			return j, ErrForbidden
		}
		if Area(a.Area) != j.Area && (j.Status == "scheduled" || j.Status == "in_progress") {
			return bad("Для изменения зоны сначала перенесите выезд")
		}
		j.Area = Area(a.Area)
		j.Email = strings.TrimSpace(a.Email)
	case "reassign":
		if !u.Manager() {
			return j, ErrForbidden
		}
		if j.Status != "assigned" && j.Status != "proposed" && j.Status != "client_confirmed" && j.Status != "scheduled" {
			return bad("Переназначение недоступно")
		}
		if a.Mechanic == "" || strings.TrimSpace(a.Text) == "" {
			return bad("Укажите механика и причину")
		}
		j.Mechanic = a.Mechanic
		j.Status = "assigned"
		j.Agreed = nil
	case "assign":
		if !u.Manager() {
			return j, ErrForbidden
		}
		if j.Status != "new" && j.Status != "assigned" {
			return bad("Переназначение доступно до предложения времени")
		}
		if a.Mechanic == "" {
			return bad("Выберите механика")
		}
		j.Mechanic = a.Mechanic
		j.Status = "assigned"
		j.Agreed = nil
	case "propose":
		if j.Mechanic == "" || (j.Status != "assigned" && j.Status != "proposed") {
			return bad("Сначала назначьте механика; подтверждённый выезд нужно перенести")
		}
		p := a.Proposal
		if p.WorkPriceSet != nil && !*p.WorkPriceSet {
			return bad("Укажите стоимость работ")
		}
		if !p.Start.After(now) || !p.End.After(p.Start) || p.End.Sub(p.Start) > 12*time.Hour {
			return bad("Укажите будущее время и длительность до 12 часов")
		}
		if p.VisitCents < 0 || p.WorkCents < 0 || p.PartsCents < 0 || p.VisitCents > 10000000 || p.WorkCents > 10000000 || p.PartsCents > 10000000 || p.VisitCents+p.WorkCents+p.PartsCents > 10000000 {
			return bad("Некорректная стоимость")
		}
		if strings.TrimSpace(p.Plan) == "" {
			return bad("Опишите план работ")
		}
		if p.Payment != "cash" && p.Payment != "card" && p.Payment != "transfer" {
			return bad("Выберите способ оплаты")
		}
		j.Proposal = p
		j.Status = "proposed"
		j.Agreed = nil
	case "client_confirm":
		if !u.Manager() {
			return j, ErrForbidden
		}
		if j.Status != "proposed" || strings.TrimSpace(a.Text) == "" {
			return bad("Добавьте сообщение или запись звонка с согласием клиента")
		}
		j.Status = "client_confirmed"
		j.Agreed = &Agreement{Proposal: j.Proposal, At: now, Actor: u.ID, Revision: j.Revision + 1}
	case "confirm":
		if !u.Manager() {
			return j, ErrForbidden
		}
		if j.Status != "client_confirmed" || !j.Proposal.Start.After(now) {
			return bad("Сначала подтвердите согласие клиента и проверьте время")
		}
		j.Status = "scheduled"
	case "reschedule":
		if !u.Manager() {
			return j, ErrForbidden
		}
		if j.Status != "scheduled" && j.Status != "client_confirmed" && j.Status != "proposed" {
			return bad("Перенос недоступен")
		}
		if strings.TrimSpace(a.Text) == "" {
			return bad("Укажите причину переноса")
		}
		j.Status = "assigned"
		j.Agreed = nil
	case "start":
		if j.Status != "scheduled" {
			return bad("Выезд ещё не подтверждён")
		}
		j.Status = "in_progress"
	case "report":
		if j.Status != "in_progress" {
			return bad("Сначала отметьте начало работ")
		}
		x := a.Report
		if strings.TrimSpace(x.Diagnosis) == "" || strings.TrimSpace(x.Parts) == "" {
			return bad("Укажите диагностику и детали (или «не меняли»)")
		}
		if strings.TrimSpace(x.Work) == "" || x.ReceivedCents < 0 || x.ReceivedCents > 10000000 {
			return bad("Укажите выполненные работы и полученную сумму")
		}
		if x.Result != "repaired" && x.Result != "partial" && x.Result != "not_repaired" {
			return bad("Укажите результат")
		}
		if x.Result != "repaired" && strings.TrimSpace(x.Remaining) == "" {
			return bad("Укажите оставшиеся работы и следующий шаг")
		}
		if x.Payment != "cash" && x.Payment != "card" && x.Payment != "transfer" {
			return bad("Укажите способ оплаты")
		}
		j.Report = x
		j.Status = "review"
	case "close":
		if !u.Manager() {
			return j, ErrForbidden
		}
		if j.Status != "review" {
			return bad("Нет отчёта для проверки")
		}
		j.Status = "closed"
	case "reopen":
		if !u.Manager() {
			return j, ErrForbidden
		}
		if j.Status != "review" && j.Status != "closed" {
			return bad("Возврат недоступен")
		}
		if strings.TrimSpace(a.Text) == "" {
			return bad("Укажите причину возврата")
		}
		j.Status = "in_progress"
	case "cancel":
		if !u.Manager() {
			return j, ErrForbidden
		}
		if j.Status == "closed" || j.Status == "cancelled" {
			return bad("Заявка уже закрыта")
		}
		if strings.TrimSpace(a.Text) == "" {
			return bad("Укажите причину отмены")
		}
		j.Status = "cancelled"
	case "technical_note", "internal_note":
		if strings.TrimSpace(a.Text) == "" || a.Channel != "" || a.Direction != "" {
			return bad("Укажите техническую заметку без канала и направления")
		}
	case "message":
		if !u.Manager() {
			return j, ErrForbidden
		}
		if strings.TrimSpace(a.Text) == "" {
			return bad("Введите текст")
		}
		if a.Direction != "incoming" && a.Direction != "outgoing" && a.Direction != "internal" {
			return bad("Укажите направление сообщения")
		}
	case "vehicle":
		if a.Mileage < 0 || a.Mileage > 3000000 {
			return bad("Некорректный пробег")
		}
		j.Model = a.Model
		j.VIN = strings.ToUpper(a.VIN)
		j.Mileage = a.Mileage
	default:
		return bad("Неизвестное действие")
	}
	j.Revision++
	j.Updated = now
	return j, nil
}
func Overlap(a, b Proposal) bool { return a.Start.Before(b.End) && b.Start.Before(a.End) }
func money(c int64) string       { return fmt.Sprintf("%d,%02d €", c/100, c%100) }
func Draft(j Job) string {
	p := j.Proposal
	date := p.Start.In(Helsinki).Format("02.01.2006 15:04")
	total := money(p.VisitCents + p.WorkCents + p.PartsCents)
	payment := map[string]string{"cash": "käteisellä", "card": "kortilla", "transfer": "tilisiirrolla"}[p.Payment]
	switch j.Language {
	case "en":
		return fmt.Sprintf("Thank you for your request! Our mechanic can arrive on %s (Helsinki time). Call-out: %s. Diagnostics and repair work: %s. Parts: %s. Total agreed estimate: %s. Payment: %s. Additional work or parts require your approval. Does this time suit you? Please confirm that you will be at the address and can provide access to the car. Thank you!", date, money(p.VisitCents), money(p.WorkCents), money(p.PartsCents), total, p.Payment)
	case "ru":
		return fmt.Sprintf("Спасибо за заявку! Мастер может приехать %s (время Хельсинки). Выезд: %s. Диагностика и ремонт: %s. Запчасти: %s. Итого по предложению: %s. Оплата: %s. Дополнительные работы и запчасти — после согласования с вами. Подходит ли вам это время? Подтвердите, пожалуйста, что будете по адресу и обеспечите доступ к автомобилю. Спасибо!", date, money(p.VisitCents), money(p.WorkCents), money(p.PartsCents), total, map[string]string{"cash": "наличными", "card": "картой", "transfer": "переводом"}[p.Payment])
	default:
		return fmt.Sprintf("Kiitos yhteydenotostanne! Mekaanikkomme voi saapua %s. Käyntimaksu: %s. Vianmääritys ja korjaustyö: %s. Varaosat: %s. Hinta-arvio yhteensä: %s. Maksu %s. Lisätöistä ja varaosista sovitaan kanssanne erikseen. Sopiiko tämä aika teille? Vahvistattehan, että olette paikalla ja autolle on pääsy sovittuna aikana. Kiitos!", date, money(p.VisitCents), money(p.WorkCents), money(p.PartsCents), total, payment)
	}
}
