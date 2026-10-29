package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const siteSettingsDocID = "landing"

var errSiteSettingsConflict = errors.New("Настройки уже изменены в другом окне. Ваши правки показаны ниже и пока не сохранены. Скопируйте нужный текст, обновите страницу и повторите сохранение.")

type siteSettingsStore interface {
	Load(context.Context) (SiteSettings, error)
	Update(context.Context, func(SiteSettings) (SiteSettings, error)) error
}

func loadSiteSettings(ctx context.Context, cfg Config) (SiteSettings, error) {
	if cfg.SettingsStore != nil {
		return cfg.SettingsStore.Load(ctx)
	}
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
		if status.Code(err) == codes.NotFound {
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
	// Keep the expected revision immutable across transaction retries.
	expectedRevision := settings.Revision
	replace := func(current SiteSettings) (SiteSettings, error) {
		if current.Revision != expectedRevision {
			return SiteSettings{}, errSiteSettingsConflict
		}
		next := settings
		next.Revision = expectedRevision + 1
		return next, nil
	}
	if cfg.SettingsStore != nil {
		return cfg.SettingsStore.Update(ctx, replace)
	}
	if strings.TrimSpace(cfg.FirestoreProjectID) == "" {
		return fmt.Errorf("FIRESTORE_PROJECT_ID is empty")
	}

	client, err := firestore.NewClientWithDatabase(ctx, cfg.FirestoreProjectID, "liikkuvakorjaamo")
	if err != nil {
		return err
	}
	defer client.Close()

	ref := client.Collection(cfg.SiteSettingsCollection).Doc(siteSettingsDocID)
	return client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(ref)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		var current SiteSettings
		if err == nil {
			if err := doc.DataTo(&current); err != nil {
				return err
			}
		}
		next, err := replace(current)
		if err != nil {
			return err
		}
		return tx.Set(ref, next)
	})
}
