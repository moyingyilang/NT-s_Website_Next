package utils

import "github.com/gofiber/fiber/v2"

// 统一响应结构体
type Response struct {
	Code int         `json:"code"` // 0成功，其他失败
	Msg  string      `json:"msg"`  // 提示信息
	Data interface{} `json:"data"` // 响应数据
}

// Success 成功响应
func Success(c *fiber.Ctx, data interface{}) error {
	return c.JSON(Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

// Fail 失败响应
func Fail(c *fiber.Ctx, msg string) error {
	return c.JSON(Response{
		Code: 1,
		Msg:  msg,
		Data: nil,
	})
}

// JWT解析用户ID工具（可根据自身JWT逻辑调整）
func GetUserIDFromToken(c *fiber.Ctx) string {
	// 此处与你的JWT中间件配合，从Locals获取用户ID
	if userID, ok := c.Locals("user_id").(string); ok {
		return userID
	}
	return ""
}
