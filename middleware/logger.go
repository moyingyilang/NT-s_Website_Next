package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)
		fmt.Printf("[%s] %s %s - %dms\n",
			time.Now().Format("2006-01-02 15:04:05"),
			c.Method(),
			c.Path(),
			duration.Milliseconds(),
		)
		return err
	}
}
