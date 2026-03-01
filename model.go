package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strconv"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid" // 修正：导入uuid包
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 全局配置（内置，不单独建文件）
const (
	DBPath     = "./ntc.db"
	StorageDir = "./data"
	AvatarDir  = "./data/avatars"
	UploadDir  = "./data/upFile"
	FileMap    = "./data/upFile/files.json"
	TempDir    = "./data/temp"
	JWTSecret  = "ntc_chat_key"
	JWTExpire  = 3600 * 24

	StatusPending  = 0
	StatusAccepted = 1
	MessageTypeText  = "text"
	MessageTypeImage = "image"

	MaxFileSize   = 500 * 1024 * 1024 // 500MB
	MinChunkSize  = 1 * 1024 * 1024  // 1MB
	MaxChunkSize  = 50 * 1024 * 1024 // 50MB
)

var (
	AllowedExtensions = []string{"zip", "rar", "7z", "tar", "gz", "bz2", "xz", "tgz"}
	AllowedImageTypes = []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	DB                *gorm.DB // 全局DB，跨文件调用
)

// 数据模型（跨文件调用，结构体首字母大写）
type User struct {
	ID         string         `gorm:"primaryKey;type:varchar(10);not null" json:"id"`
	Username   string         `gorm:"unique;type:varchar(50);not null" json:"username"`
	Password   string         `gorm:"type:varchar(100);not null" json:"-"`
	Nickname   string         `gorm:"type:varchar(50);not null" json:"nickname"`
	Bio        string         `gorm:"type:varchar(255);default:''" json:"bio"`
	Avatar     string         `gorm:"type:varchar(255);default:''" json:"avatar"`
	VerifyMode string         `gorm:"type:varchar(20);default:'need_verify'" json:"verify_mode"`
	Registered int64          `gorm:"not null" json:"registered"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type Friend struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    string         `gorm:"index" json:"user_id"`
	FriendID  string         `gorm:"index" json:"friend_id"`
	Status    int            `gorm:"default:0" json:"status"` // 0-待处理 1-已接受
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Message struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	SenderID   string         `gorm:"index" json:"sender_id"`
	ReceiverID string         `gorm:"index" json:"receiver_id"`
	Content    string         `json:"content"`
	Type       string         `gorm:"default:'text'" json:"type"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type File struct {
	ID           string         `gorm:"primaryKey;type:varchar(36);not null" json:"id"`
	OriginalName string         `json:"original_name"`
	Path         string         `json:"-"`
	Size         int64          `json:"size"`
	MD5          string         `gorm:"index" json:"md5"`
	UserID       string         `gorm:"index" json:"user_id"`
	Type         string         `json:"type"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type Announcement struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Title     string         `gorm:"type:varchar(100);not null" json:"title"`
	Summary   string         `gorm:"type:text;not null" json:"summary"`
	Date      string         `gorm:"type:varchar(20);not null" json:"date"`
	Tags      []string       `gorm:"type:json" json:"tags"`
	Visible   bool           `gorm:"default:true" json:"visible"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type UploadFile struct {
	Original string `json:"original"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	MD5      string `json:"md5"`
	Time     int64  `json:"time"`
}

// 数据库初始化（跨文件调用，首字母大写）
func InitDB() {
	dirs := []string{StorageDir, AvatarDir, UploadDir, TempDir, UploadDir + "/FileName"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("创建目录失败: %v", err)
		}
	}

	if _, err := os.Stat(FileMap); os.IsNotExist(err) {
		_ = os.WriteFile(FileMap, []byte("{}"), 0644)
	}

	db, err := gorm.Open(sqlite.Open(DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	err = db.AutoMigrate(&User{}, &Friend{}, &Message{}, &File{}, &Announcement{})
	if err != nil {
		log.Fatalf("建表失败: %v", err)
	}

	var count int64
	db.Model(&Announcement{}).Count(&count)
	if count == 0 {
		_ = db.Create(&Announcement{
			Title:   "网站安全公告",
			Summary: "NTC已正式上线！支持用户注册、聊天、文件上传等功能，请注意账号安全。",
			Date:    time.Now().Format("2006-01-02"),
			Tags:    []string{"完成建设"},
			Visible: true,
		}).Error
	}

	DB = db
	log.Println("数据库初始化成功！")
}

// 业务逻辑（完全对标PHP，修正语法错误，跨文件调用函数首字母大写）
// 1. 用户相关
func GenerateUserID() string {
	// Go没有do-while，用for循环替代，修正语法
	for {
		min := 1000000000
		max := 9999999999
		id := strconv.Itoa(min + int(time.Now().UnixNano())%(max-min+1))
		var user User
		if DB.Where("id = ?", id).First(&user).Error != nil {
			return id
		}
	}
}

func GetUserByID(id string) (*User, error) {
	var user User
	if err := DB.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByUsername(username string) (*User, error) {
	var user User
	if err := DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func CreateUser(user *User) error {
	return DB.Create(user).Error
}

func UpdateUser(user *User) error {
	return DB.Save(user).Error
}

// 2. 好友相关
func GetFriends(userID string) ([]*User, error) {
	var friends []Friend
	if err := DB.Where("user_id = ? AND status = ?", userID, StatusAccepted).Find(&friends).Error; err != nil {
		return nil, err
	}

	var userList []*User
	for _, f := range friends {
		user, err := GetUserByID(f.FriendID)
		if err == nil {
			userList = append(userList, user)
		}
	}
	return userList, nil
}

func SendFriendRequest(fromID, toID string) error {
	var exist Friend
	if DB.Where("user_id = ? AND friend_id = ?", toID, fromID).First(&exist).Error == nil {
		return fmt.Errorf("请求已发送，请等待")
	}

	return DB.Create(&Friend{
		UserID:   toID,
		FriendID: fromID,
		Status:   StatusPending,
	}).Error
}

func AcceptFriendRequest(fromID, toID string) error {
	tx := DB.Begin()
	if err := tx.Model(&Friend{}).Where("user_id = ? AND friend_id = ?", toID, fromID).Update("status", StatusAccepted).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Create(&Friend{
		UserID:   fromID,
		FriendID: toID,
		Status:   StatusAccepted,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// 3. 消息相关
func SaveMessage(msg *Message) error {
	return DB.Create(msg).Error
}

func GetMessages(senderID, receiverID string) ([]*Message, error) {
	var messages []*Message
	err := DB.Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		senderID, receiverID, receiverID, senderID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}

// 4. 文件上传相关
func CalculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func CalculateMD5FromFile(file *multipart.FileHeader) (string, error) {
	f, err := file.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func SaveFileMapping(id string, file *UploadFile) error {
	data, err := os.ReadFile(FileMap)
	if err != nil {
		return err
	}

	var fileMap map[string]*UploadFile
	if err := json.Unmarshal(data, &fileMap); err != nil {
		fileMap = make(map[string]*UploadFile)
	}

	fileMap[id] = file
	newData, err := json.MarshalIndent(fileMap, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(FileMap, newData, 0644)
}

func SearchFileByMD5(md5 string) ([]*UploadFile, []string) {
	data, err := os.ReadFile(FileMap)
	if err != nil {
		return nil, nil
	}

	var fileMap map[string]*UploadFile
	if err := json.Unmarshal(data, &fileMap); err != nil {
		return nil, nil
	}

	var files []*UploadFile
	var urls []string
	for id, file := range fileMap {
		if file.MD5 == md5 {
			files = append(files, file)
			urls = append(urls, fmt.Sprintf("/upF?action=download&id=%s", id))
		}
	}
	return files, urls
}

// 5. 公告相关
func GetVisibleAnnouncements() ([]*Announcement, error) {
	var announcements []*Announcement
	err := DB.Where("visible = ?", true).Order("created_at DESC").Find(&announcements).Error
	return announcements, err
}

func CreateAnnouncement(ann *Announcement) error {
	return DB.Create(ann).Error
}

// 分块上传用，跨文件调用（首字母大写）
func GenerateUniqueId() string {
	return uuid.NewString()
}
