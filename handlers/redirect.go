package handlers

import (
	"encoding/json"
	"fmt"
	"go-redirect/utils"
	"math/rand/v2"
	"net/url"
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
		// Fallback: try to find in CSV-defined products
		if csvProducts, err := utils.LoadProductsCSV("config/config.csv"); err == nil {
			for _, p := range csvProducts {
				if p.ID == productID {
					return doRedirect(c, p)
				}
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
	var subIDOut string

	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		key := string(k)
		val := string(v)
		queryParams[key] = val

		if key != "product" {
			filteredParams = append(filteredParams, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(val)))
		}

		if key == "type_ads" {
			switch val {
			case models.AdTypePropeller: // PropellerAds
				if sid, ok := queryParams["subid"]; ok && subIDOut == "" {
					subIDOut = sid
				}
			case models.AdTypeGalaksion: // Galaksion
				if cid, ok := queryParams["clickid"]; ok && subIDOut == "" {
					subIDOut = cid
				}
			case models.AdTypePopcash: // Popcash
				if cid, ok := queryParams["clickid"]; ok && subIDOut == "" {
					subIDOut = cid
				}
			}
		}
	})

	if subIDOut != "" {
		// Inject the derived sub_id into queryParams so helpers can fill placeholders
		queryParams["sub_id"] = subIDOut
	}

	finalURL := utils.BuildAffiliateURL(product.URL, queryParams)

	// --- Logging ---
	displayURL, err := url.QueryUnescape(finalURL)
	if err != nil || displayURL == "" {
		displayURL = finalURL
	}
	qp, _ := json.Marshal(queryParams)
	hd, _ := json.Marshal(headers)
	geoJson, _ := json.Marshal(geoInfo)

	entry := utils.LogEntry{
		Type:        models.TypeRouteRedirect,
		Timestamp:   time.Now(),
		ProductName: product.Name,
		URL:         displayURL,
		IP:          ip,
		UserAgent:   c.Get("User-Agent"),
		Browser:     browser,
		OS:          osName,
		Device:      device,
		Referer:     c.Get("Referer"),
		QueryRaw:    string(c.Request().URI().QueryString()),
		QueryParams: qp,
		Headers:     hd,
		Extra: map[string]interface{}{
			"geo":      geoInfo,
			"sub_id":   subIDOut,
			"type_ads": queryParams["type_ads"],
			"geo_json": string(geoJson),
		},
	}

	utils.LogInfo(entry)

	// --- Redirect ke affiliate ---
	return c.Redirect(finalURL, 302)
}
