package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ========== JWT工具 ==========
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userID string) (string, error) {
	expire := time.Now().Add(time.Second * JWTExpire)
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expire),
			Issuer:    "ntc",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTSecret))
}

func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("token无效")
}

// ========== 认证业务 ==========
func generateUserID() string {
	rand.Seed(time.Now().UnixNano())
	for {
		id := fmt.Sprintf("%010d", rand.Int63n(9000000000)+1000000000)
		var user User
		if DB.Where("id = ?", id).First(&user).Error == gorm.ErrRecordNotFound {
			return id
		}
	}
}

func Register(username, password string) (*User, error) {
	var exist User
	if DB.Where("username = ?", username).First(&exist).Error != gorm.ErrRecordNotFound {
		return nil, errors.New("用户名已存在")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("密码加密失败")
	}

	user := &User{
		ID:         generateUserID(),
		Username:   username,
		Password:   string(hash),
		Nickname:   username,
		Registered: time.Now().Unix(),
	}
	if err := DB.Create(user).Error; err != nil {
		return nil, errors.New("注册失败")
	}
	return user, nil
}

func Login(username, password string) (string, *User, error) {
	var user User
	if DB.Where("username = ?", username).First(&user).Error == gorm.ErrRecordNotFound {
		if DB.Where("id = ?", username).First(&user).Error == gorm.ErrRecordNotFound {
			return "", nil, errors.New("账号或密码错误")
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.New("账号或密码错误")
	}

	token, err := GenerateToken(user.ID)
	if err != nil {
		return "", nil, errors.New("登录失败")
	}
	return token, &user, nil
}

func GetUserByID(userID string) (*User, error) {
	var user User
	if err := DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, errors.New("用户不存在")
	}
	return &user, nil
}

// ========== 好友业务 ==========
func SendFriendRequest(fromUID, toUID string) error {
	if fromUID == toUID {
		return errors.New("不能添加自己")
	}

	var req Friend
	if DB.Where("user_id = ? AND friend_id = ?", fromUID, toUID).First(&req).Error == nil {
		return errors.New("已发送申请")
	}

	return DB.Create(&Friend{UserID: fromUID, FriendID: toUID, Status: StatusPending}).Error
}

func HandleFriendRequest(myUID, fromUID string, accept bool) error {
	var req Friend
	if DB.Where("user_id = ? AND friend_id = ?", fromUID, myUID).First(&req).Error != nil {
		return errors.New("申请不存在")
	}
	if req.Status != StatusPending {
		return errors.New("申请已处理")
	}

	if accept {
		req.Status = StatusAccepted
		DB.Create(&Friend{UserID: myUID, FriendID: fromUID, Status: StatusAccepted})
	} else {
		req.Status = 2
	}
	return DB.Save(&req).Error
}

func GetFriendList(uid string) ([]User, error) {
	var friends []Friend
	if err := DB.Where("user_id = ? AND status = ?", uid, StatusAccepted).Find(&friends).Error; err != nil {
		return nil, err
	}

	var uids []string
	for _, f := range friends {
		uids = append(uids, f.FriendID)
	}

	var users []User
	return users, DB.Where("id IN (?)", uids).Find(&users).Error
}

func GetFriendRequests(uid string) ([]User, error) {
	var reqs []Friend
	if err := DB.Where("friend_id = ? AND status = ?", uid, StatusPending).Find(&reqs).Error; err != nil {
		return nil, err
	}

	var uids []string
	for _, r := range reqs {
		uids = append(uids, r.UserID)
	}

	var users []User
	return users, DB.Where("id IN (?)", uids).Find(&users).Error
}

// ========== 聊天业务 ==========
func SaveMessage(fromUID, toUID, content, typ string) error {
	return DB.Create(&Message{
		SenderID:   fromUID,
		ReceiverID: toUID,
		Content:    content,
		Type:       typ,
	}).Error
}

func GetMessages(uid1, uid2 string, limit int) ([]Message, error) {
	var msgs []Message
	return msgs, DB.Where(
		"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		uid1, uid2, uid2, uid1,
	).Order("created_at asc").Limit(limit).Find(&msgs).Error
}

// ========== 文件业务 ==========
func FileMD5(file *os.File) string {
	hash := md5.New()
	_, _ = io.Copy(hash, file)
	return hex.EncodeToString(hash.Sum(nil))
}

func UploadFile(userID string, file *os.File, fileName string) (*File, error) {
	// 计算MD5
	md5Str := FileMD5(file)
	file.Seek(0, 0)

	// 保存文件
	fileExt := filepath.Ext(fileName)
	saveName := uuid.NewString() + fileExt
	savePath := filepath.Join(StorageDir, saveName)

	outFile, err := os.Create(savePath)
	if err != nil {
		return nil, errors.New("创建文件失败")
	}
	defer outFile.Close()

	fileSize, err := io.Copy(outFile, file)
	if err != nil {
		return nil, errors.New("保存文件失败")
	}

	// 存储到数据库
	fileModel := &File{
		ID:           uuid.NewString(),
		OriginalName: fileName,
		Path:         savePath,
		Size:         fileSize,
		MD5:          md5Str,
		UserID:       userID,
		Type:         "file",
	}
	if err := DB.Create(fileModel).Error; err != nil {
		os.Remove(savePath)
		return nil, errors.New("存储文件记录失败")
	}
	return fileModel, nil
}

func UploadChatImage(userID string, file *os.File, fileName string) (string, error) {
	// 保存到聊天图片目录
	_ = os.MkdirAll(filepath.Join(StorageDir, "chat_images"), 0755)
	md5Str := FileMD5(file)
	file.Seek(0, 0)

	saveName := md5Str + filepath.Ext(fileName)
	savePath := filepath.Join(StorageDir, "chat_images", saveName)

	outFile, err := os.Create(savePath)
	if err != nil {
		return "", errors.New("创建图片失败")
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		return "", errors.New("保存图片失败")
	}
	return savePath, nil
}

// ========== 公告业务 ==========
func GetAnnouncements() ([]Announcement, error) {
	var anns []Announcement
	return anns, DB.Where("visible = ?", true).Order("created_at desc").Find(&anns).Error
}

// ========== Wiki业务 ==========
func GetWikiDocs() ([]string, error) {
	wikiDir := filepath.Join(StorageDir, "wiki")
	_ = os.MkdirAll(wikiDir, 0755)

	// 获取所有.md文件
	files, err := filepath.Glob(filepath.Join(wikiDir, "*.md"))
	if err != nil {
		return nil, err
	}

	var docs []string
	for _, f := range files {
		docs = append(docs, filepath.Base(f))
	}
	return docs, nil
}

func GetWikiDocContent(docName string) (string, error) {
	wikiDir := filepath.Join(StorageDir, "wiki")
	filePath := filepath.Join(wikiDir, docName)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", errors.New("文档不存在")
	}
	return string(content), nil
}

// ========== 用户资料更新业务 ==========
func UpdateUser(userID string, updateData map[string]interface{}) (*User, error) {
    var user User
    if err := DB.Where("id = ?", userID).First(&user).Error; err != nil {
        return nil, errors.New("用户不存在")
    }

    // 处理密码更新
    if oldPwd, ok := updateData["old_password"].(string); ok {
        if newPwd, ok2 := updateData["password"].(string); ok2 {
            if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPwd)); err != nil {
                return nil, errors.New("旧密码错误")
            }
            hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
            if err != nil {
                return nil, errors.New("密码加密失败")
            }
            user.Password = string(hash)
        }
    }

    // 处理昵称更新
    if nickname, ok := updateData["nickname"].(string); ok && nickname != "" {
        user.Nickname = nickname
    }

    // 处理好友验证模式更新
    if verifyMode, ok := updateData["verify_mode"].(string); ok {
        user.VerifyMode = verifyMode
    }

    // 处理简介更新
    if bio, ok := updateData["bio"].(string); ok {
        user.Bio = bio
    }

    if err := DB.Save(&user).Error; err != nil {
        return nil, errors.New("更新失败")
    }
    return &user, nil
}

// 上传头像
func UploadUserAvatar(userID string, file *os.File) (string, error) {
    // 计算MD5
    md5Str := FileMD5(file)
    file.Seek(0, 0)

    // 保存头像
    _ = os.MkdirAll(filepath.Join(StorageDir, "avatars"), 0755)
    saveName := md5Str + ".png"
    savePath := filepath.Join(StorageDir, "avatars", saveName)

    outFile, err := os.Create(savePath)
    if err != nil {
        return "", errors.New("创建头像文件失败")
    }
    defer outFile.Close()

    _, err = io.Copy(outFile, file)
    if err != nil {
        return "", errors.New("保存头像失败")
    }

    // 更新用户头像字段
    var user User
    if err := DB.Where("id = ?", userID).First(&user).Error; err != nil {
        return "", errors.New("用户不存在")
    }
    user.Avatar = "/storage/avatars/" + saveName
    DB.Save(&user)

    return user.Avatar, nil
}

// ========== 删除好友业务 ==========
func DeleteFriend(userID, friendID string) error {
    // 检查好友关系
    var friend Friend
    if err := DB.Where("user_id = ? AND friend_id = ? AND status = ?", userID, friendID, StatusAccepted).First(&friend).Error; err != nil {
        return errors.New("好友关系不存在")
    }

    // 删除双方好友记录
    DB.Where("user_id = ? AND friend_id = ?", userID, friendID).Delete(&Friend{})
    DB.Where("user_id = ? AND friend_id = ?", friendID, userID).Delete(&Friend{})

    // 删除聊天记录
    DB.Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)", userID, friendID, friendID, userID).Delete(&Message{})

    return nil
}
