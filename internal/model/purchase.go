package model

import "time"

type PurchaseStatus string

const (
	PurchaseStatusPendingShipment PurchaseStatus = "pending_shipment"
	PurchaseStatusShipped         PurchaseStatus = "shipped"
	PurchaseStatusDelivered       PurchaseStatus = "delivered"
	PurchaseStatusCanceled        PurchaseStatus = "canceled"
)

type Purchase struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement"`
	ItemID         uint64         `gorm:"column:item_id;index;not null"`
	BuyerUID       string         `gorm:"column:buyer_uid;size:128;index;not null"`
	SellerUID      string         `gorm:"column:seller_uid;size:128;index;not null"`
	ConversationID uint64         `gorm:"column:conversation_id;index"`
	Status         PurchaseStatus `gorm:"column:status;size:32;not null"`
	ShippingQRURL  string         `gorm:"column:shipping_qr_url;type:text"`
	ShippingNote   string         `gorm:"column:shipping_note;type:text"`
	PointsUsed     float64        `gorm:"column:points_used"`
	PaidYen        int64          `gorm:"column:paid_yen"`
	ShippedAt      *time.Time     `gorm:"column:shipped_at"`
	DeliveredAt    *time.Time     `gorm:"column:delivered_at"`
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime"`
}

func (Purchase) TableName() string {
	return "purchases"
}
