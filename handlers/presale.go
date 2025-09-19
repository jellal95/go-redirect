package handlers

import (
	"go-redirect/models"
	"go-redirect/utils"
	"math/rand/v2"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mssola/user_agent"
)

func PreSaleHandler(c *fiber.Ctx) error {
	// --- Load CSV Products ---
	products, err := utils.LoadProductsCSV("config/config.csv")
	if err != nil {
		utils.LogInfo(utils.LogEntry{
			Type: "load_csv_error",
			Extra: map[string]interface{}{
				"file":  "config/config.csv",
				"error": err.Error(),
			},
		})
	}
	if len(products) == 0 {
		utils.LogInfo(utils.LogEntry{
			Type: "no_products_configured",
			Extra: map[string]interface{}{
				"file": "config/config.csv",
			},
		})
		return c.Status(404).SendString("No products configured")
	}

	// --- Select Product by Percentage ---
	total := 0.0
	for _, p := range products {
		total += p.Percentage
	}

	var selected models.Product
	if total <= 0 {
		selected = products[0]
	} else {
		r := rand.Float64() * total
		sum := 0.0
		for _, p := range products {
			sum += p.Percentage
			if r <= sum {
				selected = p
				break
			}
		}
		if selected.ID == "" {
			selected = products[len(products)-1]
		}
	}

	// --- Get IP & User Agent ---
	ip := c.Get("X-Forwarded-For")
	if ip == "" {
		ip = c.Get("X-Real-Ip")
	}
	if ip == "" {
		ip = c.IP()
	}

	ua := user_agent.New(c.Get("User-Agent"))
	browser, _ := ua.Browser()
	osName := ua.OS()
	device := "Desktop"
	if ua.Mobile() {
		device = "Mobile"
	}

	// --- Build QueryParams & Headers ---
	queryParams := make(map[string]string)
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		queryParams[string(k)] = string(v)
	})

	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(k, v []byte) {
		headers[string(k)] = string(v)
	})

	// --- Build Extra ---
	extra := map[string]interface{}{
		"query_raw": string(c.Request().URI().QueryString()),
	}

	// --- Log Impression ---
	utils.LogInfo(utils.LogEntry{
		Type:        models.TypeRoutePreSale,
		Timestamp:   time.Now(),
		ProductName: selected.Name,
		URL:         c.OriginalURL(),
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

	// --- Render Page ---
	return c.Render("pre-sale", fiber.Map{
		"ID":           selected.ID,
		"Name":         selected.Name,
		"Description":  selected.Description,
		"Image":        selected.Image,
		"Komisi":       selected.Komisi,
		"KomisiHingga": selected.KomisiHingga,
	})
}
