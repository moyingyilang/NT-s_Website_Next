package main

import (
    "github.com/gofiber/fiber/v2"
    "go-fiber-web/api"
    "go-fiber-web/config"
    "go-fiber-web/middleware"
    "go-fiber-web/model"
)

func main() {
    // 初始化数据库
    model.InitDB()

    // 创建 Fiber 实例
    app := fiber.New(fiber.Config{
        AppName: "Go-Fiber-Web",
    })

    // 全局中间件
    app.Use(middleware.Cors())

    // 静态文件服务（前端）
    app.Static("/", config.StaticDir)

    // 注册 API 路由
    api.RegisterRoutes(app)

    // 启动服务
    app.Listen(config.Port)
}
