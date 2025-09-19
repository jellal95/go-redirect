package main

import (
	"fmt"
	"go-redirect/geo"
	"go-redirect/middleware"
	"os"
	"time"

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

	// ========== 2. Init Geo Database ==========
	if err := geo.InitGeoDB("GeoLite2-City.mmdb"); err != nil {
		utils.LogInfo(utils.LogEntry{
			Type:  "geo_db_warning",
			Extra: map[string]interface{}{"error": err.Error(), "message": "Geo data will show as Unknown"},
		})
	}

	// ========== 3. Setup Bot Filter Middleware ==========
	botCfg := middleware.BotFilterConfig{
		AllowCountries:     appCfg.BotFilter.AllowCountries,
		BlacklistUA:        appCfg.BotFilter.BlacklistUA,
		BlacklistIPPrefix:  appCfg.BotFilter.BlacklistIPPrefix,
		BlacklistReferrer:  appCfg.BotFilter.BlacklistReferrer,
		BlacklistRefRegex:  appCfg.BotFilter.BlacklistRefRegex,
		RateLimitMax:       appCfg.BotFilter.RateLimitMax,
		RateLimitWindowSec: appCfg.BotFilter.RateLimitWindowSec,
		LogAllowed:         appCfg.BotFilter.LogAllowed,
		LogBlocked:         appCfg.BotFilter.LogBlocked,
		AllowMobileOnly:    appCfg.BotFilter.AllowMobileOnly,
	}
	bf, err := middleware.NewBotFilter(botCfg, "GeoLite2-Country.mmdb")
	if err != nil {
		utils.LogFatal(utils.LogEntry{
			Type:  "geoip_error",
			Extra: map[string]interface{}{"error": err.Error()},
		}, 1)
	}

	// ========== 4. Init Fiber Engine ==========
	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{Views: engine})

	// ========== 5. Request ID middleware ==========
	app.Use(middleware.RequestID())

	// ========== 6. Console logging dengan metrics ==========
	app.Use(middleware.RequestLogger())

	// ========== 7. Health & Admin endpoints (tanpa bot filter) ==========
	app.Get("/health", handlers.HealthHandler)
	app.Get("/ready", handlers.ReadinessHandler)
	app.Get("/logs", handlers.LogsHandler)
	app.Get("/postbacks", handlers.GetPostbacks)
	app.Get("/metrics", handlers.MetricsHandler)
	app.Post("/metrics/reset", handlers.MetricsResetHandler)
	app.Get("/analytics/params", handlers.AnalyticsParamsHandler)
	app.Get("/analytics/referers", handlers.AnalyticsRefererHandler)
	app.Get("/circuit-breakers", handlers.CircuitBreakersHandler)
	app.Post("/circuit-breakers/reset/:network", handlers.CircuitBreakerResetHandler)
	app.Post("/circuit-breakers/reset-all", handlers.CircuitBreakersResetAllHandler)

	// ========== 8. Pasang middleware global bot filter ==========
	app.Use(bf.Handler())

	// ========== 9. Routes lainnya ==========
	app.Get("/", handlers.RedirectHandler)
	app.Get("/postback", handlers.PostbackHandler)
	app.Get("/pre-sale", handlers.PreSaleHandler)
	app.Get("/article", handlers.ArticleHandler)
	app.Get("/main", handlers.MainHandler)

	// ========== 10. Start Server dengan Graceful Shutdown ==========
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start graceful shutdown handler in background
	go utils.GracefulShutdown(app, 30*time.Second)

	fmt.Printf("🚀 Server starting on port %s\n", port)
	utils.LogInfo(utils.LogEntry{
		Type:  "server_start",
		Extra: map[string]interface{}{"port": port},
	})

	if err := app.Listen(":" + port); err != nil {
		utils.LogFatal(utils.LogEntry{
			Type:  "fatal_error",
			Extra: map[string]interface{}{"error": err.Error()},
		}, 1)
	}
}
