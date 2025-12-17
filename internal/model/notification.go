package model

import "time"

type Notification struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`
	UserUID        string    `gorm:"column:user_uid;size:128;index;not null"`
	Type           string    `gorm:"column:type;size:64;not null"`
	Title          string    `gorm:"column:title;size:255"`
	Body           string    `gorm:"column:body;type:text"`
	ItemID         *uint64   `gorm:"column:item_id;index"`
	ConversationID *uint64   `gorm:"column:conversation_id;index"`
	PurchaseID     *uint64   `gorm:"column:purchase_id;index"`
	ReadAt         *time.Time `gorm:"column:read_at"`
	CreatedAt      time.Time  `gorm:"autoCreateTime"`
}

func (Notification) TableName() string {
	return "notifications"
}
