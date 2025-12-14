package model

import "time"

type Conversation struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ItemID    uint64    `gorm:"column:item_id;index:idx_item_buyer,unique" json:"itemId"`
	SellerUID string    `gorm:"column:seller_uid;size:128;index" json:"sellerUid"`
	BuyerUID  string    `gorm:"column:buyer_uid;size:128;index:idx_item_buyer,unique" json:"buyerUid"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (Conversation) TableName() string {
	return "conversations"
}
