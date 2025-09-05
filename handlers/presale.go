package handlers

import (
	"go-redirect/models"
	"math/rand/v2"

	"github.com/gofiber/fiber/v2"
)

func PreSaleHandler(c *fiber.Ctx) error {
	if len(Products) == 0 {
		return c.Status(404).SendString("No products configured")
	}

	total := 0.0
	for _, p := range Products {
		total += p.Percentage
	}

	var selected models.Product
	if total <= 0 {
		selected = Products[0]
	} else {
		r := rand.Float64() * total
		sum := 0.0
		for _, p := range Products {
			sum += p.Percentage
			if r <= sum {
				selected = p
				break
			}
		}
		if selected.ID == "" {
			selected = Products[len(Products)-1]
		}
	}

	return c.Render("pre-sale", fiber.Map{
		"ID":    selected.ID,
		"Name":  selected.Name,
		"Image": selected.Image,
	})
}
