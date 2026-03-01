package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/jwt/v3"
	"go-fiber-web/config"
	"go-fiber-web/utils"
)

func JWT() fiber.Handler {
	return jwt.New(jwt.Config{
		SigningKey:   []byte(config.JWTSecret),
		ErrorHandler: jwtErrorHandler,
		ContextKey:   "user_id", // 解析后存入上下文的key
	})
}

func jwtErrorHandler(c *fiber.Ctx, err error) error {
	if strings.Contains(err.Error(), "token is expired") {
		return utils.Fail(c, "登录已过期，请重新登录")
	}
	return utils.Fail(c, "未登录或无效token")
}
