package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/oschwald/geoip2-golang"
)

type Product struct {
	URL        string  `json:"url"`
	Percentage float64 `json:"percentage"`
}

type GeoInfo struct {
	Country string `json:"country"`
	Region  string `json:"region"`
	City    string `json:"city"`
}

type LogEntry struct {
	Timestamp string  `json:"timestamp"`
	URL       string  `json:"url"`
	IP        string  `json:"ip"`
	UserAgent string  `json:"user_agent"`
	Referer   string  `json:"referer"`
	Query     string  `json:"query"`
	Geo       GeoInfo `json:"geo"`
}

var products []Product
var logFile *os.File
var logs []LogEntry
var geoDB *geoip2.Reader

// Load JSON config
func LoadProducts(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &products)
}

// Init log file
func InitLog(filePath string) error {
	var err error
	logFile, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	log.SetOutput(logFile)
	return nil
}

// Init GeoIP DB
func InitGeoDB(dbPath string) error {
	var err error
	geoDB, err = geoip2.Open(dbPath)
	return err
}

// Get Geo info by IP
func GetGeoInfo(ipStr string) GeoInfo {
	ip := net.ParseIP(ipStr)
	if ip == nil || geoDB == nil {
		return GeoInfo{}
	}
	record, err := geoDB.City(ip)
	if err != nil {
		return GeoInfo{}
	}
	return GeoInfo{
		Country: record.Country.Names["en"],
		Region: func() string {
			if len(record.Subdivisions) > 0 {
				return record.Subdivisions[0].Names["en"]
			}
			return ""
		}(),
		City: record.City.Names["en"],
	}
}

// Weighted redirect handler
func RedirectHandler(c *fiber.Ctx) error {
	rand.Seed(time.Now().UnixNano())

	if len(products) == 0 {
		return c.Status(fiber.StatusNotFound).SendString("No product available")
	}

	// Hitung total percentage
	total := 0.0
	for _, p := range products {
		total += p.Percentage
	}

	// Random weighted
	r := rand.Float64() * total
	sum := 0.0
	var selected *Product
	for _, p := range products {
		sum += p.Percentage
		if r <= sum {
			selected = &p
			break
		}
	}
	if selected == nil {
		selected = &products[len(products)-1]
	}

	// Geo info
	geo := GetGeoInfo(c.IP())

	// Simpan log di memory & file
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		URL:       selected.URL,
		IP:        c.IP(),
		UserAgent: c.Get("User-Agent"),
		Referer:   c.Get("Referer"),
		Query:     c.Context().QueryArgs().String(),
		Geo:       geo,
	}

	logs = append(logs, entry)

	// Tulis ke file
	logData, _ := json.Marshal(entry)
	logFile.WriteString(string(logData) + "\n")

	// Redirect
	return c.Redirect(selected.URL, 302)
}

func main() {
	app := fiber.New()

	// Load config
	if err := LoadProducts("config/links.json"); err != nil {
		log.Fatal(err)
	}

	// Init log
	if err := InitLog("logs/clicks.log"); err != nil {
		log.Fatal(err)
	}

	// Init GeoIP DB
	if err := InitGeoDB("GeoLite2-City.mmdb"); err != nil {
		log.Println("GeoIP DB not found, skipping geo info")
	}

	// Route direct redirect
	app.Get("/", RedirectHandler)

	// Endpoint untuk akses semua log / data request
	app.Get("/logs", func(c *fiber.Ctx) error {
		return c.JSON(logs)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Server running on :" + port)
	app.Listen(":" + port)
}
