package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// Token黑名单：存储已失效的Token，解决Logout后Token仍可使用的问题
var tokenBlacklist = make(map[string]bool)
// 互斥锁：防止并发读写Token黑名单导致的数据竞争
var mutex sync.Mutex

func main() {
	// 初始化数据库连接（外部实现，如GORM初始化）
	InitDB()

	// 初始化Fiber应用，禁用预解析多部分表单
	app := fiber.New(fiber.Config{
		DisablePreParseMultipartForm: true,
	})

	// 日志中间件：记录请求时间、方法、路径、状态码、IP
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${method} ${path} | status: ${status} | ip: ${ip}\n",
	}))

	// CORS跨域中间件：允许所有来源和常用请求方法
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
	}))

	// 禁用缓存中间件：防止浏览器缓存未登录页面，导致"返回掉登"假象
	app.Use(func(c *fiber.Ctx) error {
		// 登录/注册接口放行，无需禁用缓存
		if strings.HasPrefix(c.Path(), "/api/public/") || c.Path() == "/login" {
			return c.Next()
		}
		// 设置响应头禁用缓存
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.Next()
	})

	// 静态资源映射：前端页面、数据文件、NTWiki相关资源
	app.Static("/", "./static")
	app.Static("/data", "./data")
	app.Static("/ntwiki", "./static/ntwiki")

	// 业务API路由注册：认证、用户、好友、消息、公告、文件上传等
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

	// NTwiki文档列表接口：返回docs目录下所有MD文件列表（按修改时间倒序）
	app.Get("/api/ntwiki/docs-list", func(c *fiber.Ctx) error {
		docDir := "./static/ntwiki/docs/"
		var docs []map[string]interface{}

		// 校验目录路径合法性
		if !fs.ValidPath(docDir) {
			return c.JSON(docs)
		}

		// 遍历目录，筛选MD文件（排除目录和隐藏文件）
		err := filepath.WalkDir(docDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				basename := strings.TrimSuffix(d.Name(), ".md")
				if basename != "" && basename[0] != '.' {
					fileInfo, _ := d.Info()
					docs = append(docs, map[string]interface{}{
						"name":    basename,   // 文档名称（无后缀）
						"display": basename,   // 显示名称
						"mtime":   fileInfo.ModTime().Unix(), // 修改时间戳
					})
				}
			}
			return nil
		})

		// 遍历出错时仍返回空列表，避免服务异常
		if err != nil {
			return c.JSON(docs)
		}

		// 按修改时间倒序排序
		for i := 0; i < len(docs); i++ {
			for j := i + 1; j < len(docs); j++ {
				if docs[i]["mtime"].(int64) < docs[j]["mtime"].(int64) {
					docs[i], docs[j] = docs[j], docs[i]
				}
			}
		}

		return c.JSON(docs)
	})

	// NTwiki文档内容接口：支持MD文件渲染为HTML
	app.Get("/ntwiki", func(c *fiber.Ctx) error {
		action := c.Query("action")
		docName := c.Query("doc")

		// 非文档获取请求，返回NTWiki首页
		if action != "get" || docName == "" {
			return c.SendFile("./static/ntwiki/index.html")
		}

		// 校验文档名合法性：仅允许字母、数字、_-./
		allowChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-./"
		if !strings.ContainsAny(docName, allowChars) {
			c.Status(400).SendString(`<div class="error">无效的文档名称</div>`)
			return nil
		}

		// 拼接文件路径，防止路径穿越攻击
		filename := docName + ".md"
		filePath := "./static/ntwiki/docs/" + filename
		realBase, _ := filepath.Abs("./static/ntwiki/docs/")
		realFile, _ := filepath.Abs(filePath)
		if !strings.HasPrefix(realFile, realBase) || !fs.ValidPath(realFile) {
			c.Status(404).SendString(`<div class="error">文档未找到</div>`)
			return nil
		}

		// 读取MD文件内容
		content, err := os.ReadFile(realFile)
		if err != nil {
			c.Status(500).SendString(`<div class="error">无法读取文档</div>`)
			return nil
		}

		// MD转HTML：配置扩展（自动生成标题ID、支持常用语法）和渲染器（新标签打开链接）
		extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
		p := parser.NewWithExtensions(extensions) // 避免变量名与parser包冲突
		rendererOpts := html.RendererOptions{
			Flags: html.CommonFlags | html.HrefTargetBlank,
		}
		renderer := html.NewRenderer(rendererOpts)
		bodyHtml := markdown.ToHTML(content, p, renderer)

		// 拼接最后编辑时间页脚
		fileInfo, _ := os.Stat(realFile)
		mtime := fileInfo.ModTime().Format("2006-01-02 15:04:05")
		footerHtml := `<div class="doc-footer-time" style="margin-top:20px;padding-top:10px;border-top:1px solid #eee;color:#999;">📅 最后编辑：` + mtime + `</div>`

		// 返回HTML内容（[]byte转string解决类型不匹配）
		c.Type("text/html; charset=utf-8")
		return c.SendString(string(bodyHtml) + footerHtml)
	})

	// 启动服务，监听8080端口
	log.Println("服务器启动在 :8080")
	log.Fatal(app.Listen(":8080"))
}

// RegisterHandler：用户注册接口
func RegisterHandler(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	// 参数校验：用户名和密码不能为空
	if username == "" || password == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "用户名和密码不能为空"})
	}

	// 校验用户名是否已存在
	if _, err := GetUserByUsername(username); err == nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "用户名已存在"})
	}

	// 构造用户对象（密码加密、默认昵称等）
	user := &User{
		ID:         GenerateUserID(),
		Username:   username,
		Password:   HashPassword(password), // 复用util.go的加密方法
		Nickname:   username,
		VerifyMode: "need_verify",
		Registered: time.Now().Unix(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 处理头像上传（可选）
	file, err := c.FormFile("avatar")
	if err == nil {
		mime := file.Header.Get("Content-Type")
		// 校验图片类型和大小
		if !IsAllowedImageType(mime) {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "只允许上传JPG、PNG、GIF、WEBP格式的图片"})
		}
		if file.Size > 2*1024*1024 {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "图片不能超过2MB"})
		}
		// 生成唯一文件名并保存
		ext := GetFileExt(file.Filename)
		filename := uuid.NewString() + "." + ext
		savePath := filepath.Join(AvatarDir, filename)
		if err := c.SaveFile(file, savePath); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "error": "头像保存失败"})
		}
		user.Avatar = "/data/avatars/" + filename
	}

	// 保存用户到数据库
	if err := CreateUser(user); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "注册失败"})
	}

	return c.JSON(fiber.Map{"success": true, "user": user})
}

// LoginHandler：用户登录接口
func LoginHandler(c *fiber.Ctx) error {
	usernameOrId := c.FormValue("username")
	password := c.FormValue("password")

	// 参数校验：用户名/ID和密码不能为空
	if usernameOrId == "" || password == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "请输入用户名/ID和密码"})
	}

	// 查找用户（支持用户名或ID登录）
	user, err := GetUserByUsername(usernameOrId)
	if err != nil {
		user, err = GetUserByID(usernameOrId)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"success": false, "error": "用户名/ID或密码错误"})
		}
	}

	// 校验密码
	if !CheckPassword(password, user.Password) {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "用户名/ID或密码错误"})
	}

	// 生成JWT令牌（复用util.go的方法）
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

// LogoutHandler：用户退出接口（核心：将Token加入黑名单）
func LogoutHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token != "" {
		// 处理Bearer前缀（前端可能传递Bearer Token格式）
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}
		// 加锁操作黑名单，防止并发问题
		mutex.Lock()
		tokenBlacklist[token] = true // 标记Token失效
		mutex.Unlock()
	}

	return c.JSON(fiber.Map{"success": true, "msg": "退出成功，请清除本地Token"})
}

// UserInfoHandler：获取用户信息接口（需Token验证）
func UserInfoHandler(c *fiber.Ctx) error {
	// 从请求头获取Token
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 验证Token有效性（复用util.go的方法，已集成黑名单校验）
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "令牌无效"})
	}

	// 查找用户并返回信息
	user, err := GetUserByID(claims.UserID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "用户不存在"})
	}

	return c.JSON(fiber.Map{"success": true, "user": user})
}

// UpdateUserHandler：更新用户信息接口
func UpdateUserHandler(c *fiber.Ctx) error {
	// Token验证
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 查找当前用户
	user, err := GetUserByID(claims.UserID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "用户不存在"})
	}

	// 处理密码修改（需校验旧密码）
	oldPwd := c.FormValue("old_password")
	newPwd := c.FormValue("password")
	if oldPwd != "" && newPwd != "" {
		if !CheckPassword(oldPwd, user.Password) {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "旧密码错误"})
		}
		user.Password = HashPassword(newPwd)
	}

	// 更新其他信息（昵称、个性签名、好友验证模式）
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

	// 保存更新
	if err := UpdateUser(user); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "更新失败"})
	}

	return c.JSON(fiber.Map{"success": true, "user": user})
}

// UploadAvatarHandler：上传头像接口
func UploadAvatarHandler(c *fiber.Ctx) error {
	// Token验证
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 校验文件是否上传
	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "未选择文件"})
	}

	// 校验文件大小和类型
	if file.Size > 2*1024*1024 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "图片不能超过2MB"})
	}
	mime := file.Header.Get("Content-Type")
	if !IsAllowedImageType(mime) {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "只允许上传JPG、PNG、GIF、WEBP格式的图片"})
	}

	// 生成唯一文件名并保存
	ext := GetFileExt(file.Filename)
	filename := uuid.NewString() + "." + ext
	savePath := filepath.Join(AvatarDir, filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "上传失败"})
	}

	// 更新用户头像地址
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

// SearchUserHandler：搜索用户接口（按用户ID）
func SearchUserHandler(c *fiber.Ctx) error {
	// Token验证
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 参数校验：用户ID不能为空
	userId := c.Query("userId")
	if userId == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "请输入用户ID"})
	}

	// 禁止搜索自己
	if userId == claims.UserID {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "不能搜索自己"})
	}

	// 查找用户并返回精简信息（ID、用户名、昵称）
	user, err := GetUserByID(userId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "用户不存在"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user":    fiber.Map{"id": user.ID, "username": user.Username, "nickname": user.Nickname},
	})
}

// FriendListHandler：获取好友列表接口
func FriendListHandler(c *fiber.Ctx) error {
	// Token验证
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 查询好友列表
	friends, err := GetFriends(claims.UserID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "获取好友列表失败"})
	}

	return c.JSON(fiber.Map{"success": true, "friends": friends})
}

// SendFriendRequestHandler：发送好友请求接口
func SendFriendRequestHandler(c *fiber.Ctx) error {
	// Token验证
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 参数校验：目标用户ID不能为空
	targetID := c.FormValue("targetId")
	if targetID == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "目标用户ID不能为空"})
	}

	// 禁止添加自己为好友
	if targetID == claims.UserID {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "不能添加自己为好友"})
	}

	// 校验目标用户是否存在
	targetUser, err := GetUserByID(targetID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "目标用户不存在"})
	}

	// 校验对方好友验证模式（禁止所有人则直接返回）
	if targetUser.VerifyMode == "deny_all" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "对方禁止添加好友"})
	}

	// 对方允许所有人添加：直接创建双向好友关系（事务保证原子性）
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
		if err := tx.Commit().Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "error": "添加好友失败"})
		}
		return c.JSON(fiber.Map{"success": true, "message": "添加好友成功"})
	}

	// 对方需要验证：发送好友请求
	if err := SendFriendRequest(claims.UserID, targetID); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "好友请求已发送"})
}

// AcceptFriendRequestHandler：接受好友请求接口
func AcceptFriendRequestHandler(c *fiber.Ctx) error {
	// Token验证
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 接受请求（创建双向好友关系）
	requesterID := c.FormValue("requesterId")
	if err := AcceptFriendRequest(requesterID, claims.UserID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "接受请求失败"})
	}

	return c.JSON(fiber.Map{"success": true})
}

// RejectFriendRequestHandler：拒绝好友请求接口
func RejectFriendRequestHandler(c *fiber.Ctx) error {
	// Token验证
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 删除待处理的好友请求记录
	requesterID := c.FormValue("requesterId")
	if err := DB.Where("user_id = ? AND friend_id = ? AND status = ?", claims.UserID, requesterID, StatusPending).Delete(&Friend{}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "拒绝失败"})
	}

	return c.JSON(fiber.Map{"success": true})
}

// DeleteFriendHandler：删除好友接口（事务保证：删除双向好友关系+聊天记录）
func DeleteFriendHandler(c *fiber.Ctx) error {
	// Token验证
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 参数校验：好友ID不能为空
	friendID := c.FormValue("friendId")
	if friendID == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "缺少好友ID"})
	}

	tx := DB.Begin()
	// 删除当前用户的好友记录
	if err := tx.Where("user_id = ? AND friend_id = ? AND status = ?", claims.UserID, friendID, StatusAccepted).Delete(&Friend{}).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "删除失败"})
	}
	// 删除对方的好友记录
	if err := tx.Where("user_id = ? AND friend_id = ? AND status = ?", friendID, claims.UserID, StatusAccepted).Delete(&Friend{}).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "删除失败"})
	}
	// 删除双方聊天记录
	if err := tx.Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)", claims.UserID, friendID, friendID, claims.UserID).Delete(&Message{}).Error; err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "删除消息失败"})
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "删除失败"})
	}

	return c.JSON(fiber.Map{"success": true})
}

// SendMessageHandler：发送消息接口（仅好友可发送）
func SendMessageHandler(c *fiber.Ctx) error {
	// Token验证
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 参数校验：好友ID和消息内容不能为空
	friendID := c.FormValue("friendId")
	content := c.FormValue("content")
	msgType := c.FormValue("type", MessageTypeText)
	if friendID == "" || content == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "参数不足"})
	}

	// 校验好友关系
	var isFriend bool
	if err := DB.Where("user_id = ? AND friend_id = ? AND status = ?", claims.UserID, friendID, StatusAccepted).First(&Friend{}).Error; err == nil {
		isFriend = true
	}
	if !isFriend {
		return c.Status(403).JSON(fiber.Map{"success": false, "error": "不是好友关系"})
	}

	// 构造并保存消息
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

// MessageListHandler：获取好友聊天记录接口
func MessageListHandler(c *fiber.Ctx) error {
	// Token验证
	token := c.Get("Authorization")
	claims, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 参数校验：好友ID不能为空
	friendID := c.Query("friendId")
	if friendID == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "缺少好友ID"})
	}

	// 查询聊天记录
	messages, err := GetMessages(claims.UserID, friendID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "获取消息失败"})
	}

	return c.JSON(fiber.Map{"success": true, "messages": messages})
}

// UploadImageHandler：消息图片上传接口
func UploadImageHandler(c *fiber.Ctx) error {
	// Token验证
	token := c.Get("Authorization")
	_, err := VerifyToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "未登录"})
	}

	// 校验文件是否上传
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "未选择文件"})
	}

	// 校验文件大小和类型
	if file.Size > 10*1024*1024 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "图片不能超过10MB"})
	}
	mime := file.Header.Get("Content-Type")
	if !IsAllowedImageType(mime) {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "只允许上传图片"})
	}

	// 计算文件MD5（用于唯一标识）
	md5Str, err := CalculateMD5FromFile(file)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "计算MD5失败"})
	}

	// 保存文件
	savePath := filepath.Join(UploadDir, md5Str)
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "上传失败"})
	}

	// 更新文件映射（记录原始文件名、MD5、MIME类型）
	fileMapData, _ := os.ReadFile(FileMap)
	var fileMap []map[string]string
	_ = json.Unmarshal(fileMapData, &fileMap)
	fileMap = append(fileMap, map[string]string{
		"original": file.Filename,
		"md5":      md5Str,
		"mime":     mime,
	})
	fileMapJson, err := json.Marshal(fileMap)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "保存文件映射失败"})
	}
	_ = os.WriteFile(FileMap, fileMapJson, 0644)

	return c.JSON(fiber.Map{"success": true, "fileId": md5Str})
}

// AnnouncementListHandler：获取公告列表接口
func AnnouncementListHandler(c *fiber.Ctx) error {
	// 查询可见公告
	anns, err := GetVisibleAnnouncements()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "获取公告失败"})
	}

	return c.JSON(fiber.Map{"success": true, "announcements": anns})
}

// CreateAnnouncementHandler：创建公告接口（需admin=yes权限）
func CreateAnnouncementHandler(c *fiber.Ctx) error {
	// 权限校验：仅管理员可创建
	if c.Query("admin") != "yes" {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "页面不存在"})
	}

	// 参数校验：公告标题不能为空
	title := c.FormValue("title")
	summary := c.FormValue("summary")
	visible := c.FormValue("visible") == "on"
	if title == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "公告标题不能为空"})
	}

	// 处理标签（JSON字符串转切片，解析失败则为空切片）
	var tags []string
	if err := json.Unmarshal([]byte(c.FormValue("tags")), &tags); err != nil {
		tags = []string{}
	}

	// 构造并保存公告
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

// UploadChunkHandler：分块上传接口（处理大文件分片上传）
func UploadChunkHandler(c *fiber.Ctx) error {
	// 解析分块上传参数
	fileId := c.FormValue("fileId")
	totalChunks, _ := strconv.Atoi(c.FormValue("totalChunks"))
	chunkIndex, _ := strconv.Atoi(c.FormValue("chunkIndex"))
	fileName := c.FormValue("fileName")
	fileSize, _ := strconv.ParseInt(c.FormValue("fileSize"), 10, 64)
	chunkSize, _ := strconv.ParseInt(c.FormValue("chunkSize"), 10, 64)

	// 参数合法性校验
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

	// 校验文件扩展名
	ext := GetFileExt(fileName)
	if !IsAllowedExtension(ext) {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "不支持的文件类型，仅允许：" + strings.Join(AllowedExtensions, ", ")})
	}

	// 接收分块文件
	file, err := c.FormFile("chunkFile")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "分块文件上传失败"})
	}
	if file.Size > chunkSize {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "上传的块大小超过声明值"})
	}

	// 保存分块文件到临时目录
	chunkPath := filepath.Join(TempDir, fmt.Sprintf("%s_chunk_%d.part", fileId, chunkIndex))
	if err := c.SaveFile(file, chunkPath); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "保存分块失败"})
	}

	// 记录上传进度（创建/更新上传信息JSON）
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
	infoJson, err := json.Marshal(info)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "保存上传信息失败"})
	}
	_ = os.WriteFile(infoPath, infoJson, 0644)

	return c.JSON(fiber.Map{"success": true, "chunkIndex": chunkIndex})
}

// MergeChunkHandler：分块合并接口（所有分片上传完成后合并为完整文件）
func MergeChunkHandler(c *fiber.Ctx) error {
	// 解析合并参数
	fileId := c.FormValue("fileId")
	fileName := c.FormValue("fileName")
	totalChunks, _ := strconv.Atoi(c.FormValue("totalChunks"))
	if fileName == "" || totalChunks <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "参数不足"})
	}

	// 校验上传信息是否存在
	infoPath := filepath.Join(TempDir, fmt.Sprintf("%s_info.json", fileId))
	if _, err := os.Stat(infoPath); err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "未找到上传信息"})
	}

	// 校验分块是否完整
	data, _ := os.ReadFile(infoPath)
	var info map[string]interface{}
	_ = json.Unmarshal(data, &info)
	uploadedChunks := info["uploadedChunks"].([]int)
	if len(uploadedChunks) != totalChunks {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "分块不完整，无法合并"})
	}

	// 生成最终文件名并创建文件
	ext := GetFileExt(fileName)
	uniqueId := GenerateUniqueId()
	finalName := uniqueId + "." + ext
	finalPath := filepath.Join(UploadDir, finalName)
	finalFile, err := os.Create(finalPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "无法创建最终文件"})
	}
	defer finalFile.Close()

	// 按顺序合并所有分块
	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(TempDir, fmt.Sprintf("%s_chunk_%d.part", fileId, i))
		if _, err := os.Stat(chunkPath); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": fmt.Sprintf("缺少分块 %d", i)})
		}
		chunkFile, _ := os.Open(chunkPath)
		_, _ = io.Copy(finalFile, chunkFile)
		_ = chunkFile.Close()
		_ = os.Remove(chunkPath) // 合并后删除临时分块
	}

	// 计算文件MD5并保存文件映射
	md5Str, _ := CalculateMD5(finalPath)
	fileInfo, _ := finalFile.Stat()
	_ = SaveFileMapping(uniqueId, &UploadFile{
		Original: fileName,
		Path:     finalName,
		Size:     fileInfo.Size(),
		MD5:      md5Str,
		Time:     time.Now().Unix(),
	})

	// 清理临时上传信息文件
	_ = os.Remove(infoPath)

	return c.JSON(fiber.Map{
		"success":     true,
		"fileId":      uniqueId,
		"downloadUrl": fmt.Sprintf("/upF?action=download&id=%s", uniqueId),
		"md5":         md5Str,
	})
}

// UploadStatusHandler：查询分块上传进度接口
func UploadStatusHandler(c *fiber.Ctx) error {
	// 参数校验：fileId不能为空
	fileId := c.Query("fileId")
	if fileId == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "缺少fileId参数"})
	}

	// 查询上传信息
	infoPath := filepath.Join(TempDir, fmt.Sprintf("%s_info.json", fileId))
	if _, err := os.Stat(infoPath); err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "未找到上传信息"})
	}
	data, _ := os.ReadFile(infoPath)
	var info map[string]interface{}
	_ = json.Unmarshal(data, &info)

	return c.JSON(fiber.Map{"success": true, "uploadedChunks": info["uploadedChunks"].([]int)})
}

// SearchFileHandler：按MD5搜索文件接口
func SearchFileHandler(c *fiber.Ctx) error {
	// 参数校验：MD5不能为空
	md5 := c.Query("md5")
	if md5 == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "缺少MD5参数"})
	}

	// 按MD5查询文件和下载链接
	files, urls := SearchFileByMD5(md5)
	return c.JSON(fiber.Map{"success": true, "files": files, "downloadUrls": urls})
}

// DownloadFileHandler：文件下载接口
func DownloadFileHandler(c *fiber.Ctx) error {
	// 参数校验：文件ID不能为空
	fileId := c.Query("id")
	if fileId == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "无效的文件编号"})
	}

	// 读取文件映射
	data, err := os.ReadFile(FileMap)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "读取文件映射失败"})
	}
	var fileMap map[string]*UploadFile
	if err := json.Unmarshal(data, &fileMap); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "解析文件映射失败"})
	}

	// 校验文件是否存在
	fileInfo, ok := fileMap[fileId]
	if !ok {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "文件不存在"})
	}
	filePath := filepath.Join(UploadDir, fileInfo.Path)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "物理文件不存在"})
	}

	// 触发文件下载（指定原始文件名）
	c.Attachment(filePath, fileInfo.Original)
	return c.SendFile(filePath)
}
