package crm

import (
	"cloud.google.com/go/firestore"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// AddLeadToBatch makes saving the existing site lead and its CRM case atomic.
// Call before Commit. Resolves vehicle aliases and current tariffs before preparing the batch.
func AddLeadToBatch(db *firestore.Client, b *firestore.WriteBatch, leadID, plate, name, phone, email, address, description, brand, language string, photos []string) error {
	return AddLeadToBatchAt(db, b, leadID, plate, name, phone, email, address, description, brand, language, photos, time.Time{})
}
func AddLeadToBatchAt(db *firestore.Client, b *firestore.WriteBatch, leadID, plate, name, phone, email, address, description, brand, language string, photos []string, created time.Time) error {
	if description == "" {
		description = "Клиент не описал проблему — уточнить перед выездом"
	}
	hash := sha256.Sum256([]byte(brand + ":" + leadID))
	id := hex.EncodeToString(hash[:])
	j, e := NewJob(id, plate, name, phone, address, description, brand, language)
	if e != nil {
		return e
	}
	if !created.IsZero() {
		j.Created = created
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	book, e := loadPriceBook(ctx, db)
	if e != nil {
		return e
	}
	applyPrices(&j, book)
	stable, e := resolveVehicle(ctx, db, j.Plate)
	if e != nil {
		return e
	}
	j.Vehicle = stable
	j.VehicleID = stable
	j.Email = email
	j.IntakePhotos = photos
	ref := db.Collection("crm_jobs").Doc(id)
	b.Create(ref, j)
	b.Create(ref.Collection("events").NewDoc(), Event{Actor: "site", Type: "created", Text: "Заявка с сайта; ID " + leadID, At: j.Created, Revision: 1, Snapshot: &j})
	return nil
}
