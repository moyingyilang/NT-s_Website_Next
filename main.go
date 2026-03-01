package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
    "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/google/uuid"
)

func main() {
	InitDB()

	app := fiber.New(fiber.Config{
		DisablePreParseMultipartForm: true,
	})
	
	app.Use(logger.New(logger.Config{
 	Format: "[${time}] ${method} ${path} | status: ${status} | ip: ${ip}\n",
 	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
	}))

	app.Static("/", "./static")
	app.Static("/data", "./data")
	
	app.Post("/api/public/register", RegisterHandler)
    app.Post("/api/public/login", LoginHandler)
    app.Get("/api/public/announcements", AnnouncementListHandler)
    app.Get("/api/user/info", UserInfoHandler)
    app.Get("/api/friends", FriendListHandler)
    app.Post("/api/logout", LogoutHandler)
    app.Post("/api/user/update", UpdateUserHandler)
    app.Post("/api/user/avatar", UploadAvatarHandler)
    app.Get("/api/user/search", SearchUserHandler)
    app.Post("/api/friend/request", SendFriendRequestHandler)
    app.Post("/api/friend/accept", AcceptFriendRequestHandler)
    app.Post("/api/friend/reject", RejectFriendRequestHandler)
    app.Post("/api/friend/delete", DeleteFriendHandler)
    app.Get("/api/message/list", MessageListHandler)
    app.Post("/api/message/send", SendMessageHandler)
    app.Post("/api/message/image", UploadImageHandler)
    app.Post("/api/announcement/create", CreateAnnouncementHandler)
    app.Post("/api/upload/chunk", UploadChunkHandler)
    app.Post("/api/upload/merge", MergeChunkHandler)
    app.Get("/api/upload/status", UploadStatusHandler)
    app.Get("/api/upload/search", SearchFileHandler)
    app.Get("/api/upload/download", DownloadFileHandler)
	log.Println("服务器启动在 :8080")
	log.Fatal(app.Listen(":8080"))
}

// 1. 认证控制器（完全对标PHP）
func RegisterHandler(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" || password == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "用户名和密码不能为空"})
	}

	if _, err := GetUserByUsername(username); err == nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "用户名已存在"})
	}

	user := &User{
		ID:         GenerateUserID(),
		Username:   username,
		Password:   HashPassword(password),
		Nickname:   username,
		VerifyMode: "need_verify",
		Registered: time.Now().Unix(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	file, err := c.FormFile("avatar")
	if err == nil {
		mime := file.Header.Get("Content-Type")
		if !IsAllowedImageType(mime) {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "只允许上传JPG、PNG、GIF、WEBP格式的图片"})
		}
		if file.Size > 2*1024*1024 {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "图片不能超过2MB"})
		}

		ext := GetFileExt(file.Filename)
		filename := uuid.NewString() + "." + ext
		savePath := filepath.Join(AvatarDir, filename)
		if err := c.SaveFile(file, savePath); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "error": "头像保存失败"})
		}

		user.Avatar = "/data/avatars/" + filename
	}

	if err := CreateUser(user); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "注册失败"})
	}

	return c.JSON(fiber.Map{"success": true, "user": user})
}

func LoginHandler(c *fiber.Ctx) error {
	usernameOrId := c.FormValue("username")
	password := c.FormValue("password")

	if usernameOrId == "" || password == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "请输入用户名/ID和密码"})
	}

	user, err := GetUserByUsername(usernameOrId)
	if err != nil {
		user, err = GetUserByID(usernameOrId)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"success": false, "error": "用户名/ID或密码错误"})
		}
	}

	if !CheckPassword(password, user.Password) {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "用户名/ID或密码错误"})
	}

	token, err := GenerateToken(user.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "生成令牌失败"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"token":   token,
		"user":    user,
	})
}

func LogoutHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

func UserInfoHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "令牌无效"})
	}

	user, err := GetUserByID(claims.UserID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "用户不存在"})
	}

	return c.JSON(fiber.Map{"success": true, "user": user})
}

// 2. 用户控制器（完全对标PHP）
func UpdateUserHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	user, err := GetUserByID(claims.UserID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "用户不存在"})
	}

	oldPwd := c.FormValue("old_password")
	newPwd := c.FormValue("password")
	if oldPwd != "" && newPwd != "" {
		if !CheckPassword(oldPwd, user.Password) {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "旧密码错误"})
		}
		user.Password = HashPassword(newPwd)
	}

	if nickname := c.FormValue("nickname"); nickname != "" {
		user.Nickname = nickname
	}

	if bio := c.FormValue("bio"); bio != "" {
		user.Bio = bio
	}

	if verifyMode := c.FormValue("verify_mode"); verifyMode != "" {
		user.VerifyMode = verifyMode
	}

	user.UpdatedAt = time.Now()
	if err := UpdateUser(user); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "更新失败"})
	}

	return c.JSON(fiber.Map{"success": true, "user": user})
}

func UploadAvatarHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "未选择文件"})
	}

	if file.Size > 2*1024*1024 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "图片不能超过2MB"})
	}

	mime := file.Header.Get("Content-Type")
	if !IsAllowedImageType(mime) {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "只允许上传JPG、PNG、GIF、WEBP格式的图片"})
	}

	ext := GetFileExt(file.Filename)
	filename := uuid.NewString() + "." + ext
	savePath := filepath.Join(AvatarDir, filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "上传失败"})
	}

	user, err := GetUserByID(claims.UserID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "用户不存在"})
	}

	user.Avatar = "/data/avatars/" + filename
	if err := UpdateUser(user); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "更新头像失败"})
	}

	return c.JSON(fiber.Map{"success": true, "path": user.Avatar})
}

func SearchUserHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	// 修正：使用claims变量（解决未使用报错）
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	userId := c.Query("userId")
	if userId == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "请输入用户ID"})
	}

	// 额外校验：不能搜索自己（对标PHP逻辑）
	if userId == claims.UserID {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "不能搜索自己"})
	}

	user, err := GetUserByID(userId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "用户不存在"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user":    fiber.Map{"id": user.ID, "username": user.Username, "nickname": user.Nickname},
	})
}

// 3. 好友控制器（完全对标PHP）
func FriendListHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	friends, err := GetFriends(claims.UserID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "获取好友列表失败"})
	}

	return c.JSON(fiber.Map{"success": true, "friends": friends})
}

func SendFriendRequestHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	targetID := c.FormValue("targetId")
	if targetID == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "目标用户ID不能为空"})
	}

	if targetID == claims.UserID {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "不能添加自己为好友"})
	}

	targetUser, err := GetUserByID(targetID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "目标用户不存在"})
	}

	if targetUser.VerifyMode == "deny_all" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "对方禁止添加好友"})
	}

	// 修正：gorm事务无JSON方法，先Commit再返回JSON
	if targetUser.VerifyMode == "allow_all" {
		tx := DB.Begin()
		if err := tx.Create(&Friend{UserID: claims.UserID, FriendID: targetID, Status: StatusAccepted}).Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"success": false, "error": "添加好友失败"})
		}
		if err := tx.Create(&Friend{UserID: targetID, FriendID: claims.UserID, Status: StatusAccepted}).Error; err != nil {
			tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"success": false, "error": "添加好友失败"})
		}
		// 先执行Commit并判断错误
		if err := tx.Commit().Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "error": "添加好友失败"})
		}
		return c.JSON(fiber.Map{"success": true, "message": "添加好友成功"})
	}

	if err := SendFriendRequest(claims.UserID, targetID); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "好友请求已发送"})
}

func AcceptFriendRequestHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	requesterID := c.FormValue("requesterId")
	if err := AcceptFriendRequest(requesterID, claims.UserID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "接受请求失败"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func RejectFriendRequestHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	requesterID := c.FormValue("requesterId")
	if err := DB.Where("user_id = ? AND friend_id = ? AND status = ?", claims.UserID, requesterID, StatusPending).Delete(&Friend{}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "拒绝失败"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func DeleteFriendHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	friendID := c.FormValue("friendId")
	if friendID == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "缺少好友ID"})
	}

	tx := DB.Begin()
	if err := tx.Where("user_id = ? AND friend_id = ? AND status = ?", claims.UserID, friendID, StatusAccepted).Delete(&Friend{}).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "删除失败"})
	}

	if err := tx.Where("user_id = ? AND friend_id = ? AND status = ?", friendID, claims.UserID, StatusAccepted).Delete(&Friend{}).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "删除失败"})
	}

	if err := tx.Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)", claims.UserID, friendID, friendID, claims.UserID).Delete(&Message{}).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "删除消息失败"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "删除失败"})
	}

	return c.JSON(fiber.Map{"success": true})
}

// 4. 消息控制器（完全对标PHP）
func SendMessageHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	friendID := c.FormValue("friendId")
	content := c.FormValue("content")
	msgType := c.FormValue("type", MessageTypeText)

	if friendID == "" || content == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "参数不足"})
	}

	// 校验好友关系（对标PHP）
	var isFriend bool
	if err := DB.Where("user_id = ? AND friend_id = ? AND status = ?", claims.UserID, friendID, StatusAccepted).First(&Friend{}).Error; err == nil {
		isFriend = true
	}
	if !isFriend {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": "不是好友关系"})
	}

	msg := &Message{
		SenderID:   claims.UserID,
		ReceiverID: friendID,
		Content:    content,
		Type:       msgType,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := SaveMessage(msg); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "发送消息失败"})
	}

	return c.JSON(fiber.Map{"success": true})
}

func MessageListHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	friendID := c.Query("friendId")
	if friendID == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "缺少好友ID"})
	}

	messages, err := GetMessages(claims.UserID, friendID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "获取消息失败"})
	}

	return c.JSON(fiber.Map{"success": true, "messages": messages})
}

func UploadImageHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	_, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "未选择文件"})
	}

	if file.Size > 10*1024*1024 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "图片不能超过10MB"})
	}

	mime := file.Header.Get("Content-Type")
	if !IsAllowedImageType(mime) {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "只允许上传图片"})
	}

	md5Str, err := CalculateMD5FromFile(file)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "计算MD5失败"})
	}

	savePath := filepath.Join(UploadDir, md5Str)
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "上传失败"})
	}

	// 修正：处理json.Marshal多返回值（必判err）
	fileMapData, _ := os.ReadFile(FileMap)
	var fileMap []map[string]string
	_ = json.Unmarshal(fileMapData, &fileMap)
	fileMap = append(fileMap, map[string]string{
		"original": file.Filename,
		"md5":      md5Str,
		"mime":     mime,
	})
	// 强制处理Marshal错误
	fileMapJson, err := json.Marshal(fileMap)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "保存文件映射失败"})
	}
	_ = os.WriteFile(FileMap, fileMapJson, 0644)

	return c.JSON(fiber.Map{"success": true, "fileId": md5Str})
}

// 5. 公告控制器（完全对标PHP）
func AnnouncementListHandler(c *fiber.Ctx) error {
	anns, err := GetVisibleAnnouncements()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "获取公告失败"})
	}

	return c.JSON(fiber.Map{"success": true, "announcements": anns})
}

func CreateAnnouncementHandler(c *fiber.Ctx) error {
	// 对标PHP的admin=yes权限控制
	if c.Query("admin") != "yes" {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "页面不存在"})
	}

	title := c.FormValue("title")
	summary := c.FormValue("summary")
	visible := c.FormValue("visible") == "on"

	if title == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "公告标题不能为空"})
	}

	// 处理标签（对标PHP）
	var tags []string
	if err := json.Unmarshal([]byte(c.FormValue("tags")), &tags); err != nil {
		tags = []string{}
	}

	ann := &Announcement{
		Title:     title,
		Summary:   summary,
		Date:      time.Now().Format("2006-01-02"),
		Tags:      tags,
		Visible:   visible,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := CreateAnnouncement(ann); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "创建公告失败"})
	}

	return c.JSON(fiber.Map{"success": true, "announcement": ann})
}

// 6. 分块上传控制器（完全对标PHP）
func UploadChunkHandler(c *fiber.Ctx) error {
	fileId := c.FormValue("fileId")
	totalChunks, _ := strconv.Atoi(c.FormValue("totalChunks"))
	chunkIndex, _ := strconv.Atoi(c.FormValue("chunkIndex"))
	fileName := c.FormValue("fileName")
	fileSize, _ := strconv.ParseInt(c.FormValue("fileSize"), 10, 64)
	chunkSize, _ := strconv.ParseInt(c.FormValue("chunkSize"), 10, 64)

	// 对标PHP的参数校验
	if totalChunks <= 0 || chunkIndex < 0 || chunkIndex >= totalChunks {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "无效的块索引或总块数"})
	}
	if fileName == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "缺少文件名"})
	}
	if fileSize > MaxFileSize {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "文件超过最大允许大小"})
	}
	if chunkSize < MinChunkSize || chunkSize > MaxChunkSize {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "块大小超出允许范围"})
	}

	ext := GetFileExt(fileName)
	if !IsAllowedExtension(ext) {
		// 修正：导入strings包并使用
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "不支持的文件类型，仅允许：" + strings.Join(AllowedExtensions, ", ")})
	}

	file, err := c.FormFile("chunkFile")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "分块文件上传失败"})
	}
	if file.Size > chunkSize {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "上传的块大小超过声明值"})
	}

	chunkPath := filepath.Join(TempDir, fmt.Sprintf("%s_chunk_%d.part", fileId, chunkIndex))
	if err := c.SaveFile(file, chunkPath); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "保存分块失败"})
	}

	// 保存上传信息（对标PHP）
	infoPath := filepath.Join(TempDir, fmt.Sprintf("%s_info.json", fileId))
	info := map[string]interface{}{
		"uploadedChunks": []int{chunkIndex},
		"totalChunks":    totalChunks,
		"fileName":       fileName,
		"fileSize":       fileSize,
		"chunkSize":      chunkSize,
	}
	if _, err := os.Stat(infoPath); err == nil {
		var oldInfo map[string]interface{}
		data, _ := os.ReadFile(infoPath)
		_ = json.Unmarshal(data, &oldInfo)
		info["uploadedChunks"] = append(oldInfo["uploadedChunks"].([]int), chunkIndex)
	}
	// 修正：处理json.Marshal多返回值
	infoJson, err := json.Marshal(info)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "保存上传信息失败"})
	}
	_ = os.WriteFile(infoPath, infoJson, 0644)

	return c.JSON(fiber.Map{"success": true, "chunkIndex": chunkIndex})
}

func MergeChunkHandler(c *fiber.Ctx) error {
	fileId := c.FormValue("fileId")
	fileName := c.FormValue("fileName")
	totalChunks, _ := strconv.Atoi(c.FormValue("totalChunks"))

	if fileName == "" || totalChunks <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "参数不足"})
	}

	// 校验分块完整性（对标PHP）
	infoPath := filepath.Join(TempDir, fmt.Sprintf("%s_info.json", fileId))
	if _, err := os.Stat(infoPath); err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "未找到上传信息"})
	}
	data, _ := os.ReadFile(infoPath)
	var info map[string]interface{}
	_ = json.Unmarshal(data, &info)
	uploadedChunks := info["uploadedChunks"].([]int)
	if len(uploadedChunks) != totalChunks {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "分块不完整，无法合并"})
	}

	ext := GetFileExt(fileName)
	uniqueId := GenerateUniqueId()
	finalName := uniqueId + "." + ext
	finalPath := filepath.Join(UploadDir, finalName)

	finalFile, err := os.Create(finalPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "无法创建最终文件"})
	}
	defer finalFile.Close()

	// 合并分块（对标PHP）
	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(TempDir, fmt.Sprintf("%s_chunk_%d.part", fileId, i))
		if _, err := os.Stat(chunkPath); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("缺少分块 %d", i)})
		}
		chunkFile, _ := os.Open(chunkPath)
		_, _ = io.Copy(finalFile, chunkFile)
		_ = chunkFile.Close()
		_ = os.Remove(chunkPath)
	}

	// 计算MD5+保存映射（对标PHP）
	md5Str, _ := CalculateMD5(finalPath)
	fileInfo, _ := finalFile.Stat()
	// 修正：调用首字母大写的SaveFileMapping（Go导出规则）
	_ = SaveFileMapping(uniqueId, &UploadFile{
		Original: fileName,
		Path:     finalName,
		Size:     fileInfo.Size(),
		MD5:      md5Str,
		Time:     time.Now().Unix(),
	})

	// 清理临时文件（对标PHP）
	_ = os.Remove(infoPath)

	return c.JSON(fiber.Map{
		"success":     true,
		"fileId":      uniqueId,
		"downloadUrl": fmt.Sprintf("/upF?action=download&id=%s", uniqueId),
		"md5":         md5Str,
	})
}

func UploadStatusHandler(c *fiber.Ctx) error {
	fileId := c.Query("fileId")
	if fileId == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "缺少fileId参数"})
	}

	infoPath := filepath.Join(TempDir, fmt.Sprintf("%s_info.json", fileId))
	if _, err := os.Stat(infoPath); err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "未找到上传信息"})
	}

	data, _ := os.ReadFile(infoPath)
	var info map[string]interface{}
	_ = json.Unmarshal(data, &info)
	uploadedChunks := info["uploadedChunks"].([]int)

	return c.JSON(fiber.Map{"success": true, "uploadedChunks": uploadedChunks})
}

func SearchFileHandler(c *fiber.Ctx) error {
	md5 := c.Query("md5")
	if md5 == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "缺少MD5参数"})
	}

	files, urls := SearchFileByMD5(md5)
	return c.JSON(fiber.Map{"success": true, "files": files, "downloadUrls": urls})
}

func DownloadFileHandler(c *fiber.Ctx) error {
	fileId := c.Query("id")
	if fileId == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "无效的文件编号"})
	}

	data, err := os.ReadFile(FileMap)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "读取文件映射失败"})
	}

	var fileMap map[string]*UploadFile
	if err := json.Unmarshal(data, &fileMap); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "解析文件映射失败"})
	}

	fileInfo, ok := fileMap[fileId]
	if !ok {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "文件不存在"})
	}

	filePath := filepath.Join(UploadDir, fileInfo.Path)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "物理文件不存在"})
	}

	c.Attachment(filePath, fileInfo.Original)
	return c.SendFile(filePath)
}
