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
	List(ctx context.Context, limit, offset int, categorySlug string) ([]model.Item, int64, error)
	FindByImageURL(ctx context.Context, imageURL string) (*model.Item, error)
	ListBySeller(ctx context.Context, sellerUID string) ([]model.Item, error)
	SetDB(db *gorm.DB)
}

type itemRepository struct {
	db *gorm.DB
}

var ErrDBNotReady = errors.New("database not initialized")

func NewItemRepository(db *gorm.DB) ItemRepository {
	return &itemRepository{db: db}
}

func (r *itemRepository) Create(ctx context.Context, item *model.Item) error {
	if r.db == nil {
		return ErrDBNotReady
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *itemRepository) FindByID(ctx context.Context, id uint64) (*model.Item, error) {
	if r.db == nil {
		return nil, ErrDBNotReady
	}
	var item model.Item
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *itemRepository) List(ctx context.Context, limit, offset int, categorySlug string) ([]model.Item, int64, error) {
	if r.db == nil {
		return nil, 0, ErrDBNotReady
	}
	var (
		items []model.Item
		total int64
	)
	q := r.db.WithContext(ctx).Model(&model.Item{})
	if categorySlug != "" {
		q = q.Where("category_slug = ?", categorySlug)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *itemRepository) FindByImageURL(ctx context.Context, imageURL string) (*model.Item, error) {
	if r.db == nil {
		return nil, ErrDBNotReady
	}
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

func (r *itemRepository) SetDB(db *gorm.DB) {
	r.db = db
}

func (r *itemRepository) ListBySeller(ctx context.Context, sellerUID string) ([]model.Item, error) {
	if r.db == nil {
		return nil, ErrDBNotReady
	}
	var items []model.Item
	if err := r.db.WithContext(ctx).
		Where("seller_uid = ?", sellerUID).
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
