package service

import (
	"context"
	"errors"

	"github.com/shinyyama/hackathon-backend/internal/repository"
	"gorm.io/gorm"
)

type RevenueService interface {
	Get(ctx context.Context, uid string) (int64, error)
	Deduct(ctx context.Context, uid string, cents int64) (int64, error)
	Add(ctx context.Context, uid string, cents int64) error
}

type revenueService struct {
	repo repository.UserRevenueRepository
}

func NewRevenueService(repo repository.UserRevenueRepository) RevenueService {
	return &revenueService{repo: repo}
}

func (s *revenueService) Get(ctx context.Context, uid string) (int64, error) {
	r, err := s.repo.Get(ctx, uid)
	if err != nil {
		return 0, err
	}
	return r.RevenueCents, nil
}

func (s *revenueService) Deduct(ctx context.Context, uid string, cents int64) (int64, error) {
	if cents <= 0 {
		return 0, errors.New("amount must be positive")
	}
	if err := s.repo.Deduct(ctx, uid, cents); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("insufficient balance")
		}
		return 0, err
	}
	return s.Get(ctx, uid)
}

func (s *revenueService) Add(ctx context.Context, uid string, cents int64) error {
	if cents <= 0 {
		return nil
	}
	return s.repo.Add(ctx, uid, cents)
}
