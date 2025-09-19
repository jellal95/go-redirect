package handlers

import (
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
)

var startTime = time.Now()

// HealthHandler provides health check endpoint
func HealthHandler(c *fiber.Ctx) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return c.JSON(fiber.Map{
		"status":    "ok",
		"timestamp": time.Now(),
		"uptime":    time.Since(startTime).String(),
		"memory": fiber.Map{
			"alloc":       m.Alloc / 1024 / 1024,      // MB
			"total_alloc": m.TotalAlloc / 1024 / 1024, // MB
			"sys":         m.Sys / 1024 / 1024,        // MB
			"num_gc":      m.NumGC,
		},
		"goroutines": runtime.NumGoroutine(),
		"version":    "1.0.0",
	})
}

// ReadinessHandler checks if all dependencies are ready
func ReadinessHandler(c *fiber.Ctx) error {
	// TODO: Add checks for database, geo db, etc.
	return c.JSON(fiber.Map{
		"ready": true,
		"checks": fiber.Map{
			"geo_db": "ok", // Will be dynamic later
			"config": "ok",
		},
	})
}
