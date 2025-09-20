package middleware

import (
	"sync"

	"github.com/gofiber/fiber/v2"
)

// Global bot filter state
var (
	botFilterEnabled = true
	botFilterMutex   sync.RWMutex
)

// SetBotFilterEnabled sets the global bot filter state
func SetBotFilterEnabled(enabled bool) {
	botFilterMutex.Lock()
	defer botFilterMutex.Unlock()
	botFilterEnabled = enabled
}

// IsBotFilterEnabled returns the current bot filter state
func IsBotFilterEnabled() bool {
	botFilterMutex.RLock()
	defer botFilterMutex.RUnlock()
	return botFilterEnabled
}

// ConditionalBotFilter applies bot filter only when enabled
func ConditionalBotFilter(botFilter *botFilter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if bot filter is enabled
		if !IsBotFilterEnabled() {
			// Bot filter disabled, skip to next handler
			return c.Next()
		}

		// Bot filter enabled, apply the filter
		return botFilter.Handler()(c)
	}
}
