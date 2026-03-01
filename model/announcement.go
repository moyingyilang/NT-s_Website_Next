package model

import (
	"time"

	"gorm.io/gorm"
)

type Announcement struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Title     string         `json:"title"` // 公告标题
	Summary   string         `json:"summary"` // 公告内容
	Date      string         `json:"date"` // 发布日期（YYYY-MM-DD）
	Tags      []string       `json:"tags" gorm:"type:json"` // 标签（JSON存储）
	Visible   bool           `json:"visible" gorm:"default:false"` // 是否公开
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
