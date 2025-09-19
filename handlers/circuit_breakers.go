package handlers

import (
	"go-redirect/utils"

	"github.com/gofiber/fiber/v2"
)

// CircuitBreakersHandler returns status of all circuit breakers
func CircuitBreakersHandler(c *fiber.Ctx) error {
	breakers := utils.GetPostbackBreakers()
	stats := breakers.GetStats()

	return c.JSON(fiber.Map{
		"status":           "ok",
		"circuit_breakers": stats,
	})
}

// CircuitBreakerResetHandler resets a specific circuit breaker
func CircuitBreakerResetHandler(c *fiber.Ctx) error {
	network := c.Params("network")
	if network == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "network parameter is required",
		})
	}

	breakers := utils.GetPostbackBreakers()
	success := breakers.Reset(network)

	if success {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Circuit breaker reset successfully",
			"network": network,
		})
	} else {
		return c.Status(404).JSON(fiber.Map{
			"error": "Circuit breaker not found for network: " + network,
		})
	}
}

// CircuitBreakersResetAllHandler resets all circuit breakers
func CircuitBreakersResetAllHandler(c *fiber.Ctx) error {
	breakers := utils.GetPostbackBreakers()
	breakers.ResetAll()

	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "All circuit breakers reset successfully",
	})
}
