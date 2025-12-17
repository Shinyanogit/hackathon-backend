package repository

import (
	"context"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"gorm.io/gorm"
)

type UserTreePointRepository interface {
	Add(ctx context.Context, uid string, points float64) error
	Deduct(ctx context.Context, uid string, points float64) error
	Get(ctx context.Context, uid string) (*model.UserTreePoint, error)
	SetDB(db *gorm.DB)
}

type userTreePointRepository struct {
	db *gorm.DB
}

func NewUserTreePointRepository(db *gorm.DB) UserTreePointRepository {
	return &userTreePointRepository{db: db}
}

func (r *userTreePointRepository) Add(ctx context.Context, uid string, points float64) error {
	if points <= 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tp model.UserTreePoint
		if err := tx.Where("uid = ?", uid).
			FirstOrCreate(&tp, &model.UserTreePoint{UID: uid}).Error; err != nil {
			return err
		}
		return tx.Model(&tp).
			Updates(map[string]interface{}{
				"total_points":   tp.TotalPoints + points,
				"balance_points": tp.BalancePoints + points,
			}).Error
	})
}

func (r *userTreePointRepository) Deduct(ctx context.Context, uid string, points float64) error {
	if points <= 0 {
		return nil
	}
	res := r.db.WithContext(ctx).
		Model(&model.UserTreePoint{}).
		Where("uid = ? AND balance_points >= ?", uid, points).
		Updates(map[string]interface{}{
			"balance_points": gorm.Expr("balance_points - ?", points),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *userTreePointRepository) Get(ctx context.Context, uid string) (*model.UserTreePoint, error) {
	var tp model.UserTreePoint
	if err := r.db.WithContext(ctx).
		Where("uid = ?", uid).
		FirstOrCreate(&tp, &model.UserTreePoint{UID: uid}).Error; err != nil {
		return nil, err
	}
	return &tp, nil
}

func (r *userTreePointRepository) SetDB(db *gorm.DB) {
	r.db = db
}
