package main

import (
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	DBPath     = "./ntc.db"
	StorageDir = "./storage"
	JWTSecret  = "ntc_chat_key"
	JWTExpire  = 3600 * 24

	StatusPending   = 0
	StatusAccepted  = 1
	MessageTypeText  = "text"
	MessageTypeImage = "image"
)

type User struct {
	ID         string    `gorm:"primaryKey;type:varchar(10);not null" json:"id"`
	Username   string    `gorm:"unique;type:varchar(50);not null" json:"username"`
	Password   string    `gorm:"type:varchar(100);not null" json:"-"`
	Nickname   string    `gorm:"type:varchar(50);not null" json:"nickname"`
	Bio        string    `gorm:"type:varchar(255);default:''" json:"bio"`
	Avatar     string    `gorm:"type:varchar(255);default:''" json:"avatar"`
	VerifyMode string    `gorm:"type:varchar(20);default:'need_verify'" json:"verify_mode"`
	Registered int64     `gorm:"not null" json:"registered"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type Friend struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"index" json:"user_id"`
	FriendID  string    `gorm:"index" json:"friend_id"`
	Status    int       `gorm:"default:0" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Message struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SenderID   string    `gorm:"index" json:"sender_id"`
	ReceiverID string    `gorm:"index" json:"receiver_id"`
	Content    string    `json:"content"`
	Type       string    `gorm:"default:'text'" json:"type"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type File struct {
	ID           string    `gorm:"primaryKey;type:varchar(36);not null" json:"id"`
	OriginalName string    `json:"original_name"`
	Path         string    `json:"-"`
	Size         int64     `json:"size"`
	MD5          string    `gorm:"index" json:"md5"`
	UserID       string    `gorm:"index" json:"user_id"`
	Type         string    `json:"type"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type Announcement struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"type:varchar(100);not null" json:"title"`
	Summary   string    `gorm:"type:text;not null" json:"summary"`
	Date      string    `gorm:"type:varchar(20);not null" json:"date"`
	Tags      []string  `gorm:"type:json" json:"tags"`
	Visible   bool      `gorm:"default:true" json:"visible"`
	CreatedAt time.Time `json:"created_at"`
}

var DB *gorm.DB

func InitDB() {
	_ = os.MkdirAll(StorageDir, 0755)

	// 🔥 唯一正确、无参数、无CGO、纯Go写法
	db, err := gorm.Open(sqlite.Open(DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("数据库打开失败: " + err.Error())
	}

	err = db.AutoMigrate(&User{}, &Friend{}, &Message{}, &File{}, &Announcement{})
	if err != nil {
		panic("建表失败: " + err.Error())
	}

	var count int64
	db.Model(&Announcement{}).Count(&count)
	if count == 0 {
		db.Create(&Announcement{
			Title:   "网站安全公告",
			Summary: "NTC已正式上线！支持用户注册、聊天、文件上传等功能，请注意账号安全。",
			Date:    time.Now().Format("2006-01-01"),
			Tags:    []string{"系统"},
			Visible: true,
		})
	}

	DB = db
}
