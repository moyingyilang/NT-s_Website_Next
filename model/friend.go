package model

import (
	"time"

	"gorm.io/gorm"
)

type FriendStatus string

const (
	StatusPending  FriendStatus = "pending"  // 待审核
	StatusAccepted FriendStatus = "accepted" // 已通过
)

type Friend struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    string         `gorm:"index" json:"user_id"` // 发起者ID
	FriendID  string         `gorm:"index" json:"friend_id"` // 接收者ID
	Status    FriendStatus   `json:"status" gorm:"default:'pending'"`
	CreatedAt time.Time      `json:"since"` // 建立时间
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
