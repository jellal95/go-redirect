package handlers

import (
	"log"
	"math/rand/v2"

	"go-redirect/models"
	"go-redirect/utils"

	"github.com/gofiber/fiber/v2"
)

func PreSaleHandler(c *fiber.Ctx) error {
	// Always load from CSV as per requirement
	products, err := utils.LoadProductsCSV("config/config.csv")
	if err != nil {
		log.Println("failed to load CSV products:", err)
	}
	if len(products) == 0 {
		return c.Status(404).SendString("No products configured")
	}

	total := 0.0
	for _, p := range products {
		total += p.Percentage
	}

	var selected models.Product
	if total <= 0 {
		selected = products[0]
	} else {
		r := rand.Float64() * total
		sum := 0.0
		for _, p := range products {
			sum += p.Percentage
			if r <= sum {
				selected = p
				break
			}
		}
		if selected.ID == "" {
			selected = products[len(products)-1]
		}
	}

	// Ensure Name has a default if not provided
	name := selected.Name

	return c.Render("pre-sale", fiber.Map{
		"ID":           selected.ID,
		"Name":         name,
		"Description":  selected.Description,
		"Image":        selected.Image,
		"Komisi":       selected.Komisi,
		"KomisiHingga": selected.KomisiHingga,
	})
}
