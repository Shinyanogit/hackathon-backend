package service

import (
	"context"
	"time"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/repository"
	"log"
)

type NotificationService interface {
	Notify(ctx context.Context, userUID, typ, title, body string, itemID, convID, purchaseID *uint64)
	List(ctx context.Context, userUID string, unreadOnly bool, limit int) ([]model.Notification, int64, error)
	MarkAllRead(ctx context.Context, userUID string) error
}

type notificationService struct {
	repo repository.NotificationRepository
}

func NewNotificationService(repo repository.NotificationRepository) NotificationService {
	return &notificationService{repo: repo}
}

// Notify is best-effort; it logs errors but does not return them to avoid breaking main flows.
func (s *notificationService) Notify(ctx context.Context, userUID, typ, title, body string, itemID, convID, purchaseID *uint64) {
	if userUID == "" || typ == "" {
		log.Printf("[notify] skip: missing userUID or type (user=%s type=%s)", userUID, typ)
		return
	}
	n := &model.Notification{
		UserUID:        userUID,
		Type:           typ,
		Title:          title,
		Body:           body,
		ItemID:         itemID,
		ConversationID: convID,
		PurchaseID:     purchaseID,
	}
	if err := s.repo.Create(ctx, n); err != nil {
		log.Printf("[notify] create failed user=%s type=%s item=%v conv=%v purchase=%v err=%v", userUID, typ, itemID, convID, purchaseID, err)
	} else {
		log.Printf("[notify] created user=%s type=%s item=%v conv=%v purchase=%v", userUID, typ, itemID, convID, purchaseID)
	}
}

func (s *notificationService) List(ctx context.Context, userUID string, unreadOnly bool, limit int) ([]model.Notification, int64, error) {
	if userUID == "" {
		return nil, 0, nil
	}
	list, err := s.repo.ListByUser(ctx, userUID, unreadOnly, limit)
	if err != nil {
		return nil, 0, err
	}
	cnt, err := s.repo.CountUnread(ctx, userUID)
	if err != nil {
		return list, 0, err
	}
	return list, cnt, nil
}

func (s *notificationService) MarkAllRead(ctx context.Context, userUID string) error {
	if userUID == "" {
		return nil
	}
	return s.repo.MarkAllRead(ctx, userUID)
}

// helper to return pointer
func uint64Ptr(v uint64) *uint64 {
	return &v
}

// WithDeadline wraps context with a short deadline to avoid blocking main flow.
func withShortDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 2*time.Second)
}
