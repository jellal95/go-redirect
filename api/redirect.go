package handler

import (
	"encoding/json"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

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

var (
	logs  []LogEntry
	mutex sync.Mutex
)

// Geo lookup helper
func getGeo(ipStr string, db *geoip2.Reader) GeoInfo {
	ip := net.ParseIP(ipStr)
	if ip == nil || db == nil {
		return GeoInfo{}
	}
	record, err := db.City(ip)
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

// Main handler (redirect)
func Handler(w http.ResponseWriter, r *http.Request) {
	// load config
	data, err := os.ReadFile("config/links.json")
	if err != nil {
		http.Error(w, "Config error", 500)
		return
	}
	var products []Product
	json.Unmarshal(data, &products)

	// weighted random
	total := 0.0
	for _, p := range products {
		total += p.Percentage
	}
	if total == 0 {
		http.Error(w, "No products", 500)
		return
	}
	rand.Seed(time.Now().UnixNano())
	rpick := rand.Float64() * total
	sum := 0.0
	var selected Product
	for _, p := range products {
		sum += p.Percentage
		if rpick <= sum {
			selected = p
			break
		}
	}

	// open GeoIP DB
	var geo GeoInfo
	if db, err := geoip2.Open("GeoLite2-City.mmdb"); err == nil {
		defer db.Close()
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
		geo = getGeo(ip, db)
	}

	// log entry
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		URL:       selected.URL,
		IP:        r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Referer:   r.Referer(),
		Query:     r.URL.RawQuery,
		Geo:       geo,
	}
	mutex.Lock()
	logs = append(logs, entry)
	mutex.Unlock()

	// redirect
	http.Redirect(w, r, selected.URL, http.StatusFound)
}
