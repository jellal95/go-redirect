package handlers

import (
	"go-redirect/models"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
)

func PreSaleHandler(c *fiber.Ctx) error {
	if len(Products) == 0 {
		return c.Status(404).SendString("No products configured")
	}

	rand.Seed(time.Now().UnixNano())
	total := 0.0
	for _, p := range Products {
		total += p.Percentage
	}
	r := rand.Float64() * total
	sum := 0.0
	var selected models.Product
	for _, p := range Products {
		sum += p.Percentage
		if r <= sum {
			selected = p
			break
		}
	}

	// Render view pre-sale.html dengan data produk
	return c.Render("pre-sale", fiber.Map{
		"ID":    selected.ID,
		"Name":  selected.Name,
		"Image": selected.Image,
	})
}
