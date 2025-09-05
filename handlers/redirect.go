package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/url"
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
	if len(Products) == 0 {
		return c.Status(404).SendString("No products configured")
	}
	if total <= 0 {
		// Fallback to first product deterministically if weights are zero
		return doRedirect(c, Products[0])
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
	var filteredParams []string
	var actionID string

	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		key := string(k)
		val := string(v)
		queryParams[key] = val

		if key != "product" {
			// Ensure proper URL-encoding of key and value when rebuilding query string
			filteredParams = append(filteredParams, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(val)))
		}

		if key == "type_ads" {
			switch val {
			case "1": // Propeller
				if sid, ok := queryParams["subid"]; ok {
					actionID = sid
				}
			case "2": // Galaksion
				if cid, ok := queryParams["clickid"]; ok {
					actionID = cid
				}
			}
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
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(entry); err == nil {
		log.Println("Redirect Log:", buf.String())
	}

	if actionID != "" {
		log.Println("ActionID yang dilempar ke web affiliate:", actionID)
	}

	return c.Redirect(finalURL, 302)
}
