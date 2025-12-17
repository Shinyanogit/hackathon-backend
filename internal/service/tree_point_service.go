package service

import (
	"context"
	"errors"

	"github.com/shinyyama/hackathon-backend/internal/repository"
	"gorm.io/gorm"
)

type TreePointService interface {
	Get(ctx context.Context, uid string) (total float64, balance float64, err error)
	Add(ctx context.Context, uid string, points float64) error
	Deduct(ctx context.Context, uid string, points float64) (float64, float64, error)
}

type treePointService struct {
	repo repository.UserTreePointRepository
}

func NewTreePointService(repo repository.UserTreePointRepository) TreePointService {
	return &treePointService{repo: repo}
}

func (s *treePointService) Get(ctx context.Context, uid string) (float64, float64, error) {
	tp, err := s.repo.Get(ctx, uid)
	if err != nil {
		return 0, 0, err
	}
	return tp.TotalPoints, tp.BalancePoints, nil
}

func (s *treePointService) Add(ctx context.Context, uid string, points float64) error {
	if points <= 0 {
		return nil
	}
	return s.repo.Add(ctx, uid, points)
}

func (s *treePointService) Deduct(ctx context.Context, uid string, points float64) (float64, float64, error) {
	if points <= 0 {
		return 0, 0, errors.New("points must be positive")
	}
	if err := s.repo.Deduct(ctx, uid, points); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, errors.New("insufficient points")
		}
		return 0, 0, err
	}
	return s.Get(ctx, uid)
}
