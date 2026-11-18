package crm

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type autoHelpLead struct {
	CreatedAt time.Time `firestore:"created_at"`
	Status    string    `firestore:"client_status"`
	Lead      struct {
		PricingSnapshot []Tariff `firestore:"pricing_snapshot"`
		VisitPriceEUR   int64    `firestore:"visit_price_eur"`
		Car             string   `firestore:"car"`
		Name            string   `firestore:"name"`
		Phone           string   `firestore:"phone"`
		Email           string   `firestore:"email"`
		Address         string   `firestore:"address"`
		Message         string   `firestore:"message"`
		Language        string   `firestore:"language"`
		Photos          []string `firestore:"photo_urls"`
	} `firestore:"lead"`
}

// SyncAutoHelp imports saved AutoHelp submissions into the existing CRM.
func SyncAutoHelp(ctx context.Context) error {
	source, err := firestore.NewClient(ctx, "seventh-chassis-247718")
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer source.Close()
	target, err := firestore.NewClientWithDatabase(ctx, "liikkuva-korjaamo", "liikkuvakorjaamo")
	if err != nil {
		return fmt.Errorf("open CRM: %w", err)
	}
	defer target.Close()
	media, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer media.Close()

	// 17 October 2026, midnight in Helsinki.
	first := time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC)
	end := time.Now().UTC().Add(-time.Minute)
	state := target.Collection("crm_integrations").Doc("autohelp")
	start := first
	var checkpoint struct{ Until time.Time }
	snapshot, err := state.Get(ctx)
	switch {
	case err == nil:
		if err := snapshot.DataTo(&checkpoint); err != nil {
			return fmt.Errorf("read checkpoint: %w", err)
		}
		if candidate := checkpoint.Until.Add(-10 * time.Minute); candidate.After(start) {
			start = candidate
		}
	case status.Code(err) != codes.NotFound:
		return fmt.Errorf("get checkpoint: %w", err)
	}

	docs := source.Collection("leads").Where("created_at", ">=", start).
		Where("created_at", "<", end).OrderBy("created_at", firestore.Asc).Documents(ctx)
	defer docs.Stop()
	var scanned, created, existing, withoutPlate, unavailablePhotos int
	for {
		doc, err := docs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("read source leads: %w", err)
		}
		scanned++
		var lead autoHelpLead
		if err := doc.DataTo(&lead); err != nil {
			return fmt.Errorf("decode lead %s: %w", doc.Ref.ID, err)
		}
		id := digest("autohelp:" + doc.Ref.ID)
		ref := target.Collection("crm_jobs").Doc(id)
		matches, err := target.GetAll(ctx, []*firestore.DocumentRef{
			ref, target.Collection("crm_jobs").Doc(digest("intake-api:" + doc.Ref.ID)),
		})
		if err != nil {
			return fmt.Errorf("check lead %s: %w", doc.Ref.ID, err)
		}
		if matches[0].Exists() || matches[1].Exists() {
			existing++
			continue
		}

		j := Job{
			ID: id, Brand: "autohelp", Status: "new", Revision: 1,
			Name: lead.Lead.Name, Phone: lead.Lead.Phone, Email: lead.Lead.Email,
			Address: lead.Lead.Address, Description: lead.Lead.Message,
			Area: InferArea(lead.Lead.Address), Language: lead.Lead.Language,
			Created: lead.CreatedAt, Updated: lead.CreatedAt,
			Photos: []Photo{}, Proposal: Proposal{Payment: "cash"},
		}
		if j.Language != "en" && j.Language != "ru" {
			j.Language = "fi"
		}
		if strings.TrimSpace(j.Description) == "" {
			j.Description = "Клиент не описал проблему"
		}
		if plate, err := Plate(lead.Lead.Car); err == nil {
			j.Plate = plate
			vehicle, err := resolveVehicle(ctx, target, plate)
			if err != nil {
				return fmt.Errorf("resolve vehicle for lead %s: %w", doc.Ref.ID, err)
			}
			j.Vehicle, j.VehicleID = vehicle, vehicle
		} else {
			withoutPlate++
			if value := strings.TrimSpace(lead.Lead.Car); value != "" {
				j.Description += "\nАвтомобиль (поле AutoHelp): " + value
			}
		}
		applyAutohelpLeadPrices(&j, lead.Lead.PricingSnapshot, lead.Lead.VisitPriceEUR)
		for _, raw := range lead.Lead.Photos {
			photo, err := copyAutoHelpPhoto(ctx, media, id, raw, lead.CreatedAt)
			if err != nil {
				return fmt.Errorf("copy attachment for lead %s: %w", doc.Ref.ID, err)
			}
			if photo.ID == "" {
				unavailablePhotos++
				continue
			}
			j.Photos = append(j.Photos, photo)
		}

		note := "Заявка AutoHelp, исходный ID " + doc.Ref.ID
		if lead.Status != "" {
			note += ", статус в AutoHelp: " + lead.Status
		}
		batch := target.Batch()
		batch.Create(ref, j)
		batch.Create(ref.Collection("events").Doc("autohelp-created"), Event{
			Actor: "site", Type: "created", Text: note,
			At: j.Created, Revision: 1, Snapshot: &j,
		})
		if _, err := batch.Commit(ctx); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				existing++
				continue
			}
			return fmt.Errorf("save lead %s: %w", doc.Ref.ID, err)
		}
		created++
	}
	if err := target.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		current, err := tx.Get(state)
		if err == nil {
			var saved struct{ Until time.Time }
			if err := current.DataTo(&saved); err != nil {
				return err
			}
			if !end.After(saved.Until) {
				return nil
			}
		} else if status.Code(err) != codes.NotFound {
			return err
		}
		return tx.Set(state, map[string]interface{}{"Until": end}, firestore.MergeAll)
	}); err != nil {
		return fmt.Errorf("save checkpoint: %w", err)
	}
	log.Printf("AutoHelp sync: read=%d created=%d existing=%d without_plate=%d unavailable_photos=%d",
		scanned, created, existing, withoutPlate, unavailablePhotos)
	return nil
}

func copyAutoHelpPhoto(ctx context.Context, client *storage.Client, jobID, raw string, created time.Time) (Photo, error) {
	source := os.Getenv("AUTOHELP_PHOTO_BUCKET")
	destination := os.Getenv("CRM_PHOTO_BUCKET")
	u, err := url.Parse(raw)
	if err != nil || source == "" || destination == "" || u.Scheme != "gs" ||
		u.Host != source || u.User != nil || u.Path == "" || u.RawQuery != "" || u.Fragment != "" {
		return Photo{}, nil
	}
	id := digest(raw)
	object := "crm/" + jobID + "/" + id
	src := client.Bucket(source).Object(strings.TrimPrefix(u.Path, "/"))
	dst := client.Bucket(destination).Object(object).If(storage.Conditions{DoesNotExist: true})
	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		var apiErr *googleapi.Error
		if errors.Is(err, storage.ErrObjectNotExist) || (errors.As(err, &apiErr) && apiErr.Code == 404) {
			return Photo{}, nil
		}
		if !errors.As(err, &apiErr) || apiErr.Code != 412 {
			return Photo{}, errors.New("storage copy failed; check bucket permissions and configuration")
		}
	}
	return Photo{ID: id, Object: object, Stage: "before", Caption: "AutoHelp", Author: "site", Created: created}, nil
}
