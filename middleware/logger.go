package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RequestLogger creates a simple console logger middleware for monitoring requests
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		method := c.Method()
		path := c.Path()
		ip := c.IP()
		userAgent := c.Get("User-Agent")
		referer := c.Get("Referer")

		// Log request start with referer
		if referer != "" {
			fmt.Printf("[%s] %s %s from %s | Ref: %.30s | UA: %.40s\n",
				start.Format("15:04:05"), method, path, ip, referer, userAgent)
		} else {
			fmt.Printf("[%s] %s %s from %s | Direct | UA: %.50s\n",
				start.Format("15:04:05"), method, path, ip, userAgent)
		}

		// Process request
		err := c.Next()

		// Log request end
		duration := time.Since(start)
		status := c.Response().StatusCode()

		fmt.Printf("[%s] %s %s -> %d (%v)\n",
			time.Now().Format("15:04:05"), method, path, status, duration)

		return err
	}
}
