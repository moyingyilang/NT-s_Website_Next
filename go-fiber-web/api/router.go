package api

import (
    "github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App) {
    api := app.Group("/api")

    api.Get("/", Index)
    api.Get("/user/list", GetUserList)
    api.Post("/user/create", CreateUser)
}
