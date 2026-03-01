package model

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"go-fiber-web/config"
	"time"
)

var DB *gorm.DB

// 初始化数据库
func InitDB() {
	dsn := config.DBUser + ":" + config.DBPwd + "@tcp(" + config.DBHost + ":" + config.DBPort + ")/" + config.DBDatabase + "?charset=" + config.DBCharset + "&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 开启日志
	})
	if err != nil {
		panic("数据库连接失败：" + err.Error())
	}

	// 自动迁移表
	_ = db.AutoMigrate(
		&User{}, &Friend{}, &Message{}, &File{}, &Announcement{},
	)

	DB = db
}

// 基础模型（软删除+时间戳）
type BaseModel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// User 用户模型
type User struct {
	BaseModel
	ID       string `gorm:"type:varchar(36);primaryKey" json:"id"` // UUID
	Username string `gorm:"type:varchar(50);unique;not null" json:"username"`
	Password string `gorm:"type:varchar(100);not null" json:"-"`
	Nickname string `gorm:"type:varchar(50);default:''" json:"nickname"`
	Avatar   string `gorm:"type:varchar(255);default:''" json:"avatar"`
	Bio      string `gorm:"type:text;default:''" json:"bio"`
}

// Friend 好友模型
type Friend struct {
	BaseModel
	UserID   string `gorm:"type:varchar(36);index" json:"user_id"`   // 自己ID
	FriendID string `gorm:"type:varchar(36);index" json:"friend_id"` // 好友ID
	Status   int    `gorm:"type:tinyint;default:1" json:"status"`    // 1已添加，0已删除
}

// Message 消息模型
type Message struct {
	BaseModel
	ID        string `gorm:"type:varchar(36);primaryKey" json:"id"`
	SenderID  string `gorm:"type:varchar(36);index" json:"from"` // 发送者ID
	ReceiverID string `gorm:"type:varchar(36);index" json:"to"`   // 接收者ID
	Content   string `gorm:"type:text" json:"content"`           // 消息内容
	Type      string `gorm:"type:varchar(20);default:'text'" json:"type"` // text/image
}

// File 文件模型
type File struct {
	BaseModel
	ID           string `gorm:"type:varchar(36);primaryKey" json:"id"`
	OriginalName string `gorm:"type:varchar(255);not null" json:"original_name"` // 原始文件名
	Path         string `gorm:"type:varchar(255);not null" json:"-"`             // 存储路径（不返回前端）
	Size         int64  `gorm:"type:bigint;not null" json:"size"`                 // 文件大小（字节）
	Type         string `gorm:"type:varchar(20);default:'package'" json:"type"`  // package/avatar/wiki
	MD5          string `gorm:"type:varchar(32);index;not null" json:"md5"`      // 文件MD5
	UserID       string `gorm:"type:varchar(36);index;not null" json:"user_id"`  // 上传用户ID
}

// Announcement 公告模型
type Announcement struct {
	BaseModel
	Title   string   `gorm:"type:varchar(100);not null" json:"title"`
	Summary string   `gorm:"type:text;not null" json:"summary"` // 公告内容
	Date    string   `gorm:"type:varchar(20);not null" json:"date"` // 发布日期（2025-01-01）
	Tags    []string `gorm:"type:json;default:[]" json:"tags"`      // 标签
	Visible bool     `gorm:"type:bool;default:true" json:"visible"` // 是否公开
}
