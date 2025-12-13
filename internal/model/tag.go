package model

import "time"

type Tag struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"size:120;not null;uniqueIndex:uk_tags_name"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (Tag) TableName() string {
	return "tags"
}
