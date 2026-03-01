package chat

import (
	"github.com/gofiber/fiber/v2"
)

// Routes 注册聊天/好友相关路由
// 挂载路径：/api/chat
func Routes(router fiber.Router) {
	// 获取好友列表
	router.Get("/friends", GetFriends)
	// 获取与指定好友的聊天记录
	router.Get("/messages", GetMessages)
	// 发送聊天消息
	router.Post("/send-message", SendMessage)
	// 发送好友申请
	router.Post("/add-friend", AddFriend)
	// 获取好友申请列表
	router.Get("/friend-requests", GetFriendRequests)
	// 处理好友申请（同意/拒绝）
	router.Post("/handle-request", HandleFriendRequest)
	// 删除好友
	router.Post("/delete-friend", DeleteFriend)
}
