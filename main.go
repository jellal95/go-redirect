package main

import (
	"log"
	"os"

	"go-redirect/geo"
	"go-redirect/handlers"
	"go-redirect/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	_, err := utils.InitDB()
	if err != nil {
		panic(err)
	}

	if err := utils.Migrate(); err != nil {
		panic(err)
	}

	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// Load config
	cfg, err := utils.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	handlers.Products = cfg.Products
	handlers.PropellerConfig = cfg.Propeller
	handlers.GalaksionConfig = cfg.Galaksion
	handlers.PopcashConfig = cfg.Popcash

	// Init Geo
	if err := geo.InitGeoDB("GeoLite2-City.mmdb"); err != nil {
		log.Println("GeoIP DB not found, skipping geo info")
	}

	app.Get("/", handlers.RedirectHandler)
	app.Get("/logs", handlers.LogsHandler)

	// new: endpoint postback
	app.Get("/postback", handlers.PostbackHandler)
	app.Get("/postbacks", handlers.GetPostbacks)

	app.Get("/pre-sale", handlers.PreSaleHandler)
	app.Get("/article", handlers.ArticleHandler)
	app.Get("/main", handlers.MainHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Server running on :" + port)
	err = app.Listen(":" + port)
	if err != nil {
		return
	}
}
