package repository

import (
	"context"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRevenueRepository interface {
	Add(ctx context.Context, uid string, cents int64) error
	Deduct(ctx context.Context, uid string, cents int64) error
	Get(ctx context.Context, uid string) (*model.UserRevenue, error)
	SetDB(db *gorm.DB)
}

type userRevenueRepository struct {
	db *gorm.DB
}

func NewUserRevenueRepository(db *gorm.DB) UserRevenueRepository {
	return &userRevenueRepository{db: db}
}

func (r *userRevenueRepository) Add(ctx context.Context, uid string, cents int64) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "uid"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"revenue_cents": gorm.Expr("revenue_cents + ?", cents)}),
	}).Create(&model.UserRevenue{UID: uid, RevenueCents: cents}).Error
}

func (r *userRevenueRepository) Deduct(ctx context.Context, uid string, cents int64) error {
	res := r.db.WithContext(ctx).
		Model(&model.UserRevenue{}).
		Where("uid = ? AND revenue_cents >= ?", uid, cents).
		Update("revenue_cents", gorm.Expr("revenue_cents - ?", cents))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *userRevenueRepository) Get(ctx context.Context, uid string) (*model.UserRevenue, error) {
	var ur model.UserRevenue
	if err := r.db.WithContext(ctx).Where("uid = ?", uid).FirstOrCreate(&ur, &model.UserRevenue{UID: uid}).Error; err != nil {
		return nil, err
	}
	return &ur, nil
}

func (r *userRevenueRepository) SetDB(db *gorm.DB) {
	r.db = db
}
