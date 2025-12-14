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
	Create(ctx context.Context, title, description string, price uint, imageURL *string, categorySlug string, sellerUID string) (*model.Item, error)
	Get(ctx context.Context, id uint64) (*model.Item, error)
	List(ctx context.Context, limit, offset int, categorySlug, query string) ([]model.Item, int64, error)
	ListBySeller(ctx context.Context, sellerUID string) ([]model.Item, error)
	UpdateOwned(ctx context.Context, id uint64, sellerUID string, title, description string, price uint, imageURL *string, categorySlug string) (*model.Item, error)
}

type itemService struct {
	repo repository.ItemRepository
}

func NewItemService(repo repository.ItemRepository) ItemService {
	return &itemService{repo: repo}
}

func (s *itemService) Create(ctx context.Context, title, description string, price uint, imageURL *string, categorySlug string, sellerUID string) (*model.Item, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	categorySlug = strings.TrimSpace(categorySlug)
	sellerUID = strings.TrimSpace(sellerUID)
	if title == "" || len(title) > 120 {
		return nil, errors.New("invalid title")
	}
	if description == "" {
		return nil, errors.New("invalid description")
	}
	if categorySlug == "" {
		return nil, errors.New("category is required")
	}
	if sellerUID == "" {
		return nil, errors.New("seller is required")
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
		SellerUID:    sellerUID,
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

func (s *itemService) List(ctx context.Context, limit, offset int, categorySlug, query string) ([]model.Item, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset, strings.TrimSpace(categorySlug), strings.TrimSpace(query))
}

func (s *itemService) ListBySeller(ctx context.Context, sellerUID string) ([]model.Item, error) {
	if sellerUID == "" {
		return nil, errors.New("seller is required")
	}
	return s.repo.ListBySeller(ctx, sellerUID)
}

func (s *itemService) UpdateOwned(ctx context.Context, id uint64, sellerUID string, title, description string, price uint, imageURL *string, categorySlug string) (*model.Item, error) {
	if sellerUID == "" {
		return nil, errors.New("seller is required")
	}
	if imageURL != nil && strings.HasPrefix(strings.TrimSpace(*imageURL), "data:") {
		return nil, errors.New("imageUrl must be a URL, not data URI")
	}
	fields := map[string]interface{}{}
	if title != "" {
		if len(strings.TrimSpace(title)) > 120 {
			return nil, errors.New("invalid title")
		}
		fields["title"] = strings.TrimSpace(title)
	}
	if description != "" {
		fields["description"] = strings.TrimSpace(description)
	}
	if price > 0 {
		fields["price"] = price
	}
	if imageURL != nil {
		fields["image_url"] = imageURL
	}
	if categorySlug != "" {
		fields["category_slug"] = strings.TrimSpace(categorySlug)
	}
	if len(fields) == 0 {
		return nil, errors.New("no fields to update")
	}
	if err := s.repo.UpdateBySeller(ctx, id, sellerUID, fields); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.Get(ctx, id)
}
