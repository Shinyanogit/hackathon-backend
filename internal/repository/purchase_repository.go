package repository

import (
	"context"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"gorm.io/gorm"
)

type PurchaseRepository interface {
	Create(ctx context.Context, p *model.Purchase) error
	FindByItem(ctx context.Context, itemID uint64) (*model.Purchase, error)
	FindByID(ctx context.Context, id uint64) (*model.Purchase, error)
	Update(ctx context.Context, p *model.Purchase) error
	ListByBuyer(ctx context.Context, buyerUID string) ([]model.Purchase, error)
	SetDB(db *gorm.DB)
}

type purchaseRepository struct {
	db *gorm.DB
}

func NewPurchaseRepository(db *gorm.DB) PurchaseRepository {
	return &purchaseRepository{db: db}
}

func (r *purchaseRepository) Create(ctx context.Context, p *model.Purchase) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *purchaseRepository) FindByItem(ctx context.Context, itemID uint64) (*model.Purchase, error) {
	var p model.Purchase
	if err := r.db.WithContext(ctx).
		Where("item_id = ?", itemID).
		Order("id DESC").
		First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *purchaseRepository) FindByID(ctx context.Context, id uint64) (*model.Purchase, error) {
	var p model.Purchase
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *purchaseRepository) Update(ctx context.Context, p *model.Purchase) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *purchaseRepository) ListByBuyer(ctx context.Context, buyerUID string) ([]model.Purchase, error) {
	var list []model.Purchase
	if err := r.db.WithContext(ctx).
		Where("buyer_uid = ?", buyerUID).
		Order("id DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *purchaseRepository) SetDB(db *gorm.DB) {
	r.db = db
}
