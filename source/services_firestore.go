package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
)

func asyncLeadPipelineEnabled(cfg Config) bool {
	return strings.TrimSpace(cfg.FirestoreProjectID) != "" &&
		strings.TrimSpace(cfg.CloudTasksProjectID) != "" &&
		strings.TrimSpace(cfg.CloudTasksLocation) != "" &&
		strings.TrimSpace(cfg.CloudTasksQueue) != ""
}

func newFirestoreClient(ctx context.Context, cfg Config) (*firestore.Client, error) {
	if strings.TrimSpace(cfg.FirestoreProjectID) == "" {
		return nil, fmt.Errorf("FIRESTORE_PROJECT_ID is empty")
	}
	return firestore.NewClientWithDatabase(ctx, cfg.FirestoreProjectID, "liikkuvakorjaamo")
}

func saveLeadRecord(ctx context.Context, cfg Config, lead Lead) (string, error) {
	client, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		return "", err
	}
	defer client.Close()

	now := time.Now().UTC()
	record := LeadRecord{
		Lead:                    lead,
		ProcessingStatus:        "queued",
		TelegramStatus:          "pending",
		SheetsStatus:            "pending",
		ClientStatus:            "new",
		ReadyEmailStatus:        "not_sent",
		ConfirmationEmailStatus: confirmationEmailInitialStatus(lead),
		ReviewInviteEmailStatus: "not_sent",
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	ref, _, err := client.Collection(cfg.FirestoreLeadsCollection).Add(ctx, record)
	if err != nil {
		return "", fmt.Errorf("save lead to firestore: %w", err)
	}

	return ref.ID, nil
}

func confirmationEmailInitialStatus(lead Lead) string {
	if strings.TrimSpace(lead.Email) == "" {
		return "not_applicable"
	}
	return "pending"
}

func getLeadRecord(ctx context.Context, cfg Config, leadID string) (LeadRecord, error) {
	client, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		return LeadRecord{}, err
	}
	defer client.Close()

	doc, err := client.Collection(cfg.FirestoreLeadsCollection).Doc(leadID).Get(ctx)
	if err != nil {
		return LeadRecord{}, fmt.Errorf("get lead from firestore: %w", err)
	}

	var record LeadRecord
	if err := doc.DataTo(&record); err != nil {
		return LeadRecord{}, fmt.Errorf("decode lead from firestore: %w", err)
	}
	record.ID = doc.Ref.ID
	return record, nil
}

func listLeadRecords(ctx context.Context, cfg Config, limit int, statusFilter string) ([]LeadRecord, error) {
	if limit <= 0 {
		limit = 50
	}

	statusFilter = strings.ToLower(strings.TrimSpace(statusFilter))

	fetchLimit := limit
	if statusFilter != "" && statusFilter != "all" {
		fetchLimit = limit * 3
	}

	client, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	docs, err := client.
		Collection(cfg.FirestoreLeadsCollection).
		OrderBy("created_at", firestore.Desc).
		Limit(fetchLimit).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, fmt.Errorf("list leads from firestore: %w", err)
	}

	records := make([]LeadRecord, 0, len(docs))
	for _, doc := range docs {
		var record LeadRecord
		if err := doc.DataTo(&record); err != nil {
			return nil, fmt.Errorf("decode lead from firestore: %w", err)
		}
		record.ID = doc.Ref.ID

		if statusFilter != "" && statusFilter != "all" {
			if strings.ToLower(strings.TrimSpace(record.ProcessingStatus)) != statusFilter {
				continue
			}
		}

		records = append(records, record)
		if len(records) >= limit {
			break
		}
	}

	return records, nil
}

func updateLeadRecordFields(ctx context.Context, cfg Config, leadID string, updates []firestore.Update) error {
	client, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	updates = append(updates, firestore.Update{Path: "updated_at", Value: time.Now().UTC()})
	_, err = client.Collection(cfg.FirestoreLeadsCollection).Doc(leadID).Update(ctx, updates)
	if err != nil {
		return fmt.Errorf("update lead in firestore: %w", err)
	}

	return nil
}

func markLeadQueued(ctx context.Context, cfg Config, leadID string) error {
	return updateLeadRecordFields(ctx, cfg, leadID, []firestore.Update{
		{Path: "processing_status", Value: "queued"},
		{Path: "task_enqueued_at", Value: time.Now().UTC()},
		{Path: "last_error", Value: ""},
	})
}

func markLeadProcessing(ctx context.Context, cfg Config, leadID string) error {
	return updateLeadRecordFields(ctx, cfg, leadID, []firestore.Update{
		{Path: "processing_status", Value: "processing"},
		{Path: "last_error", Value: ""},
	})
}

func markTelegramStatus(ctx context.Context, cfg Config, leadID, status, lastError string) error {
	updates := []firestore.Update{{Path: "telegram_status", Value: status}}
	if lastError != "" {
		updates = append(updates, firestore.Update{Path: "last_error", Value: lastError})
	}
	return updateLeadRecordFields(ctx, cfg, leadID, updates)
}

func markSheetsStatus(ctx context.Context, cfg Config, leadID, status, lastError string) error {
	updates := []firestore.Update{{Path: "sheets_status", Value: status}}
	if lastError != "" {
		updates = append(updates, firestore.Update{Path: "last_error", Value: lastError})
	}
	return updateLeadRecordFields(ctx, cfg, leadID, updates)
}

func markLeadDone(ctx context.Context, cfg Config, leadID string) error {
	return updateLeadRecordFields(ctx, cfg, leadID, []firestore.Update{
		{Path: "processing_status", Value: "done"},
		{Path: "processed_at", Value: time.Now().UTC()},
		{Path: "last_error", Value: ""},
	})
}

func markLeadFailed(ctx context.Context, cfg Config, leadID, lastError string) error {
	return updateLeadRecordFields(ctx, cfg, leadID, []firestore.Update{
		{Path: "processing_status", Value: "failed"},
		{Path: "last_error", Value: lastError},
	})
}

func markClientStatus(ctx context.Context, cfg Config, leadID, status string) error {
	return updateLeadRecordFields(ctx, cfg, leadID, []firestore.Update{
		{Path: "client_status", Value: strings.TrimSpace(status)},
	})
}

func markReadyEmailSent(ctx context.Context, cfg Config, leadID string) error {
	return updateLeadRecordFields(ctx, cfg, leadID, []firestore.Update{
		{Path: "ready_email_status", Value: "sent"},
		{Path: "ready_email_error", Value: ""},
		{Path: "ready_email_sent_at", Value: time.Now().UTC()},
		{Path: "client_status", Value: "ready_for_pickup"},
	})
}

func markReadyEmailFailed(ctx context.Context, cfg Config, leadID, lastError string) error {
	return updateLeadRecordFields(ctx, cfg, leadID, []firestore.Update{
		{Path: "ready_email_status", Value: "failed"},
		{Path: "ready_email_error", Value: strings.TrimSpace(lastError)},
	})
}

func markConfirmationEmailSent(ctx context.Context, cfg Config, leadID string) error {
	return updateLeadRecordFields(ctx, cfg, leadID, []firestore.Update{
		{Path: "confirmation_email_status", Value: "sent"},
		{Path: "confirmation_email_error", Value: ""},
		{Path: "confirmation_email_sent_at", Value: time.Now().UTC()},
	})
}

func markConfirmationEmailFailed(ctx context.Context, cfg Config, leadID, lastError string) error {
	return updateLeadRecordFields(ctx, cfg, leadID, []firestore.Update{
		{Path: "confirmation_email_status", Value: "failed"},
		{Path: "confirmation_email_error", Value: strings.TrimSpace(lastError)},
	})
}

func markReviewInviteEmailSent(ctx context.Context, cfg Config, leadID string) error {
	return updateLeadRecordFields(ctx, cfg, leadID, []firestore.Update{
		{Path: "review_invite_email_status", Value: "sent"},
		{Path: "review_invite_email_error", Value: ""},
		{Path: "review_invite_email_sent_at", Value: time.Now().UTC()},
	})
}

func markReviewInviteEmailFailed(ctx context.Context, cfg Config, leadID, lastError string) error {
	return updateLeadRecordFields(ctx, cfg, leadID, []firestore.Update{
		{Path: "review_invite_email_status", Value: "failed"},
		{Path: "review_invite_email_error", Value: strings.TrimSpace(lastError)},
	})
}
