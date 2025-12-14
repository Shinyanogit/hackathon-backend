package model

import "time"

type Message struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID  uint64    `gorm:"column:conversation_id;index" json:"conversationId"`
	SenderUID       string    `gorm:"column:sender_uid;size:128;index" json:"senderUid"`
	SenderName      string    `gorm:"column:sender_name;size:120" json:"senderName"`
	SenderIconURL   *string   `gorm:"column:sender_icon_url;type:text" json:"senderIconUrl,omitempty"`
	ParentMessageID *uint64   `gorm:"column:parent_message_id" json:"parentMessageId,omitempty"`
	Depth           int       `gorm:"column:depth;default:0" json:"depth"`
	Body            string    `gorm:"type:text;not null" json:"body"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (Message) TableName() string {
	return "messages"
}
