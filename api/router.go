package api

import (
	"github.com/gofiber/fiber/v2"
	"go-fiber-web/api/auth"
	"go-fiber-web/api/chat"
	"go-fiber-web/api/announcement"
	"go-fiber-web/api/upload"
	"go-fiber-web/api/wiki"
	"go-fiber-web/middleware"
)

// 注册所有路由
func RegisterRoutes(app *fiber.App) {
	api := app.Group("/api")

	// 认证路由（无需JWT）
	authGroup := api.Group("/auth")
	authGroup.Post("/register", auth.Register)
	authGroup.Post("/login", auth.Login)
	authGroup.Get("/userinfo", middleware.JWT(), auth.GetUserInfo)
	authGroup.Post("/update", middleware.JWT(), auth.UpdateUser)
	authGroup.Post("/upload-avatar", middleware.JWT(), auth.UploadAvatar)
	authGroup.Post("/change-pwd", middleware.JWT(), auth.ChangePassword)

	// 聊天/好友路由（需JWT）
	chatGroup := api.Group("/chat", middleware.JWT())
	chatGroup.Post("/friend/request", chat.SendFriendRequest)
	chatGroup.Post("/friend/accept", chat.AcceptFriendRequest)
	chatGroup.Post("/friend/reject", chat.RejectFriendRequest)
	chatGroup.Get("/friend/requests", chat.GetFriendRequests)
	chatGroup.Get("/friends", chat.GetFriends)
	chatGroup.Post("/message/send", chat.SendMessage)
	chatGroup.Get("/messages", chat.GetMessages)
	chatGroup.Post("/friend/delete", chat.DeleteFriend)
	chatGroup.Post("/upload-image", chat.UploadChatImage)

	// 压缩包上传路由（需JWT）
	uploadGroup := api.Group("/upload", middleware.JWT())
	uploadGroup.Post("/chunk", upload.UploadChunk)
	uploadGroup.Post("/merge", upload.MergeChunk)
	uploadGroup.Get("/search", upload.SearchByMD5)
	uploadGroup.Get("/download", upload.DownloadFile)

	// 公告路由（公开）
	announcementGroup := api.Group("/announcement")
	announcementGroup.Get("/list", announcement.GetAnnouncements)
	announcementGroup.Post("/publish", announcement.PublishAnnouncement) // 管理员接口

	// NTwiki路由（公开）
	wikiGroup := api.Group("/wiki")
	wikiGroup.Get("/docs", wiki.GetDocList)
	wikiGroup.Get("/doc", wiki.GetDocContent)
}
