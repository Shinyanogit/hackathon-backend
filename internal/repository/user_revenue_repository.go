package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserWallet struct {
	UID        string    `gorm:"column:uid;primaryKey;size:128"`
	BalanceYen int64     `gorm:"column:revenue_balance_yen;not null;default:0"`
	TotalYen   int64     `gorm:"column:revenue_total_yen;not null;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserWallet) TableName() string { return "users" }

type UserRevenueRepository interface {
	Add(ctx context.Context, uid string, yen int64) error
	Deduct(ctx context.Context, uid string, yen int64) error
	Get(ctx context.Context, uid string) (*UserWallet, error)
	SetDB(db *gorm.DB)
}

type userRevenueRepository struct {
	db *gorm.DB
}

func NewUserRevenueRepository(db *gorm.DB) UserRevenueRepository {
	return &userRevenueRepository{db: db}
}

func (r *userRevenueRepository) Add(ctx context.Context, uid string, yen int64) error {
	if yen <= 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "uid"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"revenue_balance_yen": gorm.Expr("COALESCE(revenue_balance_yen,0) + ?", yen),
			"revenue_total_yen":   gorm.Expr("COALESCE(revenue_total_yen,0) + ?", yen),
			"updated_at":          gorm.Expr("NOW()"),
		}),
	}).Create(&UserWallet{UID: uid, BalanceYen: yen, TotalYen: yen}).Error
}

func (r *userRevenueRepository) Deduct(ctx context.Context, uid string, yen int64) error {
	if yen <= 0 {
		return nil
	}
	res := r.db.WithContext(ctx).
		Model(&UserWallet{}).
		Where("uid = ? AND COALESCE(revenue_balance_yen,0) >= ?", uid, yen).
		Updates(map[string]interface{}{
			"revenue_balance_yen": gorm.Expr("COALESCE(revenue_balance_yen,0) - ?", yen),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *userRevenueRepository) Get(ctx context.Context, uid string) (*UserWallet, error) {
	var ur UserWallet
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		FirstOrCreate(&ur, &UserWallet{UID: uid}).Error; err != nil {
		return nil, err
	}
	return &ur, nil
}

func (r *userRevenueRepository) SetDB(db *gorm.DB) {
	r.db = db
}
