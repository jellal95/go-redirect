package handlers

import (
	"go-redirect/geo"
	"go-redirect/models"
	"go-redirect/utils"
	"math/rand/v2"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mssola/user_agent"
)

var Products []models.Product

func RedirectHandler(c *fiber.Ctx) error {
	if productID := c.Query("product"); productID != "" {
		for _, p := range Products {
			if p.ID == productID {
				return doRedirect(c, p)
			}
		}
		// fallback CSV
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
	headers := map[string]string{}
	c.Request().Header.VisitAll(func(k, v []byte) {
		headers[string(k)] = string(v)
	})

	// --- Query Params ---
	queryParams := map[string]string{}
	var subIDOut string
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		key := string(k)
		val := string(v)
		queryParams[key] = val

		// sub_id logic
		if key == "type_ads" {
			switch val {
			case models.AdTypePropeller:
				if sid, ok := queryParams["subid"]; ok && subIDOut == "" {
					subIDOut = sid
				}
			case models.AdTypeGalaksion:
				if cid, ok := queryParams["clickid"]; ok && subIDOut == "" {
					subIDOut = cid
				}
			case models.AdTypePopcash:
				if cid, ok := queryParams["clickid"]; ok && subIDOut == "" {
					subIDOut = cid
				}
			}
		}
	})

	if subIDOut != "" {
		queryParams["sub_id"] = subIDOut
	}

	finalURL := utils.BuildAffiliateURL(product.URL, queryParams)

	// --- Logging ---
	extra := map[string]interface{}{
		"geo":      geoInfo,
		"sub_id":   subIDOut,
		"type_ads": queryParams["type_ads"],
	}

	utils.LogInfo(utils.LogEntry{
		Type:        models.TypeRouteRedirect,
		Timestamp:   time.Now(),
		ProductName: product.Name,
		URL:         finalURL,
		IP:          ip,
		UserAgent:   c.Get("User-Agent"),
		Browser:     browser,
		OS:          osName,
		Device:      device,
		Referer:     c.Get("Referer"),
		QueryParams: queryParams,
		Headers:     headers,
		Extra:       extra,
	})

	// --- Redirect ---
	return c.Redirect(finalURL, 302)
}
