package handlers

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"go-redirect/geo"
	"go-redirect/models"

	"github.com/gofiber/fiber/v2"
	"github.com/mssola/user_agent"
)

var Products []models.Product
var Logs []models.LogEntry

func RedirectHandler(c *fiber.Ctx) error {
	rand.Seed(time.Now().UnixNano())

	if len(Products) == 0 {
		return c.Status(fiber.StatusNotFound).SendString("No product available")
	}

	// weighted random
	total := 0.0
	for _, p := range Products {
		total += p.Percentage
	}
	r := rand.Float64() * total
	sum := 0.0
	var selected *models.Product
	for _, p := range Products {
		sum += p.Percentage
		if r <= sum {
			selected = &p
			break
		}
	}
	if selected == nil {
		selected = &Products[len(Products)-1]
	}

	// Geo
	ip := c.Get("X-Forwarded-For")
	if ip == "" {
		ip = c.IP()
	}
	geoInfo := geo.GetGeoInfo(ip)

	// User Agent parsing
	ua := user_agent.New(c.Get("User-Agent"))
	browser, _ := ua.Browser()
	osName := ua.OS()
	device := "Desktop"
	if ua.Mobile() {
		device = "Mobile"
	}

	// Headers
	headers := map[string]string{}
	c.Request().Header.VisitAll(func(k, v []byte) {
		headers[string(k)] = string(v)
	})

	// Query Params
	queryParams := map[string]string{}
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		queryParams[string(k)] = string(v)
	})

	entry := models.LogEntry{
		Timestamp:   time.Now().Format(time.RFC3339),
		URL:         selected.URL,
		IP:          c.IP(),
		UserAgent:   c.Get("User-Agent"),
		Browser:     browser,
		OS:          osName,
		Device:      device,
		Referer:     c.Get("Referer"),
		QueryRaw:    c.Context().QueryArgs().String(),
		QueryParams: queryParams,
		Headers:     headers,
		Geo:         geoInfo,
	}

	Logs = append(Logs, entry)
	logData, _ := json.Marshal(entry)
	log.Println(string(logData))

	return c.Redirect(selected.URL, 302)
}
