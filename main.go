package main

import (
	"go-redirect/middleware"
	"os"

	"go-redirect/handlers"
	"go-redirect/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func main() {
	//// Load .env file
	//if err := godotenv.Load(); err != nil {
	//	log.Println("No .env file found, using system environment")
	//}
	//
	//_, err := utils.InitDB()
	//if err != nil {
	//	panic(err)
	//}
	//
	//if err := utils.Migrate(); err != nil {
	//	panic(err)
	//}

	// ========== 1. Load Campaign Config ==========
	appCfg, err := utils.LoadConfig("config/config.yaml")
	if err != nil {
		utils.LogFatal(utils.LogEntry{
			Type:  "fatal_error",
			Extra: map[string]interface{}{"error": err.Error()},
		}, 1)
	}

	handlers.Products = appCfg.Products
	handlers.PropellerConfig = appCfg.Propeller
	handlers.GalaksionConfig = appCfg.Galaksion
	handlers.PopcashConfig = appCfg.Popcash

	// ========== 2. Setup Bot Filter Middleware ==========
	botCfg := middleware.BotFilterConfig{
		AllowCountries: []string{"ID"},
		BlacklistUA:    []string{"curl", "bot", "spider", "crawler", "python", "scrapy", "headless"},
		BlacklistIPPrefix: []string{
			"34.",      // Google Cloud
			"35.",      // Google Cloud
			"66.249.",  // Googlebot
			"212.93.",  // IPXO / EU-RU datacenter
			"104.28.",  // Cloudflare proxy/bot
			"13.217.",  // AWS Singapore
			"103.111.", // Local DC Jakarta
		},
		BlacklistReferrer: []string{
			"deliv12.com", "torzor.com", "miluwo.com", "asdfix.com", "explorads.com", "pcdelv.com", "popcash.net",
			"sahorjj.com", "viowrel.com",
		},
		BlacklistRefRegex:  []string{`^p\.[a-z0-9\-]+\.com$`},
		RateLimitMax:       10,
		RateLimitWindowSec: 10,
		LogAllowed:         false,
		LogBlocked:         true,
		AllowMobileOnly:    true,
	}
	bf, err := middleware.NewBotFilter(botCfg, "GeoLite2-Country.mmdb")
	if err != nil {
		utils.LogFatal(utils.LogEntry{
			Type:  "geoip_error",
			Extra: map[string]interface{}{"error": err.Error()},
		}, 1)
	}

	// ========== 3. Init Fiber Engine ==========
	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{Views: engine})

	app.Get("/logs", handlers.LogsHandler)
	app.Get("/postbacks", handlers.GetPostbacks)

	// Pasang middleware global bot filter
	app.Use(bf.Handler())

	// ========== 5. Routes lainnya ==========
	app.Get("/", handlers.RedirectHandler)
	app.Get("/postback", handlers.PostbackHandler)
	app.Get("/pre-sale", handlers.PreSaleHandler)
	app.Get("/article", handlers.ArticleHandler)
	app.Get("/main", handlers.MainHandler)

	// ========== 6. Start Server ==========
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := app.Listen(":" + port); err != nil {
		utils.LogFatal(utils.LogEntry{
			Type:  "fatal_error",
			Extra: map[string]interface{}{"error": err.Error()},
		}, 1)
	}
}
