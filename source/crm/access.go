package crm

// Drafts are private to the owner, including action responses.
func jobResponse(j Job, u User) map[string]any {
	out := map[string]any{"job": jobView(j, u)}
	if u.Manager() {
		out["draft"] = Draft(j)
		drafts := map[string]string{}
		for _, l := range []string{"fi", "en", "ru"} {
			copy := j
			copy.Language = l
			drafts[l] = Draft(copy)
		}
		out["drafts"] = drafts
	}
	return out
}
func eventView(e Event, u User) (any, bool) {
	if u.Manager() {
		return e, true
	}
	text := ""
	switch e.Type {
	case "technical_note":
		text = e.Text
	case "internal_note":
		if e.Actor != u.ID {
			return nil, false
		}
		text = e.Text
	case "created", "assign", "propose", "client_confirm", "confirm", "reschedule", "start", "report", "close", "reopen", "cancel", "vehicle", "photo":
	default:
		return nil, false
	}
	return map[string]any{"id": e.ID, "type": e.Type, "at": e.At, "revision": e.Revision, "text": text}, true
}
func technicalVisit(j Job) map[string]any {
	return map[string]any{"id": j.ID, "created": j.Created, "status": j.Status, "description": j.Description, "model": j.Model, "mileage": j.Mileage, "photos": j.Photos, "report": map[string]any{"diagnosis": j.Report.Diagnosis, "parts": j.Report.Parts, "work": j.Report.Work, "remaining": j.Report.Remaining, "result": j.Report.Result}}
}

func jobView(j Job, u User) any {
	if u.Manager() {
		return j
	}
	return map[string]any{"id": j.ID, "vehicle_id": vehicleID(j), "plate": j.Plate, "brand": j.Brand, "area": j.Area, "name": j.Name, "phone": j.Phone, "address": j.Address, "description": j.Description, "model": j.Model, "vin": j.VIN, "mileage": j.Mileage, "mechanic": j.Mechanic, "status": j.Status, "proposal": j.Proposal, "report": j.Report, "photos": j.Photos, "intake_photo_count": len(j.IntakePhotos), "revision": j.Revision, "created": j.Created, "updated": j.Updated}
}
