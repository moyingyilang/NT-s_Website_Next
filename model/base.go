package model

import (
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
    "log"
    "os"
    "time"

    "go-fiber-web/config"
)

var DB *gorm.DB

// 初始化数据库
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
        panic("数据库连接失败: " + err.Error())
    }

    DB = db
    // 自动迁移（在此添加模型）
    // DB.AutoMigrate(&User{})
}

// 示例模型
type User struct {
    ID   uint   `gorm:"primaryKey" json:"id"`
    Name string `json:"name"`
}
