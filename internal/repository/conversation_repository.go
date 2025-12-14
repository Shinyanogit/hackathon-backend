package repository

import (
	"context"
	"errors"
	"time"

	"github.com/shinyyama/hackathon-backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConversationRepository interface {
	FindOrCreate(ctx context.Context, itemID uint64, sellerUID, buyerUID string) (*model.Conversation, error)
	FindOrCreateByItem(ctx context.Context, itemID uint64, sellerUID string) (*model.Conversation, error)
	FindByUser(ctx context.Context, uid string) ([]model.Conversation, error)
	FindByID(ctx context.Context, id uint64) (*model.Conversation, error)
	FindByItem(ctx context.Context, itemID uint64) (*model.Conversation, error)
	CreateMessage(ctx context.Context, msg *model.Message) error
	ListMessages(ctx context.Context, convID uint64) ([]model.Message, error)
	FindMessage(ctx context.Context, msgID uint64) (*model.Message, error)
	UpsertState(ctx context.Context, convID uint64, uid string) error
	HasUnread(ctx context.Context, convID uint64, uid string, lastMessageID uint64) (bool, error)
	LastMessage(ctx context.Context, convID uint64) (*model.Message, error)
	DeleteMessage(ctx context.Context, convID uint64, msgID uint64, uid string) error
}

type conversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return &conversationRepository{db: db}
}

func (r *conversationRepository) FindOrCreate(ctx context.Context, itemID uint64, sellerUID, buyerUID string) (*model.Conversation, error) {
	cv := model.Conversation{ItemID: itemID, SellerUID: sellerUID, BuyerUID: buyerUID}
	if err := r.db.WithContext(ctx).
		Where("item_id = ? AND buyer_uid = ?", itemID, buyerUID).
		FirstOrCreate(&cv).Error; err != nil {
		return nil, err
	}
	return &cv, nil
}

func (r *conversationRepository) FindByUser(ctx context.Context, uid string) ([]model.Conversation, error) {
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
	var cv model.Conversation
	if err := r.db.WithContext(ctx).First(&cv, id).Error; err != nil {
		return nil, err
	}
	return &cv, nil
}

func (r *conversationRepository) FindByItem(ctx context.Context, itemID uint64) (*model.Conversation, error) {
	var cv model.Conversation
	if err := r.db.WithContext(ctx).
		Where("item_id = ?", itemID).
		First(&cv).Error; err != nil {
		return nil, err
	}
	return &cv, nil
}

func (r *conversationRepository) FindOrCreateByItem(ctx context.Context, itemID uint64, sellerUID string) (*model.Conversation, error) {
	cv := model.Conversation{ItemID: itemID, SellerUID: sellerUID}
	if err := r.db.WithContext(ctx).
		Where("item_id = ?", itemID).
		FirstOrCreate(&cv).Error; err != nil {
		return nil, err
	}
	return &cv, nil
}

func (r *conversationRepository) CreateMessage(ctx context.Context, msg *model.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *conversationRepository) ListMessages(ctx context.Context, convID uint64) ([]model.Message, error) {
	var msgs []model.Message
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", convID).
		Order("id ASC").
		Find(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}

func (r *conversationRepository) UpsertState(ctx context.Context, convID uint64, uid string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "conversation_id"}, {Name: "uid"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"last_read_at": gorm.Expr("CURRENT_TIMESTAMP")}),
		},
	).Create(&model.ConversationState{ConversationID: convID, UID: uid, LastReadAt: now}).Error
}

func (r *conversationRepository) LastMessage(ctx context.Context, convID uint64) (*model.Message, error) {
	var msg model.Message
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", convID).
		Order("id DESC").
		Limit(1).
		First(&msg).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *conversationRepository) HasUnread(ctx context.Context, convID uint64, uid string, lastMessageID uint64) (bool, error) {
	var state model.ConversationState
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND uid = ?", convID, uid).
		First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// no state -> unread if there is any message
		return lastMessageID > 0, nil
	}
	if err != nil {
		return false, err
	}
	// If last read message id stored? We only have timestamp. Use last_read_at vs messages created_at.
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ? AND id > ?", convID, lastMessageID).
		Where("created_at > ?", state.LastReadAt).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *conversationRepository) DeleteMessage(ctx context.Context, convID uint64, msgID uint64, uid string) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND conversation_id = ? AND sender_uid = ?", msgID, convID, uid).
		Delete(&model.Message{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *conversationRepository) FindMessage(ctx context.Context, msgID uint64) (*model.Message, error) {
	var msg model.Message
	if err := r.db.WithContext(ctx).First(&msg, msgID).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}
