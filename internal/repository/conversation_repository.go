package repository

import (
	"context"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"gorm.io/gorm"
)

type ConversationRepository interface {
	FindOrCreate(ctx context.Context, itemID uint64, sellerUID, buyerUID string) (*model.Conversation, error)
	FindByUser(ctx context.Context, uid string) ([]model.Conversation, error)
	FindByID(ctx context.Context, id uint64) (*model.Conversation, error)
	CreateMessage(ctx context.Context, msg *model.Message) error
	ListMessages(ctx context.Context, convID uint64) ([]model.Message, error)
	SetDB(db *gorm.DB)
}

type conversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return &conversationRepository{db: db}
}

func (r *conversationRepository) SetDB(db *gorm.DB) {
	r.db = db
}

func (r *conversationRepository) FindOrCreate(ctx context.Context, itemID uint64, sellerUID, buyerUID string) (*model.Conversation, error) {
	if r.db == nil {
		return nil, ErrDBNotReady
	}
	cv := model.Conversation{ItemID: itemID, SellerUID: sellerUID, BuyerUID: buyerUID}
	if err := r.db.WithContext(ctx).
		Where("item_id = ? AND buyer_uid = ?", itemID, buyerUID).
		FirstOrCreate(&cv).Error; err != nil {
		return nil, err
	}
	return &cv, nil
}

func (r *conversationRepository) FindByUser(ctx context.Context, uid string) ([]model.Conversation, error) {
	if r.db == nil {
		return nil, ErrDBNotReady
	}
	var list []model.Conversation
	if err := r.db.WithContext(ctx).
		Where("seller_uid = ? OR buyer_uid = ?", uid, uid).
		Order("id DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *conversationRepository) FindByID(ctx context.Context, id uint64) (*model.Conversation, error) {
	if r.db == nil {
		return nil, ErrDBNotReady
	}
	var cv model.Conversation
	if err := r.db.WithContext(ctx).First(&cv, id).Error; err != nil {
		return nil, err
	}
	return &cv, nil
}

func (r *conversationRepository) CreateMessage(ctx context.Context, msg *model.Message) error {
	if r.db == nil {
		return ErrDBNotReady
	}
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *conversationRepository) ListMessages(ctx context.Context, convID uint64) ([]model.Message, error) {
	if r.db == nil {
		return nil, ErrDBNotReady
	}
	var msgs []model.Message
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", convID).
		Order("id ASC").
		Find(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}
