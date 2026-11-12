package crm

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type WorkException struct {
	Date   string  `json:"date"`
	Shifts []Shift `json:"shifts"`
}
type Shift struct {
	Day   int    `json:"day"`
	Start string `json:"start"`
	End   string `json:"end"`
}

func minutes(s string) (int, error) {
	if len(s) != 5 || s[2] != ':' {
		return 0, errors.New("Время должно быть HH:MM")
	}
	h, e := strconv.Atoi(s[:2])
	if e != nil || h < 0 || h > 23 {
		return 0, errors.New("Некорректный час")
	}
	m, e := strconv.Atoi(s[3:])
	if e != nil || m < 0 || m > 59 {
		return 0, errors.New("Некорректная минута")
	}
	return h*60 + m, nil
}
func Area(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "хельсинки", "helsinki":
		return "Helsinki"
	case "ванта", "вантаа", "вента", "vantaa":
		return "Vantaa"
	case "эспоо", "эспо", "espoo":
		return "Espoo"
	}
	return strings.TrimSpace(s)
}
func InferArea(address string) string {
	s := strings.ToLower(address)
	for _, v := range []string{"helsinki", "vantaa", "espoo", "хельсинки", "вантаа", "ванта", "эспоо", "эспо"} {
		if strings.Contains(s, v) {
			return Area(v)
		}
	}
	return ""
}
func ValidateAvailability(u User) error {
	if len(u.Exceptions) > 366 {
		return errors.New("Слишком много исключений")
	}
	seen := map[string]bool{}
	for _, x := range u.Exceptions {
		if _, e := time.Parse("2006-01-02", x.Date); e != nil || seen[x.Date] {
			return errors.New("Некорректная дата исключения")
		}
		seen[x.Date] = true
		if e := ValidateAvailability(User{Shifts: x.Shifts}); e != nil {
			return e
		}
	}
	if len(u.Areas) > 30 || len(u.Shifts) > 28 || len(u.DaysOff) > 366 {
		return errors.New("Слишком много зон, смен или выходных")
	}
	for _, a := range u.Areas {
		if strings.TrimSpace(a) == "" || len(a) > 100 {
			return errors.New("Укажите непустые зоны до 100 символов")
		}
	}
	for i, a := range u.Shifts {
		start, e := minutes(a.Start)
		if e != nil {
			return e
		}
		end, e := minutes(a.End)
		if e != nil {
			return e
		}
		if a.Day < 0 || a.Day > 6 || end <= start {
			return errors.New("Смена должна завершаться позже начала в тот же день")
		}
		for _, b := range u.Shifts[:i] {
			bs, _ := minutes(b.Start)
			be, _ := minutes(b.End)
			if a.Day == b.Day && start < be && bs < end {
				return errors.New("Рабочие смены пересекаются")
			}
		}
	}
	for _, d := range u.DaysOff {
		if _, e := time.Parse("2006-01-02", d); e != nil {
			return errors.New("Выходной: YYYY-MM-DD")
		}
	}
	return nil
}
func CanServe(u User, area string) bool {
	if Area(area) == "" {
		return false
	}
	for _, a := range append([]string{u.MainCity}, u.Areas...) {
		if strings.EqualFold(Area(a), Area(area)) {
			return true
		}
	}
	return false
}
func WithinHours(u User, p Proposal) bool {
	a, b := p.Start.In(Helsinki), p.End.In(Helsinki)
	if !b.After(a) || a.Format("2006-01-02") != b.Format("2006-01-02") {
		return false
	}
	for _, d := range u.DaysOff {
		if a.Format("2006-01-02") == d {
			return false
		}
	}
	shifts := u.Shifts
	for _, ex := range u.Exceptions {
		if ex.Date == a.Format("2006-01-02") {
			shifts = append([]Shift{}, ex.Shifts...)
			for i := range shifts {
				shifts[i].Day = int(a.Weekday())
			}
			break
		}
	}
	for _, shift := range shifts {
		if shift.Day != int(a.Weekday()) {
			continue
		}
		start, e := minutes(shift.Start)
		if e != nil {
			continue
		}
		end, e := minutes(shift.End)
		if e != nil {
			continue
		}
		startAt := time.Date(a.Year(), a.Month(), a.Day(), start/60, start%60, 0, 0, Helsinki)
		endAt := time.Date(a.Year(), a.Month(), a.Day(), end/60, end%60, 0, 0, Helsinki)
		if !a.Before(startAt) && !b.After(endAt) {
			return true
		}
	}
	return false
}
func CheckAvailability(u User, j Job, checkTime bool) error {
	if !u.Active || u.Role != "mechanic" {
		return errors.New("Механик недоступен")
	}
	if j.Area == "" {
		return errors.New("Укажите город или зону обслуживания в карточке заявки")
	}
	if !CanServe(u, j.Area) {
		return fmt.Errorf("Механик не обслуживает зону %s", j.Area)
	}
	if checkTime && !WithinHours(u, j.Proposal) {
		return errors.New("Выезд выходит за график механика или попадает на выходной")
	}
	return nil
}
