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
	GeneratedAt       time.Time        `json:"generated_at"`
	Logs              []utils.LogEntry `json:"logs"`
	TotalLogs         int              `json:"total_logs"`
	TypeSummary       map[string]int   `json:"type_summary"`
	IPSummary         map[string]int   `json:"ip_summary"`
	ProductSummary    map[string]int   `json:"product_summary"`
	SubIDSummary      map[string]int   `json:"sub_id_summary"`
	TypeAdsSummary    map[string]int   `json:"type_ads_summary"`
	SpotIDSummary     map[string]int   `json:"spot_id_summary"`
	CampaignIDSummary map[string]int   `json:"campaign_id_summary"`
	TimeSummary       map[string]int   `json:"time_summary"` // per jam
	GeoSummary        map[string]int   `json:"geo_summary"`  // country
}

func LogsHandler(c *fiber.Ctx) error {
	folder := "logs"
	files, err := filepath.Glob(fmt.Sprintf("%s/log-*.jsonl", folder))
	if err != nil || len(files) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "no log files found"})
	}

	resp := LogsResponse{
		GeneratedAt:       time.Now(),
		Logs:              []utils.LogEntry{},
		TotalLogs:         0,
		TypeSummary:       make(map[string]int),
		IPSummary:         make(map[string]int),
		ProductSummary:    make(map[string]int),
		SubIDSummary:      make(map[string]int),
		TypeAdsSummary:    make(map[string]int),
		SpotIDSummary:     make(map[string]int),
		CampaignIDSummary: make(map[string]int),
		TimeSummary:       make(map[string]int),
		GeoSummary:        make(map[string]int),
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

			// --- summary ---
			resp.TypeSummary[entry.Type]++
			if entry.IP != "" {
				resp.IPSummary[entry.IP]++
			}
			if entry.ProductName != "" {
				resp.ProductSummary[entry.ProductName]++
			}
			if entry.Extra != nil {
				if subID, ok := entry.Extra["sub_id"].(string); ok && subID != "" {
					resp.SubIDSummary[subID]++
				}
				if typeAds, ok := entry.Extra["type_ads"].(string); ok && typeAds != "" {
					resp.TypeAdsSummary[typeAds]++
				}
				if spotID, ok := entry.Extra["spot_id"].(string); ok && spotID != "" {
					resp.SpotIDSummary[spotID]++
				}
				if campaignID, ok := entry.Extra["campaign_id"].(string); ok && campaignID != "" {
					resp.CampaignIDSummary[campaignID]++
				}
				if geo, ok := entry.Extra["geo"].(map[string]interface{}); ok {
					if country, ok := geo["country"].(string); ok && country != "" {
						resp.GeoSummary[country]++
					}
				}
			}

			// per jam
			hour := entry.Timestamp.Format("2006-01-02 15:00")
			resp.TimeSummary[hour]++
		}

		file.Close()
	}

	resp.TotalLogs = len(resp.Logs)

	// optional: sort maps by key
	sortMapKeys(resp.TypeSummary)
	sortMapKeys(resp.IPSummary)
	sortMapKeys(resp.ProductSummary)
	sortMapKeys(resp.SubIDSummary)
	sortMapKeys(resp.TypeAdsSummary)
	sortMapKeys(resp.SpotIDSummary)
	sortMapKeys(resp.CampaignIDSummary)
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
