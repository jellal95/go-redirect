package handlers

import (
	"encoding/base64"
	"fmt"
	"go-redirect/geo"
	"go-redirect/models"
	"go-redirect/utils"
	"math/rand/v2"
	"net/url"
	"strings"
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

	uaString := strings.ToLower(c.Get("User-Agent"))
	isIOS := strings.Contains(uaString, "iphone") || strings.Contains(uaString, "ipad")
	isAndroid := strings.Contains(uaString, "android")

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

		if key == "type_ads" {
			switch val {
			case models.AdTypePropeller, models.AdTypeGalaksion, models.AdTypePopcash:
				if cid := queryParams["clickid"]; cid != "" && subIDOut == "" {
					subIDOut = cid
				}
				if sid := queryParams["subid"]; sid != "" && subIDOut == "" {
					subIDOut = sid
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

	// --- App Scheme detect ---
	var appScheme string

	// Shopee (direct, accesstrade, involve, ecomobi)
	if strings.Contains(product.URL, "s.shopee.co.id") ||
		strings.Contains(strings.ToLower(product.Name), "shopee") ||
		(strings.Contains(product.URL, "atid.me") && strings.Contains(strings.ToLower(product.Name), "shopee")) ||
		(strings.Contains(product.URL, "invl.") && strings.Contains(strings.ToLower(product.Name), "shopee")) ||
		(strings.Contains(product.URL, "goeco.mobi") && strings.Contains(strings.ToLower(product.Name), "shopee")) {

		if isIOS {
			// iOS → pakai universal link (langsung Shopee web redirect)
			return c.Redirect(finalURL, 302)
		}
		if isAndroid {
			// Android pakai app scheme
			navJSON := fmt.Sprintf(`{"paths":[{"webNav":{"url":"%s"}}]}`, finalURL)
			navB64 := base64.StdEncoding.EncodeToString([]byte(navJSON))
			appScheme = "shopeeid://home?navRoute=" + navB64
		}

	} else if strings.Contains(product.URL, "c.lazada.co.id") ||
		strings.Contains(strings.ToLower(product.Name), "lazada") {
		appScheme = "lazada://id/web?url=" + url.QueryEscape(finalURL)

	} else if strings.Contains(product.URL, "goeco.mobi") &&
		strings.Contains(strings.ToLower(product.Name), "tiktok") {
		appScheme = "tiktokshop://"

	} else if strings.Contains(product.URL, "invl.") &&
		strings.Contains(strings.ToLower(product.Name), "traveloka") {
		appScheme = "traveloka://"

	} else if strings.Contains(product.URL, "agoda.com") ||
		strings.Contains(strings.ToLower(product.Name), "agoda") {
		appScheme = "agoda://"

	} else if strings.Contains(strings.ToLower(product.Name), "tokopedia") ||
		strings.Contains(product.URL, "tokopedia.") {
		appScheme = "tokopedia://home"

	} else if strings.Contains(strings.ToLower(product.Name), "blibli") ||
		strings.Contains(product.URL, "blibli.") {
		appScheme = "blibli://home"
	}

	// --- Hybrid redirect ---
	if appScheme != "" {
		html := fmt.Sprintf(`<!doctype html><html><head>
		<meta name="viewport" content="width=device-width,initial-scale=1">
		<title>Opening App...</title></head><body>
		<script>
		window.location = "%s";
		setTimeout(function(){ window.location = "%s"; }, 800);
		</script>
		</body></html>`, appScheme, finalURL)

		c.Type("html")

		return c.Status(200).SendString(html)
	}

	return c.Redirect(finalURL, 302)
}
