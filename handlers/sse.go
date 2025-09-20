package handlers

import (
	"encoding/json"
	"fmt"
	"go-redirect/utils"
	"time"

	"github.com/gofiber/fiber/v2"
)

// SSEHandler provides server-sent events for real-time dashboard updates
func SSEHandler(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Headers", "Cache-Control")

	// Send initial connection message
	c.WriteString("event: connected\n")
	c.WriteString(fmt.Sprintf("data: {\"message\": \"Connected to live analytics\", \"timestamp\": \"%s\"}\n\n", time.Now().Format(time.RFC3339)))

	// Create a ticker for periodic updates
	ticker := time.NewTicker(10 * time.Second) // Send updates every 10 seconds
	defer ticker.Stop()

	// Channel to signal when client disconnects
	clientGone := make(chan bool)

	// Monitor client connection
	go func() {
		<-c.Context().Done()
		clientGone <- true
	}()

	for {
		select {
		case <-clientGone:
			return nil
		case <-ticker.C:
			// Get latest analytics data
			analyticsData := getAnalyticsUpdate()
			jsonData, err := json.Marshal(analyticsData)
			if err != nil {
				continue
			}

			// Send analytics update
			c.WriteString("event: analytics\n")
			c.WriteString(fmt.Sprintf("data: %s\n\n", string(jsonData)))
		}
	}
}

// getAnalyticsUpdate returns current analytics summary for SSE
func getAnalyticsUpdate() map[string]interface{} {
	// Get current log summary (simplified version)
	summary := utils.LogsSummary()

	// Add timestamp
	summary["timestamp"] = time.Now().Format(time.RFC3339)

	// Add recent log count (last minute)
	recentCount := getRecentLogCount()
	summary["recent_activity"] = recentCount

	return summary
}

// getRecentLogCount returns the number of logs in the last minute
func getRecentLogCount() int {
	count := 0
	cutoffTime := time.Now().Add(-1 * time.Minute)

	for _, entry := range utils.Logs {
		if entry.Timestamp.After(cutoffTime) {
			count++
		}
	}

	return count
}
