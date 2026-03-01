package announcement

import (
	"github.com/gofiber/fiber/v2"
)

// Routes 注册系统公告相关路由
// 挂载路径：/api/announcement
func Routes(router fiber.Router) {
	// 发布新公告
	router.Post("/publish", PublishAnnouncement)
	// 获取公开公告列表
	router.Get("/list", GetAnnouncements)
	// 获取单条公告详情
	router.Get("/detail", GetAnnouncementByID)
	// 更新公告信息
	router.Post("/update", UpdateAnnouncement)
	// 删除公告
	router.Post("/delete", DeleteAnnouncement)
}
