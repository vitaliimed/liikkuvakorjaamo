package crm

import (
	"google.golang.org/api/iterator"
	"net/http"
	"sort"
)

// Calendar is independent of the paginated search results. Role scoping happens here.
func (s *Server) calendarList(w http.ResponseWriter, r *http.Request, u User) {
	if !u.Active || (u.Role != "owner" && u.Role != "mechanic") {
		fail(w, 403, "Нет доступа")
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
	it := q.Documents(r.Context())
	defer it.Stop()
	jobs := []Job{}
	for {
		d, e := it.Next()
		if e == iterator.Done {
			break
		}
		if e != nil {
			fail(w, 503, "Не удалось загрузить полный календарь")
			return
		}
		var j Job
		if d.DataTo(&j) != nil {
			fail(w, 503, "Ошибка записи календаря")
			return
		}
		if u.CanRead(j) && (j.Status == "scheduled" || j.Status == "in_progress") {
			jobs = append(jobs, j)
		}
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].Proposal.Start.Before(jobs[j].Proposal.Start) })
	out := []any{}
	for _, j := range jobs {
		out = append(out, map[string]any{"id": j.ID, "plate": j.Plate, "name": j.Name, "address": j.Address, "mechanic": j.Mechanic, "status": j.Status, "proposal": map[string]any{"start": j.Proposal.Start, "end": j.Proposal.End}})
	}
	write(w, 200, map[string]any{"jobs": out})
}
