package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/shinyyama/hackathon-backend/internal/co2ctx"
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
	if price < 100 {
		return nil, errors.New("price must be at least 100")
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
	// 自動CO2推定（出品と同時にバックグラウンド実行）
	if s.co2Estimator != nil {
		img := ""
		if imageURL != nil {
			img = *imageURL
		}
		go func(id uint64, t, d, imgURL string, price uint) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[co2] item=%d stage=create_panic panic=%v", id, r)
				}
			}()
			ctxShort, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			val, err := s.co2Estimator.Estimate(ctxShort, t, d, imgURL)
			if err != nil {
				log.Printf("[co2] item=%d stage=create_estimate_err err=%v", id, err)
				return
			}
			limit := float64(price) * 0.05
			if val > limit {
				log.Printf("[co2] item=%d stage=create_cap raw=%.3f cap=%.3f", id, val, limit)
				val = limit
			}
			if rows, err := s.repo.UpdateCO2(ctxShort, id, &val); err != nil {
				log.Printf("[co2] item=%d stage=create_db_err err=%v", id, err)
			} else {
				log.Printf("[co2] item=%d stage=create_db rows=%d", id, rows)
			}
		}(item.ID, item.Title, item.Description, img, item.Price)
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
		if price < 100 {
			return nil, errors.New("price must be at least 100")
		}
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
	rid := co2ctx.RID(ctx)
	itemID := id
	log.Printf("[co2] rid=%s item=%d stage=start", rid, itemID)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[co2] rid=%s item=%d panic=%v", rid, itemID, r)
		}
	}()
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		log.Printf("[co2] rid=%s item=%d stage=find err=%v", rid, itemID, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if item.SellerUID != sellerUID {
		log.Printf("[co2] rid=%s item=%d stage=validate err=forbidden", rid, itemID)
		return nil, errors.New("forbidden")
	}
	ctxShort, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	ctxShort = co2ctx.WithItemID(ctxShort, itemID)
	estimateStart := time.Now()
	img := ""
	if item.ImageURL != nil {
		img = *item.ImageURL
	}
	val, err := s.co2Estimator.Estimate(ctxShort, item.Title, item.Description, img)
	if err != nil {
		log.Printf("[co2] rid=%s item=%d stage=estimate err=%v", rid, itemID, err)
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("timeout")
		}
		return nil, err
	}
	log.Printf("[co2] rid=%s item=%d stage=estimate_done durMs=%d", rid, itemID, time.Since(estimateStart).Milliseconds())
	// cap co2 to 5% of price
	limit := float64(item.Price) * 0.05
	if val > limit {
		log.Printf("[co2] rid=%s item=%d stage=cap applied raw=%.3f cap=%.3f", rid, itemID, val, limit)
		val = limit
	}
	dbStart := time.Now()
	rows, err := s.repo.UpdateCO2Force(ctxShort, id, &val)
	if err != nil {
		log.Printf("[co2] rid=%s item=%d stage=db_update_err err=%v", rid, itemID, err)
		return nil, err
	}
	log.Printf("[co2] rid=%s item=%d stage=db_update rows=%d durMs=%d", rid, itemID, rows, time.Since(dbStart).Milliseconds())
	return &val, nil
}
