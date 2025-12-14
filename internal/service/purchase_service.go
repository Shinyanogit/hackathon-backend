package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/repository"
	"gorm.io/gorm"
)

var ErrAlreadyPurchased = errors.New("already_purchased")
var ErrForbidden = errors.New("forbidden")

type PurchaseService interface {
	PurchaseItem(ctx context.Context, itemID uint64, buyerUID string) (*model.Purchase, error)
	GetByItem(ctx context.Context, itemID uint64, uid string) (*model.Purchase, error)
	MarkShipped(ctx context.Context, purchaseID uint64, sellerUID string) (*model.Purchase, error)
	MarkDelivered(ctx context.Context, purchaseID uint64, buyerUID string) (*model.Purchase, error)
	Cancel(ctx context.Context, purchaseID uint64, buyerUID string) (*model.Purchase, error)
	ListByBuyer(ctx context.Context, buyerUID string) ([]PurchaseWithItem, error)
}

type purchaseService struct {
	purchaseRepo repository.PurchaseRepository
	itemRepo     repository.ItemRepository
	convRepo     repository.ConversationRepository
}

type PurchaseWithItem struct {
	Purchase model.Purchase
	Item     *model.Item
}

func NewPurchaseService(purchaseRepo repository.PurchaseRepository, itemRepo repository.ItemRepository, convRepo repository.ConversationRepository) PurchaseService {
	return &purchaseService{purchaseRepo: purchaseRepo, itemRepo: itemRepo, convRepo: convRepo}
}

func (s *purchaseService) PurchaseItem(ctx context.Context, itemID uint64, buyerUID string) (*model.Purchase, error) {
	if buyerUID == "" {
		return nil, errors.New("buyer is required")
	}
	item, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if item.SellerUID == "" {
		return nil, errors.New("item has no seller")
	}
	if item.SellerUID == buyerUID {
		return nil, errors.New("cannot buy your own item")
	}
	if existing, err := s.purchaseRepo.FindByItem(ctx, itemID); err == nil && existing != nil {
		if existing.Status != model.PurchaseStatusCanceled {
			return existing, ErrAlreadyPurchased
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	cv, err := s.convRepo.FindOrCreate(ctx, itemID, item.SellerUID, buyerUID)
	if err != nil {
		return nil, err
	}
	escapedBuyer := url.QueryEscape(buyerUID)
	shippingQR := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=240x240&data=item-%d-buyer-%s", itemID, escapedBuyer)
	shippingNote := "コンビニ端末で「発送受付」を選び、店員にこのQRコードを見せて発送手続きを完了してください。梱包用の袋は店舗で用意されます。"

	p := &model.Purchase{
		ItemID:         itemID,
		BuyerUID:       buyerUID,
		SellerUID:      item.SellerUID,
		ConversationID: cv.ID,
		Status:         model.PurchaseStatusPendingShipment,
		ShippingQRURL:  shippingQR,
		ShippingNote:   shippingNote,
	}
	if err := s.purchaseRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	_ = s.convRepo.CreateMessage(ctx, &model.Message{
		ConversationID: cv.ID,
		SenderUID:      buyerUID,
		SenderName:     "購入者",
		Body:           "購入手続きを完了しました。発送用QRコードを使って、コンビニでの発送手続きをお願いします。",
	})
	return p, nil
}

func (s *purchaseService) GetByItem(ctx context.Context, itemID uint64, uid string) (*model.Purchase, error) {
	p, err := s.purchaseRepo.FindByItem(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if uid != "" && uid != p.BuyerUID && uid != p.SellerUID {
		return nil, ErrForbidden
	}
	return p, nil
}

func (s *purchaseService) MarkShipped(ctx context.Context, purchaseID uint64, sellerUID string) (*model.Purchase, error) {
	p, err := s.purchaseRepo.FindByID(ctx, purchaseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if p.SellerUID != sellerUID {
		return nil, ErrForbidden
	}
	now := time.Now()
	p.Status = model.PurchaseStatusShipped
	p.ShippedAt = &now
	if err := s.purchaseRepo.Update(ctx, p); err != nil {
		return nil, err
	}
	if p.ConversationID != 0 {
		_ = s.convRepo.CreateMessage(ctx, &model.Message{
			ConversationID: p.ConversationID,
			SenderUID:      sellerUID,
			SenderName:     "出品者",
			Body:           "発送手続きが完了しました。追跡番号はコンビニ受付の控えをご確認ください。",
		})
	}
	return p, nil
}

func (s *purchaseService) MarkDelivered(ctx context.Context, purchaseID uint64, buyerUID string) (*model.Purchase, error) {
	p, err := s.purchaseRepo.FindByID(ctx, purchaseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if p.BuyerUID != buyerUID {
		return nil, ErrForbidden
	}
	if p.Status == model.PurchaseStatusDelivered {
		return p, nil
	}
	now := time.Now()
	p.Status = model.PurchaseStatusDelivered
	p.DeliveredAt = &now
	if err := s.purchaseRepo.Update(ctx, p); err != nil {
		return nil, err
	}
	if p.ConversationID != 0 {
		_ = s.convRepo.CreateMessage(ctx, &model.Message{
			ConversationID: p.ConversationID,
			SenderUID:      buyerUID,
			SenderName:     "購入者",
			Body:           "商品を受け取りました。ありがとうございました！",
		})
	}
	return p, nil
}

func (s *purchaseService) Cancel(ctx context.Context, purchaseID uint64, buyerUID string) (*model.Purchase, error) {
	p, err := s.purchaseRepo.FindByID(ctx, purchaseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if p.BuyerUID != buyerUID {
		return nil, ErrForbidden
	}
	if p.Status != model.PurchaseStatusPendingShipment {
		return nil, errors.New("cannot cancel after shipment")
	}
	p.Status = model.PurchaseStatusCanceled
	p.DeliveredAt = nil
	p.ShippedAt = nil
	if err := s.purchaseRepo.Update(ctx, p); err != nil {
		return nil, err
	}
	if p.ConversationID != 0 {
		_ = s.convRepo.CreateMessage(ctx, &model.Message{
			ConversationID: p.ConversationID,
			SenderUID:      buyerUID,
			SenderName:     "購入者",
			Body:           "購入をキャンセルしました。",
		})
	}
	return p, nil
}

func (s *purchaseService) ListByBuyer(ctx context.Context, buyerUID string) ([]PurchaseWithItem, error) {
	if buyerUID == "" {
		return nil, errors.New("buyer is required")
	}
	purchases, err := s.purchaseRepo.ListByBuyer(ctx, buyerUID)
	if err != nil {
		return nil, err
	}
	resp := make([]PurchaseWithItem, 0, len(purchases))
	for _, p := range purchases {
		item, _ := s.itemRepo.FindByID(ctx, p.ItemID)
		resp = append(resp, PurchaseWithItem{
			Purchase: p,
			Item:     item,
		})
	}
	return resp, nil
}
