package model

import (
	"time"

	"gorm.io/gorm"
)

type MessageType string

const (
	MsgText  MessageType = "text"  // 文本消息
	MsgImage MessageType = "image" // 图片消息
)

type Message struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	FromID    string         `gorm:"index" json:"from"` // 发送者ID
	ToID      string         `gorm:"index" json:"to"` // 接收者ID
	Content   string         `json:"content"` // 消息内容（文本/图片路径）
	Type      MessageType    `json:"type" gorm:"default:'text'"`
	Timestamp int64          `json:"timestamp"` // 发送时间戳
	CreatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
