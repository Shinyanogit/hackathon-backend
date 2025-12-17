package repository

import (
	"context"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"gorm.io/gorm"
)

type NotificationRepository interface {
	Create(ctx context.Context, n *model.Notification) error
	ListByUser(ctx context.Context, userUID string, unreadOnly bool, limit int) ([]model.Notification, error)
	MarkAllRead(ctx context.Context, userUID string) error
	CountUnread(ctx context.Context, userUID string) (int64, error)
	SetDB(db *gorm.DB)
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, n *model.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *notificationRepository) ListByUser(ctx context.Context, userUID string, unreadOnly bool, limit int) ([]model.Notification, error) {
	var list []model.Notification
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Model(&model.Notification{}).Where("user_uid = ?", userUID)
	if unreadOnly {
		q = q.Where("read_at IS NULL")
	}
	if err := q.Order("created_at DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, userUID string) error {
	now := r.db.NowFunc()
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_uid = ? AND read_at IS NULL", userUID).
		Update("read_at", now).Error
}

func (r *notificationRepository) CountUnread(ctx context.Context, userUID string) (int64, error) {
	var cnt int64
	if err := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_uid = ? AND read_at IS NULL", userUID).
		Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

func (r *notificationRepository) SetDB(db *gorm.DB) {
	r.db = db
}
