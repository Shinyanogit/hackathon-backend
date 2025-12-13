package model

import "time"

type ItemImage struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	ItemID    uint64    `gorm:"column:item_id;not null;index:idx_item_images_item_id"`
	ImageURL  string    `gorm:"column:image_url;size:512;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (ItemImage) TableName() string {
	return "item_images"
}
