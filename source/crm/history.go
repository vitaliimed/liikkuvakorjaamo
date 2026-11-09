package crm

import (
	"cloud.google.com/go/firestore"
	"net/http"
	"time"
)

type VehicleVisit struct {
	ID          string    `json:"id"`
	Created     time.Time `json:"created"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Model       string    `json:"model"`
	Mileage     int64     `json:"mileage"`
	Proposal    Proposal  `json:"proposal"`
	Report      Report    `json:"report"`
	Photos      []Photo   `json:"photos"`
}

// Assigned mechanics can inspect the technical history for that vehicle,
// without prior customers' names, addresses, phone numbers or conversations.
func (s *Server) vehicleHistory(w http.ResponseWriter, r *http.Request, current Job, u User) {
	q := s.Collection("jobs").Where("Vehicle", "==", current.Vehicle).OrderBy(firestore.DocumentID, firestore.Asc)
	if c := r.URL.Query().Get("cursor"); c != "" {
		if len(c) != 64 {
			fail(w, 400, "Некорректный курсор")
			return
		}
		q = q.StartAfter(c)
	}
	docs, e := q.Limit(101).Documents(r.Context()).GetAll()
	if e != nil {
		fail(w, 503, "Не удалось загрузить историю авто")
		return
	}
	next := ""
	if len(docs) > 100 {
		docs = docs[:100]
		next = docs[99].Ref.ID
	}
	visits := []any{}
	for _, d := range docs {
		var j Job
		if d.DataTo(&j) != nil {
			fail(w, 503, "Ошибка истории авто")
			return
		}
		if u.Manager() {
			visits = append(visits, j)
		} else {
			visits = append(visits, technicalVisit(j))
		}
	}
	write(w, 200, map[string]any{"visits": visits, "cursor": next})
}
