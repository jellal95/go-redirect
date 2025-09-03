package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"strings"
	"time"

	"go-redirect/geo"
	"go-redirect/models"

	"github.com/gofiber/fiber/v2"
	"github.com/mssola/user_agent"
)

var Products []models.Product
var Logs []models.LogEntry

func RedirectHandler(c *fiber.Ctx) error {
	if productID := c.Query("product"); productID != "" {
		for _, p := range Products {
			if p.ID == productID {
				return doRedirect(c, p)
			}
		}
	}

	total := 0.0
	for _, p := range Products {
		total += p.Percentage
	}
	r := rand.Float64() * total
	sum := 0.0
	for _, p := range Products {
		sum += p.Percentage
		if r <= sum {
			return doRedirect(c, p)
		}
	}

	return doRedirect(c, Products[len(Products)-1])
}

func doRedirect(c *fiber.Ctx, product models.Product) error {
	// --- IP & Geo ---
	ip := c.Get("X-Forwarded-For")
	if ip == "" {
		ip = c.Get("X-Real-Ip")
	}
	if ip == "" {
		ip = c.IP()
	}
	geoInfo := geo.GetGeoInfo(ip)

	// --- User Agent ---
	ua := user_agent.New(c.Get("User-Agent"))
	browser, _ := ua.Browser()
	osName := ua.OS()
	device := "Desktop"
	if ua.Mobile() {
		device = "Mobile"
	}

	// --- Headers ---
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(k, v []byte) {
		headers[string(k)] = string(v)
	})

	// --- Query Params (all) ---
	queryParams := make(map[string]string)
	filteredParams := []string{}
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		key := string(k)
		val := string(v)
		queryParams[key] = val
		if key != "product" { // jangan ikutkan product ke merchant
			filteredParams = append(filteredParams, fmt.Sprintf("%s=%s", key, val))
		}
	})

	// --- Build final URL ---
	finalURL := product.URL
	if len(filteredParams) > 0 {
		sep := "?"
		if strings.Contains(finalURL, "?") {
			sep = "&"
		}
		finalURL = finalURL + sep + strings.Join(filteredParams, "&")
	}

	// --- Logging ---
	entry := models.LogEntry{
		Timestamp:   time.Now().Format(time.RFC3339),
		ProductName: product.Name,
		URL:         finalURL,
		IP:          ip,
		UserAgent:   c.Get("User-Agent"),
		Browser:     browser,
		OS:          osName,
		Device:      device,
		Referer:     c.Get("Referer"),
		QueryRaw:    string(c.Request().URI().QueryString()),
		QueryParams: queryParams,
		Headers:     headers,
		Geo:         geoInfo,
	}

	Logs = append(Logs, entry)
	if data, err := json.Marshal(entry); err == nil {
		log.Println(string(data))
	}

	// --- Redirect ke merchant ---
	return c.Redirect(finalURL, 302)
}
