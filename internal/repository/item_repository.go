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
	List(ctx context.Context, limit, offset int, categorySlug, query, sellerUID string) ([]model.Item, int64, error)
	FindByImageURL(ctx context.Context, imageURL string) (*model.Item, error)
	ListBySeller(ctx context.Context, sellerUID string) ([]model.Item, error)
	UpdateBySeller(ctx context.Context, id uint64, sellerUID string, fields map[string]interface{}) error
	SetDB(db *gorm.DB)
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

func (r *itemRepository) List(ctx context.Context, limit, offset int, categorySlug, query, sellerUID string) ([]model.Item, int64, error) {
	var (
		items []model.Item
		total int64
	)
	q := r.db.WithContext(ctx).Model(&model.Item{})
	if categorySlug != "" {
		q = q.Where("category_slug = ?", categorySlug)
	}
	if sellerUID != "" {
		q = q.Where("seller_uid = ?", sellerUID)
	}
	if query != "" {
		like := "%" + query + "%"
		q = q.Where("title LIKE ? OR description LIKE ?", like, like)
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
	var items []model.Item
	if err := r.db.WithContext(ctx).
		Where("seller_uid = ?", sellerUID).
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *itemRepository) UpdateBySeller(ctx context.Context, id uint64, sellerUID string, fields map[string]interface{}) error {
	res := r.db.WithContext(ctx).
		Model(&model.Item{}).
		Where("id = ? AND seller_uid = ?", id, sellerUID).
		Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
