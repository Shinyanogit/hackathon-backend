package model

import "time"

type ConversationState struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`
	ConversationID uint64    `gorm:"column:conversation_id;uniqueIndex:uniq_conv_uid"`
	UID            string    `gorm:"column:uid;size:128;uniqueIndex:uniq_conv_uid"`
	LastReadAt     time.Time `gorm:"column:last_read_at;autoUpdateTime"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func (ConversationState) TableName() string {
	return "conversation_states"
}
