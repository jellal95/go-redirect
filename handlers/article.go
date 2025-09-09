package handlers

import "github.com/gofiber/fiber/v2"

func ArticleHandler(c *fiber.Ctx) error {
	return c.Render("article-1", fiber.Map{})
}
