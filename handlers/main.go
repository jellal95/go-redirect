package handlers

import (
	"go-redirect/utils"
	"log"

	"github.com/gofiber/fiber/v2"
)

func MainHandler(c *fiber.Ctx) error {
	products, err := utils.LoadProductsCSV("config/config.csv")
	if err != nil {
		log.Println("failed to load CSV products:", err)
	}

	if len(products) == 0 {
		return c.Status(404).SendString("No products configured")
	}

	return c.Render("main", products)
}
