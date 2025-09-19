package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go-redirect/utils"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gofiber/fiber/v2"
)

type LogsResponse struct {
	// === SUMMARY (shown first) ===
	GeneratedAt    time.Time      `json:"generated_at"`
	TotalLogs      int            `json:"total_logs"`
	TypeSummary    map[string]int `json:"type_summary"`     // Essential: log types
	ProductSummary map[string]int `json:"product_summary"`  // Essential: business metrics
	TypeAdsSummary map[string]int `json:"type_ads_summary"` // Essential: traffic sources
	DeviceSummary  map[string]int `json:"device_summary"`   // Device source tracking
	BrowserSummary map[string]int `json:"browser_summary"`  // Browser analytics
	SpotIDSummary  map[string]int `json:"spot_id_summary"`  // Track spot performance
	TimeSummary    map[string]int `json:"time_summary"`     // Essential: hourly performance
	GeoSummary     map[string]int `json:"geo_summary"`      // Essential: country targeting

	// === RAW LOGS (shown last) ===
	Logs []utils.LogEntry `json:"logs"`
}

func LogsHandler(c *fiber.Ctx) error {
	folder := "logs"
	files, err := filepath.Glob(fmt.Sprintf("%s/log-*.jsonl", folder))
	if err != nil || len(files) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "no log files found"})
	}

	resp := LogsResponse{
		GeneratedAt:    time.Now(),
		TotalLogs:      0,
		TypeSummary:    make(map[string]int),
		ProductSummary: make(map[string]int),
		TypeAdsSummary: make(map[string]int),
		DeviceSummary:  make(map[string]int),
		BrowserSummary: make(map[string]int),
		SpotIDSummary:  make(map[string]int),
		TimeSummary:    make(map[string]int),
		GeoSummary:     make(map[string]int),
		Logs:           []utils.LogEntry{},
	}

	for _, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var entry utils.LogEntry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				continue
			}

			resp.Logs = append(resp.Logs, entry)

			// --- Essential summaries only ---
			resp.TypeSummary[entry.Type]++

			if entry.ProductName != "" {
				resp.ProductSummary[entry.ProductName]++
			}

			if entry.Device != "" {
				resp.DeviceSummary[entry.Device]++
			}

			if entry.Browser != "" {
				resp.BrowserSummary[entry.Browser]++
			}

			if entry.Extra != nil {
				if typeAds, ok := entry.Extra["type_ads"].(string); ok && typeAds != "" {
					resp.TypeAdsSummary[typeAds]++
				}
				if geo, ok := entry.Extra["geo"].(map[string]interface{}); ok {
					if country, ok := geo["country"].(string); ok && country != "" {
						resp.GeoSummary[country]++
					}
				}
			}

			// Check spot_id from query params
			if entry.QueryParams != nil {
				if spotID, ok := entry.QueryParams["spot_id"]; ok && spotID != "" {
					resp.SpotIDSummary[spotID]++
				}
			}

			// Hourly performance tracking
			hour := entry.Timestamp.Format("2006-01-02 15:00")
			resp.TimeSummary[hour]++
		}

		file.Close()
	}

	resp.TotalLogs = len(resp.Logs)

	// Sort essential summaries by key for consistent output
	sortMapKeys(resp.TypeSummary)
	sortMapKeys(resp.ProductSummary)
	sortMapKeys(resp.TypeAdsSummary)
	sortMapKeys(resp.DeviceSummary)
	sortMapKeys(resp.BrowserSummary)
	sortMapKeys(resp.SpotIDSummary)
	sortMapKeys(resp.TimeSummary)
	sortMapKeys(resp.GeoSummary)

	return c.JSON(resp)
}

func sortMapKeys(m map[string]int) {
	if len(m) == 0 {
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := make(map[string]int, len(m))
	for _, k := range keys {
		sorted[k] = m[k]
	}
	for k := range m {
		delete(m, k)
	}
	for k, v := range sorted {
		m[k] = v
	}
}
