package wiki

import (
	"github.com/gofiber/fiber/v2"
)

// Routes 注册Wiki文档相关路由
// 挂载路径：/api/wiki
func Routes(router fiber.Router) {
	// 获取Wiki文档列表
	router.Get("/list", GetDocList)
	// 获取单篇文档内容
	router.Get("/content", GetDocContent)
	// 上传/编辑Wiki文档
	router.Post("/upload", UploadDoc)
	// 删除Wiki文档
	router.Post("/delete", DeleteDoc)
}
