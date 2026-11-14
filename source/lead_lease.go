package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"time"
)

// Serialize duplicate Cloud Tasks deliveries. Expiry permits recovery after a
// terminated worker. The caller's processing timeout must stay below the lease.
func acquireLeadLease(ctx context.Context, cfg Config, id string) (func(), error) {
	client, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	ref := client.Collection(cfg.FirestoreLeadsCollection).Doc(id)
	token := liikkuvaSubmissionID()
	err = client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, e := tx.Get(ref)
		if e != nil {
			return e
		}
		expiry, _ := doc.Data()["processing_lease_until"].(time.Time)
		if expiry.After(time.Now()) {
			return fmt.Errorf("lead processing already in progress")
		}
		return tx.Update(ref, []firestore.Update{{Path: "processing_lease_token", Value: token}, {Path: "processing_lease_until", Value: time.Now().Add(time.Hour)}})
	})
	if err != nil {
		client.Close()
		return nil, err
	}
	return func() {
		defer client.Close()
		releaseCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = client.RunTransaction(releaseCtx, func(ctx context.Context, tx *firestore.Transaction) error {
			doc, e := tx.Get(ref)
			if e != nil {
				return e
			}
			if doc.Data()["processing_lease_token"] != token {
				return nil
			}
			return tx.Update(ref, []firestore.Update{{Path: "processing_lease_token", Value: firestore.Delete}, {Path: "processing_lease_until", Value: firestore.Delete}})
		})
	}, nil
}
