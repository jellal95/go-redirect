package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mathrand "math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RequestID generates unique request IDs for tracing
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if request ID already exists (from load balancer, etc.)
		reqID := c.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}

		// Set request ID in context and response header
		c.Locals("request_id", reqID)
		c.Set("X-Request-ID", reqID)

		return c.Next()
	}
}

// generateRequestID creates a random hex string
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto rand fails
		mathrand.Seed(time.Now().UnixNano())
		return fmt.Sprintf("req_%d", mathrand.Int63())
	}
	return hex.EncodeToString(bytes)
}

// GetRequestID extracts request ID from fiber context
func GetRequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals("request_id").(string); ok {
		return id
	}
	return ""
}
