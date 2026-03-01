package api

import (
    "github.com/gofiber/fiber/v2"
    "go-fiber-web/model"
    "go-fiber-web/utils"
)

// 首页测试接口
func Index(c *fiber.Ctx) error {
    return utils.Success(c, "Go+Fiber 后端服务运行中")
}

// 获取用户列表
func GetUserList(c *fiber.Ctx) error {
    var list []model.User
    model.DB.Find(&list)
    return utils.Success(c, list)
}

// 创建用户
func CreateUser(c *fiber.Ctx) error {
    var user model.User
    if err := c.BodyParser(&user); err != nil {
        return utils.Fail(c, "参数错误")
    }
    model.DB.Create(&user)
    return utils.Success(c, user)
}
