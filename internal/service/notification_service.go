package service

import (
	"context"
	"time"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/repository"
)

type NotificationService interface {
	Notify(ctx context.Context, userUID, typ, title, body string, itemID, convID, purchaseID *uint64)
	List(ctx context.Context, userUID string, unreadOnly bool, limit int) ([]model.Notification, int64, error)
	MarkAllRead(ctx context.Context, userUID string) error
	MarkByConversation(ctx context.Context, userUID string, convID uint64) error
	MarkByPurchase(ctx context.Context, userUID string, purchaseID uint64) error
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
	_ = s.repo.Create(ctx, n)
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

func (s *notificationService) MarkByConversation(ctx context.Context, userUID string, convID uint64) error {
	if userUID == "" || convID == 0 {
		return nil
	}
	return s.repo.MarkByConversation(ctx, userUID, convID)
}

func (s *notificationService) MarkByPurchase(ctx context.Context, userUID string, purchaseID uint64) error {
	if userUID == "" || purchaseID == 0 {
		return nil
	}
	return s.repo.MarkByPurchase(ctx, userUID, purchaseID)
}

// helper to return pointer
func uint64Ptr(v uint64) *uint64 {
	return &v
}

// WithDeadline wraps context with a short deadline to avoid blocking main flow.
func withShortDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 2*time.Second)
}
