package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/repository"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

type ItemService interface {
	Create(ctx context.Context, title, description string, price uint, imageURL *string, categorySlug string, sellerUID string) (*model.Item, error)
	Get(ctx context.Context, id uint64) (*model.Item, error)
	List(ctx context.Context, limit, offset int, categorySlug, query, sellerUID string) ([]model.Item, int64, error)
	ListBySeller(ctx context.Context, sellerUID string) ([]model.Item, error)
	UpdateOwned(ctx context.Context, id uint64, sellerUID string, title, description string, price uint, imageURL *string, categorySlug string, status string) (*model.Item, error)
	DeleteOwned(ctx context.Context, id uint64, sellerUID string) error
	EstimateCO2(ctx context.Context, id uint64, sellerUID string) (*float64, error)
}

type itemService struct {
	repo         repository.ItemRepository
	co2Estimator interface {
		Estimate(ctx context.Context, title, description, imageURL string) (float64, error)
	}
}

func NewItemService(repo repository.ItemRepository, estimator interface {
	Estimate(ctx context.Context, title, description, imageURL string) (float64, error)
}) ItemService {
	return &itemService{repo: repo, co2Estimator: estimator}
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
		Status:       model.ItemStatusListed,
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

func (s *itemService) List(ctx context.Context, limit, offset int, categorySlug, query, sellerUID string) ([]model.Item, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset, strings.TrimSpace(categorySlug), strings.TrimSpace(query), strings.TrimSpace(sellerUID))
}

func (s *itemService) ListBySeller(ctx context.Context, sellerUID string) ([]model.Item, error) {
	if sellerUID == "" {
		return nil, errors.New("seller is required")
	}
	return s.repo.ListBySeller(ctx, sellerUID)
}

func (s *itemService) UpdateOwned(ctx context.Context, id uint64, sellerUID string, title, description string, price uint, imageURL *string, categorySlug string, status string) (*model.Item, error) {
	if sellerUID == "" {
		return nil, errors.New("seller is required")
	}
	if imageURL != nil && strings.HasPrefix(strings.TrimSpace(*imageURL), "data:") {
		return nil, errors.New("imageUrl must be a URL, not data URI")
	}
	fields := map[string]interface{}{}
	if status != "" {
		switch model.ItemStatus(status) {
		case model.ItemStatusListed, model.ItemStatusPaused:
			fields["status"] = model.ItemStatus(status)
		default:
			return nil, errors.New("invalid status")
		}
	}
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
	needsReset := false
	if imageURL != nil {
		fields["image_url"] = imageURL
		needsReset = true
	}
	if title != "" || description != "" {
		needsReset = true
	}
	if needsReset {
		fields["co2_kg"] = nil
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

func (s *itemService) DeleteOwned(ctx context.Context, id uint64, sellerUID string) error {
	if sellerUID == "" {
		return errors.New("seller is required")
	}
	if err := s.repo.DeleteBySeller(ctx, id, sellerUID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *itemService) EstimateCO2(ctx context.Context, id uint64, sellerUID string) (*float64, error) {
	if s.co2Estimator == nil {
		return nil, errors.New("co2 estimator not configured")
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[co2-estimate] recover panic: %v", r)
		}
	}()
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if item.SellerUID != sellerUID {
		return nil, errors.New("forbidden")
	}
	if item.ImageURL == nil || strings.TrimSpace(*item.ImageURL) == "" {
		return nil, errors.New("image is required for estimation")
	}
	ctxShort, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	val, err := s.co2Estimator.Estimate(ctxShort, item.Title, item.Description, *item.ImageURL)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("timeout")
		}
		return nil, err
	}
	dbStart := time.Now()
	if err := s.repo.UpdateCO2Force(ctxShort, id, &val); err != nil {
		return nil, err
	}
	log.Printf("[co2] db update took %v", time.Since(dbStart))
	return &val, nil
}
