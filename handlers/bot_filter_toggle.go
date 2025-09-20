package handlers

import (
	"go-redirect/middleware"

	"github.com/gofiber/fiber/v2"
)

// ToggleBotFilterHandler toggles bot filter on/off
func ToggleBotFilterHandler(c *fiber.Ctx) error {
	type ToggleRequest struct {
		Enabled bool `json:"enabled"`
	}

	var req ToggleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Set bot filter state
	middleware.SetBotFilterEnabled(req.Enabled)

	return c.JSON(fiber.Map{
		"success": true,
		"enabled": req.Enabled,
		"message": func() string {
			if req.Enabled {
				return "Bot filter enabled"
			}
			return "Bot filter disabled"
		}(),
	})
}

// BotFilterStatusHandler returns current bot filter status
func BotFilterStatusHandler(c *fiber.Ctx) error {
	enabled := middleware.IsBotFilterEnabled()

	return c.JSON(fiber.Map{
		"enabled": enabled,
		"status": func() string {
			if enabled {
				return "active"
			}
			return "inactive"
		}(),
	})
}
