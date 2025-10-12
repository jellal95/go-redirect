package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// color constants
var (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

// RequestLogger logs each request with referer, UA, and ad tracking info
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		method := c.Method()
		path := c.Path()
		ip := c.IP()
		userAgent := c.Get("User-Agent")
		referer := c.Get("Referer")

		typeAds := c.Query("type_ads")
		subID := c.Query("subid")
		spotID := c.Query("spot_id")

		// Detect a device type (simple)
		device := "[DESKTOP]"
		if strings.Contains(strings.ToLower(userAgent), "mobile") {
			device = "[MOBILE]"
		}

		// Log start
		if referer != "" {
			fmt.Printf("[%s] %s %s from %s %s\n", start.Format("15:04:05"), method, path, ip, device)
			fmt.Printf("       ↳ Ref: %s\n", referer)
		} else {
			fmt.Printf("[%s] %s %s from %s %s (Direct)\n", start.Format("15:04:05"), method, path, ip, device)
		}

		fmt.Printf("       ↳ UA: %s\n", userAgent)

		if typeAds != "" || subID != "" || spotID != "" {
			fmt.Printf("       ↳ Ads: %s%-10s%s | subid=%s | spotid=%s\n",
				colorCyan, typeAds, colorReset, subID, spotID)
		}

		// Process
		err := c.Next()

		// After response
		status := c.Response().StatusCode()
		duration := time.Since(start)

		var color string
		switch {
		case status >= 200 && status < 300:
			color = colorGreen
		case status >= 300 && status < 400:
			color = colorBlue
		case status >= 400 && status < 500:
			color = colorYellow
		default:
			color = colorRed
		}

		fmt.Printf("       ↳ %sStatus: %d%s | Duration: %v\n",
			color, status, colorReset, duration)

		return err
	}
}
