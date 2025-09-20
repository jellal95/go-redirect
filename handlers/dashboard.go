package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// DashboardHandler serves the analytics dashboard
func DashboardHandler(c *fiber.Ctx) error {
	return c.Render("dashboard", fiber.Map{
		"Title": "Go-Redirect Analytics Dashboard",
	})
}
