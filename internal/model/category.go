package model

import "time"

type Category struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"size:120;not null;uniqueIndex:uk_categories_name"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (Category) TableName() string {
	return "categories"
}
