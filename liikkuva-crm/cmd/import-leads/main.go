// Idempotent import of the existing landing leads. Run only after backup.
package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"flag"
	"fmt"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"liikkuvakorjaamo.fi/landing/crm"
	"log"
	"time"
)

func main() {
	project := flag.String("project", "liikkuva-korjaamo", "GCP project")
	database := flag.String("database", "liikkuvakorjaamo", "Firestore database")
	collection := flag.String("collection", "leads", "source collection")
	apply := flag.Bool("apply", false, "write CRM cases (default: validate only)")
	flag.Parse()
	ctx := context.Background()
	db, e := firestore.NewClientWithDatabase(ctx, *project, *database)
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	it := db.Collection(*collection).Documents(ctx)
	defer it.Stop()
	valid, skip, invalid := 0, 0, 0
	for {
		d, e := it.Next()
		if e == iterator.Done {
			break
		}
		if e != nil {
			log.Fatal(e)
		}
		var old struct {
			Lead struct {
				Car      string   `firestore:"car"`
				Name     string   `firestore:"name"`
				Phone    string   `firestore:"phone"`
				Email    string   `firestore:"email"`
				Address  string   `firestore:"address"`
				Message  string   `firestore:"message"`
				Language string   `firestore:"language"`
				Photos   []string `firestore:"photo_urls"`
			} `firestore:"lead"`
			Created time.Time `firestore:"created_at"`
		}
		if e = d.DataTo(&old); e != nil {
			invalid++
			fmt.Printf("INVALID document %s: decode\n", d.Ref.ID)
			continue
		}
		b := db.Batch()
		l := old.Lead
		e = crm.AddLeadToBatchAt(db, b, d.Ref.ID, l.Car, l.Name, l.Phone, l.Email, l.Address, l.Message, "liikkuva", l.Language, l.Photos, old.Created)
		if e != nil {
			invalid++
			fmt.Printf("INVALID document %s: required fields/plate\n", d.Ref.ID)
			continue
		}
		if !*apply {
			valid++
			continue
		}
		if _, e = b.Commit(ctx); status.Code(e) == codes.AlreadyExists {
			skip++
		} else if e != nil {
			log.Fatal(e)
		} else {
			valid++
		}
	}
	fmt.Printf("Valid/imported: %d; already present: %d; invalid: %d; apply=%v\n", valid, skip, invalid, *apply)
}
