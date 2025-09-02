package handlers

import "github.com/gofiber/fiber/v2"

func LogsHandler(c *fiber.Ctx) error {
	return c.JSON(Logs)
}
