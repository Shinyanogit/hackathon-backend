package model

import "time"

type Item struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement"`
	Title        string    `gorm:"size:120;not null"`
	Description  string    `gorm:"type:text;not null"`
	Price        uint      `gorm:"not null"`
	CategorySlug string    `gorm:"column:category_slug;size:64;index"`
	ImageURL     *string   `gorm:"column:image_url;type:text" json:"imageUrl,omitempty"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (Item) TableName() string {
	return "items"
}
