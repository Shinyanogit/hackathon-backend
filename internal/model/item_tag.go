package model

import "time"

type ItemTag struct {
	ItemID    uint64    `gorm:"column:item_id;not null;primaryKey"`
	TagID     uint64    `gorm:"column:tag_id;not null;primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (ItemTag) TableName() string {
	return "item_tags"
}
