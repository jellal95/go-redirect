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
	TypeAdsSummary map[string]int `json:"type_ads_summary"` // Essential: traffic sources
	DeviceSummary  map[string]int `json:"device_summary"`   // Device source tracking
	BrowserSummary map[string]int `json:"browser_summary"`  // Browser analytics
	SubIDSummary   map[string]int `json:"sub_id_summary"`   // Sub ID / Click ID tracking
	SpotIDSummary  map[string]int `json:"spot_id_summary"`  // Track spot performance
	// === GEO ANALYTICS ===
	GeoSummary     map[string]int `json:"geo_summary"`     // Country targeting
	CitySummary    map[string]int `json:"city_summary"`    // City analytics
	RegionSummary  map[string]int `json:"region_summary"`  // Region analytics
	ProductSummary map[string]int `json:"product_summary"` // Essential: business metrics

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
		SubIDSummary:   make(map[string]int),
		SpotIDSummary:  make(map[string]int),
		GeoSummary:     make(map[string]int),
		CitySummary:    make(map[string]int),
		RegionSummary:  make(map[string]int),
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

				// Extract sub_id from extra data
				if subID, ok := entry.Extra["sub_id"].(string); ok && subID != "" {
					resp.SubIDSummary[subID]++
				}

				// Enhanced geo data processing
				if geo, ok := entry.Extra["geo"].(map[string]interface{}); ok {
					if country, ok := geo["country"].(string); ok && country != "" {
						resp.GeoSummary[country]++
					}
					if city, ok := geo["city"].(string); ok && city != "" {
						resp.CitySummary[city]++
					}
					if region, ok := geo["region"].(string); ok && region != "" {
						resp.RegionSummary[region]++
					}
				}
			}

			// Check spot_id and sub_id from query params
			if entry.QueryParams != nil {
				if spotID, ok := entry.QueryParams["spot_id"]; ok && spotID != "" {
					resp.SpotIDSummary[spotID]++
				}

				// Extract sub_id based on type_ads from query params
				if typeAds, ok := entry.QueryParams["type_ads"]; ok {
					var subID string
					switch typeAds {
					case "1": // PropellerAds
						if sid, ok := entry.QueryParams["subid"]; ok && sid != "" {
							subID = sid
						}
					case "2": // Galaksion
						if cid, ok := entry.QueryParams["clickid"]; ok && cid != "" {
							subID = cid
						}
					case "3": // Popcash
						if cid, ok := entry.QueryParams["clickid"]; ok && cid != "" {
							subID = cid
						}
					case "4": // ClickAdilla
						if cid, ok := entry.QueryParams["click_id"]; ok && cid != "" {
							subID = cid
						}
					}
					if subID != "" {
						resp.SubIDSummary[subID]++
					}
				}
			}
		}

		file.Close()
	}

	resp.TotalLogs = len(resp.Logs)

	sortMapKeys(resp.TypeSummary)
	sortMapKeys(resp.ProductSummary)
	sortMapKeys(resp.TypeAdsSummary)
	sortMapKeys(resp.DeviceSummary)
	sortMapKeys(resp.BrowserSummary)
	sortMapKeys(resp.SubIDSummary)
	sortMapKeys(resp.SpotIDSummary)
	sortMapKeys(resp.GeoSummary)
	sortMapKeys(resp.CitySummary)
	sortMapKeys(resp.RegionSummary)

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
