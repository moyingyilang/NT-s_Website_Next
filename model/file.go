package model

import (
	"time"

	"gorm.io/gorm"
)

type FileType string

const (
	FilePackage FileType = "package" // 压缩包
	FileImage   FileType = "image"   // 聊天图片
)

type File struct {
	ID           string         `gorm:"primaryKey" json:"id"` // 唯一ID（MD5）
	OriginalName string         `json:"original"` // 原始文件名
	Path         string         `json:"path"` // 存储路径
	Size         int64          `json:"size"` // 文件大小（字节）
	Type         FileType       `json:"type"` // 文件类型
	MD5          string         `gorm:"uniqueIndex" json:"md5"` // 文件MD5
	UserID       string         `gorm:"index" json:"user_id"` // 上传者ID
	CreatedAt    time.Time      `json:"time"` // 上传时间
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
