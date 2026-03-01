package auth

import (
	"github.com/gofiber/fiber/v2"
)

// Routes 注册用户认证/信息相关路由
// 挂载路径：/api/auth
func Routes(router fiber.Router) {
	// 获取用户信息
	router.Get("/userinfo", UserInfo)
	// 退出登录
	router.Post("/logout", Logout)
	// 更新用户信息（昵称/个性签名等）
	router.Post("/update", UpdateUser)
	// 上传用户头像
	router.Post("/upload-avatar", UploadAvatar)
	// 修改密码
	router.Post("/change-pwd", ChangePwd)
}
