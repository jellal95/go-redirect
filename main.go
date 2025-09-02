package main

import (
	"log"
	"os"

	"go-redirect/geo"
	"go-redirect/handlers"
	"go-redirect/utils"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	// Load config
	products, err := utils.LoadProducts("config/links.json")
	if err != nil {
		log.Fatal(err)
	}
	handlers.Products = products

	// Init Geo
	if err := geo.InitGeoDB("GeoLite2-City.mmdb"); err != nil {
		log.Println("GeoIP DB not found, skipping geo info")
	}

	app.Get("/", handlers.RedirectHandler)
	app.Get("/logs", handlers.LogsHandler)

	// new: endpoint postback
	app.Get("/postback", handlers.PostbackHandler)
	app.Get("/postbacks", handlers.GetPostbacks)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Server running on :" + port)
	app.Listen(":" + port)
}
