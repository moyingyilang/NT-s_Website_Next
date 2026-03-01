package utils

import "github.com/gofiber/fiber/v2"

// 统一返回格式
func Success(c *fiber.Ctx, data interface{}) error {
    return c.JSON(fiber.Map{
        "code": 200,
        "msg":  "success",
        "data": data,
    })
}

func Fail(c *fiber.Ctx, msg string) error {
    return c.Status(400).JSON(fiber.Map{
        "code": 400,
        "msg":  msg,
        "data": nil,
    })
}
