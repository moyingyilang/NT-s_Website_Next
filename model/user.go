package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID         string         `gorm:"primaryKey" json:"id"` // 10位数字ID
	Username   string         `gorm:"uniqueIndex" json:"username"` // 登录用户名
	Password   string         `json:"-"` // 密码哈希（不返回前端）
	Nickname   string         `json:"nickname"` // 昵称
	Avatar     string         `json:"avatar,omitempty"` // 头像路径
	Bio        string         `json:"bio,omitempty"` // 简介
	VerifyMode string         `json:"verify_mode" gorm:"default:'need_verify'"` // 好友验证方式
	Registered int64          `json:"registered"` // 注册时间戳
	CreatedAt  time.Time      `json:"-"`
	UpdatedAt  time.Time      `json:"-"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}
