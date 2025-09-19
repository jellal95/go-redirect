package handlers

import (
	"go-redirect/utils"

	"github.com/gofiber/fiber/v2"
)

// MetricsHandler provides application metrics endpoint
func MetricsHandler(c *fiber.Ctx) error {
	metrics := utils.GetMetrics()

	// Add calculated fields
	response := fiber.Map{
		"metrics": metrics,
		"calculated": fiber.Map{
			"average_response_time_ms": metrics.AverageResponseTime(),
			"error_rate":               calculateErrorRate(metrics),
			"redirect_rate":            calculateRedirectRate(metrics),
			"block_rate":               calculateBlockRate(metrics),
		},
	}

	return c.JSON(response)
}

// MetricsResetHandler resets all metrics
func MetricsResetHandler(c *fiber.Ctx) error {
	utils.ResetMetrics()
	return c.JSON(fiber.Map{
		"status":  "metrics_reset",
		"message": "All metrics have been reset to zero",
	})
}

func calculateErrorRate(m *utils.Metrics) float64 {
	if m.RequestCount == 0 {
		return 0
	}
	return float64(m.ErrorCount) / float64(m.RequestCount) * 100
}

func calculateRedirectRate(m *utils.Metrics) float64 {
	if m.RequestCount == 0 {
		return 0
	}
	return float64(m.RedirectCount) / float64(m.RequestCount) * 100
}

func calculateBlockRate(m *utils.Metrics) float64 {
	if m.RequestCount == 0 {
		return 0
	}
	return float64(m.BlockedCount) / float64(m.RequestCount) * 100
}
