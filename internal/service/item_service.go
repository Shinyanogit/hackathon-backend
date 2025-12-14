package service

import (
	"context"
	"errors"
	"strings"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/repository"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

type ItemService interface {
	Create(ctx context.Context, title, description string, price uint, imageURL *string, categorySlug string) (*model.Item, error)
	Get(ctx context.Context, id uint64) (*model.Item, error)
	List(ctx context.Context, limit, offset int, categorySlug string) ([]model.Item, int64, error)
}

type itemService struct {
	repo repository.ItemRepository
}

func NewItemService(repo repository.ItemRepository) ItemService {
	return &itemService{repo: repo}
}

func (s *itemService) Create(ctx context.Context, title, description string, price uint, imageURL *string, categorySlug string) (*model.Item, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	categorySlug = strings.TrimSpace(categorySlug)
	if title == "" || len(title) > 120 {
		return nil, errors.New("invalid title")
	}
	if description == "" {
		return nil, errors.New("invalid description")
	}
	if categorySlug == "" {
		return nil, errors.New("category is required")
	}
	if imageURL != nil && strings.HasPrefix(strings.TrimSpace(*imageURL), "data:") {
		return nil, errors.New("imageUrl must be a URL, not data URI")
	}

	item := &model.Item{
		Title:        title,
		Description:  description,
		Price:        price,
		ImageURL:     imageURL,
		CategorySlug: categorySlug,
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *itemService) Get(ctx context.Context, id uint64) (*model.Item, error) {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (s *itemService) List(ctx context.Context, limit, offset int, categorySlug string) ([]model.Item, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset, strings.TrimSpace(categorySlug))
}
