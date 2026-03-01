package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go-fiber-web/config"
)

func Cors() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:  config.AllowOrigins,
		AllowMethods:  "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:  "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	})
}
