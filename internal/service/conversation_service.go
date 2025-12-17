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
	ListByUser(ctx context.Context, uid string) ([]ConversationWithUnread, error)
	Get(ctx context.Context, convID uint64, uid string) (*model.Conversation, error)
	ListMessages(ctx context.Context, convID uint64, uid string) ([]model.Message, error)
	CreateMessage(ctx context.Context, convID uint64, uid, body, senderName string, senderIconURL *string) error
	MarkRead(ctx context.Context, convID uint64, uid string) error
	DeleteMessage(ctx context.Context, convID uint64, msgID uint64, uid string) error
	ThreadByItem(ctx context.Context, itemID uint64) (*model.Conversation, []model.Message, error)
	PostMessageToItem(ctx context.Context, itemID uint64, uid, text, senderName string, senderIconURL *string, parentID *uint64) (*model.Message, *model.Conversation, error)
}

type ConversationWithUnread struct {
	model.Conversation
	HasUnread bool   `json:"hasUnread"`
	LastMsgID uint64 `json:"lastMessageId"`
}

type conversationService struct {
	convRepo repository.ConversationRepository
	itemRepo repository.ItemRepository
	notify   NotificationService
}

func NewConversationService(convRepo repository.ConversationRepository, itemRepo repository.ItemRepository, notify NotificationService) ConversationService {
	return &conversationService{convRepo: convRepo, itemRepo: itemRepo, notify: notify}
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

func (s *conversationService) ListByUser(ctx context.Context, uid string) ([]ConversationWithUnread, error) {
	convs, err := s.convRepo.FindByUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	resp := make([]ConversationWithUnread, 0, len(convs))
	for _, cv := range convs {
		last, err := s.convRepo.LastMessage(ctx, cv.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		hasUnread := false
		var lastID uint64
		if last != nil {
			lastID = last.ID
			hasUnread, err = s.convRepo.HasUnread(ctx, cv.ID, uid, last.ID)
			if err != nil {
				return nil, err
			}
		}
		resp = append(resp, ConversationWithUnread{
			Conversation: cv,
			HasUnread:    hasUnread,
			LastMsgID:    lastID,
		})
	}
	return resp, nil
}

func (s *conversationService) Get(ctx context.Context, convID uint64, uid string) (*model.Conversation, error) {
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
	return cv, nil
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
	if err := s.convRepo.CreateMessage(ctx, msg); err != nil {
		return err
	}
	s.notifyDM(ctx, cv, uid, msg.ID, msg.Body)
	return nil
}

func (s *conversationService) MarkRead(ctx context.Context, convID uint64, uid string) error {
	return s.convRepo.UpsertState(ctx, convID, uid)
}

func (s *conversationService) DeleteMessage(ctx context.Context, convID uint64, msgID uint64, uid string) error {
	msg, err := s.convRepo.FindMessage(ctx, msgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if msg.ConversationID != convID {
		return ErrNotFound
	}
	if msg.SenderUID != uid {
		return errors.New("forbidden")
	}
	return s.convRepo.DeleteMessage(ctx, convID, msgID, uid)
}

func (s *conversationService) ThreadByItem(ctx context.Context, itemID uint64) (*model.Conversation, []model.Message, error) {
	cv, err := s.convRepo.FindByItem(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, []model.Message{}, nil
		}
		return nil, nil, err
	}
	msgs, err := s.convRepo.ListMessages(ctx, cv.ID)
	if err != nil {
		return nil, nil, err
	}
	return cv, msgs, nil
}

func (s *conversationService) PostMessageToItem(ctx context.Context, itemID uint64, uid, text, senderName string, senderIconURL *string, parentID *uint64) (*model.Message, *model.Conversation, error) {
	const maxDepth = 3
	if text == "" {
		return nil, nil, errors.New("text is required")
	}
	if senderName == "" {
		senderName = uid
	}
	item, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	cv, err := s.convRepo.FindOrCreateByItem(ctx, itemID, item.SellerUID)
	if err != nil {
		return nil, nil, err
	}
	msg := &model.Message{
		ConversationID: cv.ID,
		SenderUID:      uid,
		SenderName:     senderName,
		SenderIconURL:  senderIconURL,
	}
	if parentID != nil {
		parent, err := s.convRepo.FindMessage(ctx, *parentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil, errors.New("parent not found")
			}
			return nil, nil, err
		}
		if parent.ConversationID != cv.ID {
			return nil, nil, errors.New("parent not in conversation")
		}
		msg.ParentMessageID = parentID
		msg.Depth = parent.Depth + 1
		if msg.Depth > maxDepth {
			return nil, nil, errors.New("depth exceeded")
		}
	} else {
		// root投稿は出品者は禁止
		if uid == item.SellerUID {
			return nil, nil, errors.New("seller cannot create root message")
		}
		msg.Depth = 0
	}
	msg.Body = text
	if err := s.convRepo.CreateMessage(ctx, msg); err != nil {
		return nil, nil, err
	}
	s.notifyDM(ctx, cv, uid, msg.ID, msg.Body)
	return msg, cv, nil
}

func (s *conversationService) notifyDM(ctx context.Context, cv *model.Conversation, senderUID string, msgID uint64, body string) {
	if s.notify == nil || cv == nil {
		return
	}
	var target string
	switch senderUID {
	case cv.SellerUID:
		target = cv.BuyerUID
	case cv.BuyerUID:
		target = cv.SellerUID
	}
	if target == "" || target == senderUID {
		return
	}
	ctxShort, cancel := withShortDeadline(ctx)
	defer cancel()
	preview := body
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	s.notify.Notify(ctxShort, target, "dm_received", "新しいメッセージ", preview, &cv.ItemID, &cv.ID, nil)
}
