package service

import (
	"context"
	"errors"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/repository"
	"gorm.io/gorm"
)

type ConversationService interface {
	CreateOrGet(ctx context.Context, itemID uint64, buyerUID string) (*model.Conversation, error)
	ListByUser(ctx context.Context, uid string) ([]model.Conversation, error)
	ListMessages(ctx context.Context, convID uint64, uid string) ([]model.Message, error)
	CreateMessage(ctx context.Context, convID uint64, uid, body, senderName string, senderIconURL *string) error
}

type conversationService struct {
	convRepo repository.ConversationRepository
	itemRepo repository.ItemRepository
}

func NewConversationService(convRepo repository.ConversationRepository, itemRepo repository.ItemRepository) ConversationService {
	return &conversationService{convRepo: convRepo, itemRepo: itemRepo}
}

func (s *conversationService) CreateOrGet(ctx context.Context, itemID uint64, buyerUID string) (*model.Conversation, error) {
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
		return nil, errors.New("cannot chat with yourself")
	}
	return s.convRepo.FindOrCreate(ctx, itemID, item.SellerUID, buyerUID)
}

func (s *conversationService) ListByUser(ctx context.Context, uid string) ([]model.Conversation, error) {
	return s.convRepo.FindByUser(ctx, uid)
}

func (s *conversationService) ListMessages(ctx context.Context, convID uint64, uid string) ([]model.Message, error) {
	cv, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if cv.BuyerUID != uid && cv.SellerUID != uid {
		return nil, errors.New("forbidden")
	}
	return s.convRepo.ListMessages(ctx, convID)
}

func (s *conversationService) CreateMessage(ctx context.Context, convID uint64, uid, body, senderName string, senderIconURL *string) error {
	if body == "" {
		return errors.New("body is required")
	}
	cv, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if cv.BuyerUID != uid && cv.SellerUID != uid {
		return errors.New("forbidden")
	}
	msg := &model.Message{
		ConversationID: convID,
		SenderUID:      uid,
		Body:           body,
		SenderName:     senderName,
		SenderIconURL:  senderIconURL,
	}
	return s.convRepo.CreateMessage(ctx, msg)
}
