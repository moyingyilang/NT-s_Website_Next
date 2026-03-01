package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/jwt"
	"go-fiber-web/config"
	"go-fiber-web/model"
	"go-fiber-web/api/auth"
	"go-fiber-web/api/chat"
	"go-fiber-web/api/upload"
	"go-fiber-web/api/announcement"
	"go-fiber-web/api/wiki"
	"path/filepath"
)

func main() {
	// 初始化配置
	config.InitDir()
	// 初始化数据库
	model.InitDB()

	// 创建Fiber实例
	app := fiber.New(fiber.Config{
		AppName: "NTC-Chat",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.JSON(fiber.Map{"code": 1, "msg": err.Error(), "data": nil})
		},
	})

	// 跨域中间件
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type,Authorization",
	}))

	// 静态文件服务（前端页面/资源）
	app.Static("/", "./public", fiber.StaticConfig{
		Index:         "index.html",
		CacheDuration: 0,
	})
	// 静态文件服务（存储的文件/头像）
	app.Static("/storage", "./storage", fiber.StaticConfig{
		CacheDuration: 0,
	})

	// 无需登录的路由
	noAuth := app.Group("/api")
	noAuth.Post("/auth/register", auth.Register)
	noAuth.Post("/auth/login", auth.Login)

	// 需要JWT登录的路由
	authGroup := app.Group("/api", jwt.New(jwt.Config{
		SigningKey: config.JWTSecret,
		ContextKey: "user", // 可根据自身逻辑解析user_id到Locals
	}))
	// 注册各模块路由
	authGroup.Group("/auth", auth.Routes)
	authGroup.Group("/chat", chat.Routes)
	authGroup.Group("/upload", upload.Routes)
	authGroup.Group("/announcement", announcement.Routes)
	authGroup.Group("/wiki", wiki.Routes)

	// 启动服务
	_ = app.Listen(config.ServerPort)
	
	// main.go 中登录验证后的路由分组
    authGroup := app.Group("/api", jwt.New(jwt.Config{
	    SigningKey: config.JWTSecret,
	    // 可选：自定义JWT解析器，将user_id存入Locals
	    TokenLookup: "header:Authorization",
	    TokenPrefix: "Bearer",
    }))
    // 注册所有模块的路由（关键：与各routes.go匹配）
    authGroup.Group("/auth").Use(auth.Routes)
    authGroup.Group("/chat").Use(chat.Routes)
    authGroup.Group("/upload").Use(upload.Routes)
    authGroup.Group("/announcement").Use(announcement.Routes)
    authGroup.Group("/wiki").Use(wiki.Routes)

    // 无需登录的开放路由（注册/登录）
    noAuth := app.Group("/api")
    noAuth.Post("/auth/register", auth.Register)
    noAuth.Post("/auth/login", auth.Login)
}
