package repository

import (
	"context"
	"errors"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"gorm.io/gorm"
)

type ItemRepository interface {
	Create(ctx context.Context, item *model.Item) error
	FindByID(ctx context.Context, id uint64) (*model.Item, error)
	List(ctx context.Context, limit, offset int) ([]model.Item, int64, error)
	FindByImageURL(ctx context.Context, imageURL string) (*model.Item, error)
}

type itemRepository struct {
	db *gorm.DB
}

func NewItemRepository(db *gorm.DB) ItemRepository {
	return &itemRepository{db: db}
}

func (r *itemRepository) Create(ctx context.Context, item *model.Item) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *itemRepository) FindByID(ctx context.Context, id uint64) (*model.Item, error) {
	var item model.Item
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *itemRepository) List(ctx context.Context, limit, offset int) ([]model.Item, int64, error) {
	var (
		items []model.Item
		total int64
	)
	if err := r.db.WithContext(ctx).Model(&model.Item{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *itemRepository) FindByImageURL(ctx context.Context, imageURL string) (*model.Item, error) {
	var item model.Item
	if err := r.db.WithContext(ctx).
		Where("image_url = ?", imageURL).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}
