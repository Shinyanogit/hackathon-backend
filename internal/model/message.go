package model

import "time"

type Message struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID uint64    `gorm:"column:conversation_id;index" json:"conversationId"`
	SenderUID      string    `gorm:"column:sender_uid;size:128;index" json:"senderUid"`
	Body           string    `gorm:"type:text;not null" json:"body"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (Message) TableName() string {
	return "messages"
}
