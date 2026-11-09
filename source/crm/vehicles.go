package crm

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"time"
)

type VehicleAlias struct {
	VehicleID string
	Plate     string
}

func vehicleID(j Job) string {
	if j.VehicleID != "" {
		return j.VehicleID
	}
	return j.Vehicle
}
func resolveVehicle(ctx context.Context, db *firestore.Client, plate string) (string, error) {
	id := ""
	ref := db.Collection("crm_vehicle_aliases").Doc(plate)
	e := db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		d, e := tx.Get(ref)
		if e == nil {
			var a VehicleAlias
			if e = d.DataTo(&a); e != nil {
				return e
			}
			id = a.VehicleID
			return nil
		}
		if status.Code(e) != codes.NotFound {
			return e
		}
		docs, e := tx.Documents(db.Collection("crm_jobs").Where("Plate", "==", plate).Limit(1)).GetAll()
		if e != nil {
			return e
		}
		id = randomID()
		if len(docs) > 0 {
			var j Job
			if e = docs[0].DataTo(&j); e != nil {
				return e
			}
			id = vehicleID(j)
			if id == "" {
				return errors.New("Не найден ID автомобиля")
			}
		}
		return tx.Create(ref, VehicleAlias{id, plate})
	})
	return id, e
}
func (s *Server) changePlate(w http.ResponseWriter, r *http.Request, u User, j Job) {
	if !u.Manager() {
		fail(w, 403, "Только владелец")
		return
	}
	if r.Method != "POST" {
		fail(w, 405, "Method not allowed")
		return
	}
	var x struct {
		Plate    string `json:"plate"`
		Revision int64  `json:"revision"`
	}
	if e := decode(w, r, &x); e != nil {
		fail(w, 400, e.Error())
		return
	}
	plate, e := Plate(x.Plate)
	if e != nil {
		fail(w, 400, e.Error())
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
		if live.Revision != x.Revision {
			return ErrConflict
		}
		id := vehicleID(live)
		if id == "" {
			return errors.New("Пустой ID")
		}
		aliases := []string{live.Plate}
		if plate != live.Plate {
			aliases = append(aliases, plate)
		}
		missing := []string{}
		for _, p := range aliases {
			d, e := tx.Get(s.Collection("vehicle_aliases").Doc(p))
			if e == nil {
				var a VehicleAlias
				if d.DataTo(&a) != nil || a.VehicleID != id {
					return errors.New("Номер связан с другим автомобилем")
				}
			} else if status.Code(e) == codes.NotFound {
				missing = append(missing, p)
			} else {
				return e
			}
		}
		docs, e := tx.Documents(s.Collection("jobs").Where("Plate", "==", plate)).GetAll()
		if e != nil {
			return e
		}
		for _, d := range docs {
			var old Job
			if d.DataTo(&old) != nil || vehicleID(old) != id {
				return errors.New("Для номера существует другая история")
			}
		}
		for _, p := range missing {
			if e = tx.Create(s.Collection("vehicle_aliases").Doc(p), VehicleAlias{id, p}); e != nil {
				return e
			}
		}
		before := live.Plate
		live.Plate = plate
		live.Vehicle = id
		live.VehicleID = id
		live.Revision++
		live.Updated = time.Now().UTC()
		if e = tx.Set(ref, live); e != nil {
			return e
		}
		return tx.Create(ref.Collection("events").NewDoc(), Event{Type: "plate_changed", Actor: u.ID, Text: before + " → " + plate, At: live.Updated, Revision: live.Revision, Snapshot: &live})
	})
	if e != nil {
		fail(w, 409, e.Error())
		return
	}
	write(w, 200, map[string]bool{"ok": true})
}
