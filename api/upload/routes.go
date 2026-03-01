package upload

import (
	"github.com/gofiber/fiber/v2"
)

// Routes 注册文件分块上传相关路由
// 挂载路径：/api/upload
func Routes(router fiber.Router) {
	// 上传文件分块
	router.Post("/chunk", UploadChunk)
	// 合并文件分块
	router.Post("/merge", MergeChunk)
	// 根据MD5查询文件是否已存在（避免重复上传）
	router.Get("/check-md5", SearchByMD5)
	// 下载已上传的文件
	router.Get("/download", DownloadFile)
	// 获取当前用户的所有上传文件列表
	router.Get("/list", ListUserFiles)
	// 删除已上传的文件
	router.Post("/delete", DeleteFile)
}
