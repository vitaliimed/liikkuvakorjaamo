package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	reviewStatusPending  = "pending"
	reviewStatusApproved = "approved"
	reviewStatusRejected = "rejected"
)

var errReviewAlreadySubmitted = errors.New("review already submitted")

func reviewsCollection(cfg Config) string {
	value := strings.TrimSpace(cfg.FirestoreReviewsCollection)
	if value == "" {
		return "reviews"
	}
	return value
}

func createReviewRecord(ctx context.Context, cfg Config, review ReviewRecord) error {
	client, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	now := time.Now().UTC()
	review.ID = ""
	review.Status = reviewStatusPending
	review.CreatedAt = now
	review.UpdatedAt = now
	review.ModeratedAt = time.Time{}

	_, err = client.Collection(reviewsCollection(cfg)).Doc(review.LeadID).Create(ctx, review)
	if status.Code(err) == codes.AlreadyExists {
		return errReviewAlreadySubmitted
	}
	if err != nil {
		return fmt.Errorf("save review to firestore: %w", err)
	}

	return nil
}

func getReviewRecord(ctx context.Context, cfg Config, reviewID string) (ReviewRecord, error) {
	client, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		return ReviewRecord{}, err
	}
	defer client.Close()

	doc, err := client.Collection(reviewsCollection(cfg)).Doc(reviewID).Get(ctx)
	if err != nil {
		return ReviewRecord{}, err
	}

	var review ReviewRecord
	if err := doc.DataTo(&review); err != nil {
		return ReviewRecord{}, fmt.Errorf("decode review from firestore: %w", err)
	}
	review.ID = doc.Ref.ID
	return review, nil
}

func reviewExists(ctx context.Context, cfg Config, reviewID string) (bool, error) {
	_, err := getReviewRecord(ctx, cfg, reviewID)
	if status.Code(err) == codes.NotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func listReviewRecords(ctx context.Context, cfg Config, limit int) ([]ReviewRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	client, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	docs, err := client.Collection(reviewsCollection(cfg)).
		OrderBy("created_at", firestore.Desc).
		Limit(limit).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, fmt.Errorf("list reviews from firestore: %w", err)
	}

	reviews := make([]ReviewRecord, 0, len(docs))
	for _, doc := range docs {
		var review ReviewRecord
		if err := doc.DataTo(&review); err != nil {
			return nil, fmt.Errorf("decode review from firestore: %w", err)
		}
		review.ID = doc.Ref.ID
		reviews = append(reviews, review)
	}

	return reviews, nil
}

func listApprovedReviewRecords(ctx context.Context, cfg Config, lang string, limit int) ([]ReviewRecord, error) {
	if limit <= 0 {
		limit = 12
	}
	lang = sanitizeLanguage(lang, "fi")

	records, err := listReviewRecords(ctx, cfg, 100)
	if err != nil {
		return nil, err
	}

	reviews := make([]ReviewRecord, 0, limit)
	for _, review := range records {
		if review.Status != reviewStatusApproved || review.Language != lang || !review.PublishConsent {
			continue
		}
		reviews = append(reviews, review)
		if len(reviews) >= limit {
			break
		}
	}
	return reviews, nil
}

func filterReviewRecords(records []ReviewRecord, statusFilter string) []ReviewRecord {
	statusFilter = normalizeReviewStatusFilter(statusFilter)
	if statusFilter == "all" {
		return records
	}

	filtered := make([]ReviewRecord, 0, len(records))
	for _, record := range records {
		if strings.EqualFold(strings.TrimSpace(record.Status), statusFilter) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func approvedReviewCards(records []ReviewRecord, lang string, limit int) []TestimonialCard {
	items := make([]TestimonialCard, 0, len(records))
	for _, record := range records {
		if record.Status != reviewStatusApproved || record.Language != lang || !record.PublishConsent {
			continue
		}
		items = append(items, TestimonialCard{
			Quote:       record.Message,
			Author:      record.Name,
			Rating:      normalizeTestimonialRatingValue(record.Rating),
			RatingStars: testimonialStars(record.Rating),
		})
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items
}

func moderateReviewRecord(ctx context.Context, cfg Config, reviewID, action string) error {
	newStatus := ""
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "approve":
		newStatus = reviewStatusApproved
	case "reject":
		newStatus = reviewStatusRejected
	default:
		return fmt.Errorf("invalid moderation action")
	}

	client, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.Collection(reviewsCollection(cfg)).Doc(reviewID).Update(ctx, []firestore.Update{
		{Path: "status", Value: newStatus},
		{Path: "updated_at", Value: time.Now().UTC()},
		{Path: "moderated_at", Value: time.Now().UTC()},
	})
	if err != nil {
		return fmt.Errorf("moderate review: %w", err)
	}
	return nil
}

func normalizeReviewStatusFilter(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case reviewStatusPending, reviewStatusApproved, reviewStatusRejected:
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "all"
	}
}
