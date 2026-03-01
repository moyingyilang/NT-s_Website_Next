package model

import (
	"fmt"
	"log"
	"os"
	"time"

	"go-fiber-web/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// 初始化数据库（自动迁移表）
func InitDB() {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      true,
		},
	)

	db, err := gorm.Open(sqlite.Open(config.DBPath), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic(fmt.Sprintf("数据库连接失败: %v", err))
	}

	// 自动迁移所有表
	err = db.AutoMigrate(
		&User{},
		&Friend{},
		&Message{},
		&File{},
		&Announcement{},
	)
	if err != nil {
		panic(fmt.Sprintf("表迁移失败: %v", err))
	}

	DB = db
	fmt.Println("数据库初始化成功")
}
