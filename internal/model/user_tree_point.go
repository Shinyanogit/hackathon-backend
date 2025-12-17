package model

import "time"

// UserTreePoint stores cumulative and balance tree points (tree-years unit).
type UserTreePoint struct {
	UID           string    `gorm:"column:uid;primaryKey;size:128"`
	TotalPoints   float64   `gorm:"column:total_points;not null;default:0"`
	BalancePoints float64   `gorm:"column:balance_points;not null;default:0"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (UserTreePoint) TableName() string {
	return "user_tree_points"
}
