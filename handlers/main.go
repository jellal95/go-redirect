package handlers

import (
	"go-redirect/utils"

	"github.com/gofiber/fiber/v2"
)

func MainHandler(c *fiber.Ctx) error {
	products, err := utils.LoadProductsCSV("config/config.csv")
	if err != nil {
		utils.LogInfo(utils.LogEntry{
			Type: "load_csv_error",
			Extra: map[string]interface{}{
				"file":  "config/config.csv",
				"error": err.Error(),
			},
		})
	}

	if len(products) == 0 {
		utils.LogInfo(utils.LogEntry{
			Type: "no_products_configured",
			Extra: map[string]interface{}{
				"file": "config/config.csv",
			},
		})
		return c.Status(404).SendString("No products configured")
	}

	utils.LogInfo(utils.LogEntry{
		Type: "products_loaded",
		Extra: map[string]interface{}{
			"count": len(products),
			"file":  "config/config.csv",
		},
	})

	return c.Render("main", products)
}
