package auth

import (
	"path/filepath"
	"strings"

	"github.com/fiber-go/fiber/v2"
	"go-fiber-web/service"
	"go-fiber-web/utils"
)

// 注册接口
func Register(c *fiber.Ctx) error {
	type req struct {
		Username string `form:"username"`
		Password string `form:"password"`
	}
	var r req
	if err := c.BodyParser(&r); err != nil {
		return utils.Fail(c, "参数错误")
	}

	if r.Username == "" || r.Password == "" {
		return utils.Fail(c, "用户名和密码不能为空")
	}

	user, err := service.Register(r.Username, r.Password)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, user)
}

// 登录接口
func Login(c *fiber.Ctx) error {
	type req struct {
		Username string `form:"username"`
		Password string `form:"password"`
	}
	var r req
	if err := c.BodyParser(&r); err != nil {
		return utils.Fail(c, "参数错误")
	}

	token, user, err := service.Login(r.Username, r.Password)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	// 设置Token到响应头
	c.Set("Authorization", "Bearer "+token)
	return utils.Success(c, fiber.Map{
		"token": token,
		"user":  user,
	})
}

// 获取当前用户信息
func GetUserInfo(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	user, err := service.GetUserByID(userID)
	if err != nil {
		return utils.Fail(c, err.Error())
	}
	return utils.Success(c, user)
}

// 更新用户信息（昵称/简介/验证方式）
func UpdateUser(c *fiber.Ctx) error {
	type req struct {
		Nickname   string `form:"nickname"`
		Bio        string `form:"bio"`
		VerifyMode string `form:"verify_mode"`
	}
	var r req
	if err := c.BodyParser(&r); err != nil {
		return utils.Fail(c, "参数错误")
	}

	userID := c.Locals("user_id").(string)
	user, err := service.UpdateUserInfo(userID, r.Nickname, r.Bio, r.VerifyMode)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, user)
}

// 上传头像
func UploadAvatar(c *fiber.Ctx) error {
	file, err := c.FormFile("avatar")
	if err != nil {
		return utils.Fail(c, "未获取到头像文件")
	}

	// 校验文件类型
	ext := filepath.Ext(file.Filename)
	allowedExt := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	valid := false
	for _, e := range allowedExt {
		if strings.ToLower(ext) == e {
			valid = true
			break
		}
	}
	if !valid {
		return utils.Fail(c, "只允许上传JPG、PNG、GIF、WEBP格式")
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		return utils.Fail(c, "文件打开失败")
	}
	defer src.Close()

	// 保存文件
	filename := utils.FileMD5(src) + ext
	path, err := utils.SaveAvatar(src, filename)
	if err != nil {
		return utils.Fail(c, "头像保存失败")
	}

	// 更新用户头像
	userID := c.Locals("user_id").(string)
	if err := service.UpdateAvatar(userID, path); err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, fiber.Map{"avatar_path": path})
}

// 修改密码
func ChangePassword(c *fiber.Ctx) error {
	type req struct {
		OldPwd string `form:"old_password"`
		NewPwd string `form:"new_password"`
	}
	var r req
	if err := c.BodyParser(&r); err != nil {
		return utils.Fail(c, "参数错误")
	}

	if r.OldPwd == "" || r.NewPwd == "" {
		return utils.Fail(c, "旧密码和新密码不能为空")
	}

	userID := c.Locals("user_id").(string)
	if err := service.UpdatePassword(userID, r.OldPwd, r.NewPwd); err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, nil)
}
