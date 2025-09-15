package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand/v2"
	"time"

	"go-redirect/models"
	"go-redirect/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/mssola/user_agent"
)

func PreSaleHandler(c *fiber.Ctx) error {
	// Always load from CSV as per requirement
	products, err := utils.LoadProductsCSV("config/config.csv")
	if err != nil {
		log.Println("failed to load CSV products:", err)
	}
	if len(products) == 0 {
		return c.Status(404).SendString("No products configured")
	}

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

	// --- Logging Impression ---
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

	qp, _ := json.Marshal(c.Queries())
	hd := make(map[string]string)
	c.Request().Header.VisitAll(func(k, v []byte) {
		hd[string(k)] = string(v)
	})
	hdJson, _ := json.Marshal(hd)

	entry := models.LogEntry{
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
		QueryRaw:    string(c.Request().URI().QueryString()),
		QueryParams: qp,
		Headers:     hdJson,
	}
	if err := utils.DB.Create(&entry).Error; err != nil {
		log.Println("Failed insert pre-sale log:", err)
	}

	Logs = append(Logs, entry)
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(entry); err == nil {
		log.Println("PreSale Log (view):", buf.String())
	}

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
