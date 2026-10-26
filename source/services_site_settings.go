package main

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
)

const siteSettingsDocID = "landing"

func loadSiteSettings(ctx context.Context, cfg Config) (SiteSettings, error) {
	if strings.TrimSpace(cfg.FirestoreProjectID) == "" {
		return SiteSettings{}, nil
	}

	client, err := firestore.NewClientWithDatabase(ctx, cfg.FirestoreProjectID, "liikkuvakorjaamo")
	if err != nil {
		return SiteSettings{}, err
	}
	defer client.Close()

	doc, err := client.Collection(cfg.SiteSettingsCollection).Doc(siteSettingsDocID).Get(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "notfound") {
			return SiteSettings{}, nil
		}
		return SiteSettings{}, err
	}

	var settings SiteSettings
	if err := doc.DataTo(&settings); err != nil {
		return SiteSettings{}, err
	}

	return settings, nil
}

func saveSiteSettings(ctx context.Context, cfg Config, settings SiteSettings) error {
	if strings.TrimSpace(cfg.FirestoreProjectID) == "" {
		return fmt.Errorf("FIRESTORE_PROJECT_ID is empty")
	}

	client, err := firestore.NewClientWithDatabase(ctx, cfg.FirestoreProjectID, "liikkuvakorjaamo")
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.Collection(cfg.SiteSettingsCollection).Doc(siteSettingsDocID).Set(ctx, settings)
	return err
}
