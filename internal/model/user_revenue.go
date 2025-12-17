package model

import "time"

type UserRevenue struct {
	UID          string    `gorm:"column:uid;primaryKey;size:128"`
	RevenueCents int64     `gorm:"column:revenue_cents;not null;default:0"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

func (UserRevenue) TableName() string {
	return "user_revenues"
}
