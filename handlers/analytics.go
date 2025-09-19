package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go-redirect/utils"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type ParamAnalytics struct {
	GeneratedAt    time.Time                   `json:"generated_at"`
	TotalRequests  int                         `json:"total_requests"`
	ParameterStats map[string]ParameterStat    `json:"parameter_stats"`
	TopValues      map[string]map[string]int64 `json:"top_values"`
	Combinations   map[string]int64            `json:"common_combinations"`
}

type ParameterStat struct {
	Count        int64   `json:"count"`
	Percentage   float64 `json:"percentage"`
	UniqueValues int     `json:"unique_values"`
}

// AnalyticsParamsHandler analyzes query parameters from logs
func AnalyticsParamsHandler(c *fiber.Ctx) error {
	analytics := &ParamAnalytics{
		GeneratedAt:    time.Now(),
		ParameterStats: make(map[string]ParameterStat),
		TopValues:      make(map[string]map[string]int64),
		Combinations:   make(map[string]int64),
	}

	// Read log files
	folder := "logs"
	files, err := filepath.Glob(fmt.Sprintf("%s/log-*.jsonl", folder))
	if err != nil || len(files) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "no log files found"})
	}

	paramCounts := make(map[string]int64)
	valueCounts := make(map[string]map[string]int64)
	combinations := make(map[string]int64)
	totalRequests := 0

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

			// Only analyze redirect and pre-sale requests
			if entry.Type != "redirect" && entry.Type != "pre-sale" {
				continue
			}

			totalRequests++

			if entry.QueryParams != nil {
				var paramKeys []string
				for key, value := range entry.QueryParams {
					if key == "" || value == "" {
						continue
					}

					// Count parameter frequency
					paramCounts[key]++
					paramKeys = append(paramKeys, key)

					// Count value frequency for each parameter
					if valueCounts[key] == nil {
						valueCounts[key] = make(map[string]int64)
					}
					// Truncate long values
					if len(value) > 50 {
						value = value[:50] + "..."
					}
					valueCounts[key][value]++
				}

				// Count parameter combinations
				if len(paramKeys) > 1 {
					sort.Strings(paramKeys)
					combo := strings.Join(paramKeys, "+")
					combinations[combo]++
				}
			}
		}
		file.Close()
	}

	analytics.TotalRequests = totalRequests

	// Build parameter stats
	for param, count := range paramCounts {
		uniqueValues := len(valueCounts[param])
		percentage := 0.0
		if totalRequests > 0 {
			percentage = float64(count) / float64(totalRequests) * 100
		}

		analytics.ParameterStats[param] = ParameterStat{
			Count:        count,
			Percentage:   percentage,
			UniqueValues: uniqueValues,
		}

		// Get top 10 values for each parameter
		topValues := getTopValues(valueCounts[param], 10)
		analytics.TopValues[param] = topValues
	}

	// Get top 10 combinations
	analytics.Combinations = getTopValues(combinations, 10)

	return c.JSON(analytics)
}

// AnalyticsRefererHandler analyzes referer patterns
func AnalyticsRefererHandler(c *fiber.Ctx) error {
	type RefererAnalytics struct {
		GeneratedAt    time.Time        `json:"generated_at"`
		TotalRequests  int              `json:"total_requests"`
		DirectTraffic  int64            `json:"direct_traffic"`
		RefererDomains map[string]int64 `json:"referer_domains"`
		RefererPaths   map[string]int64 `json:"referer_paths"`
		TopReferers    map[string]int64 `json:"top_referers"`
	}

	analytics := &RefererAnalytics{
		GeneratedAt:    time.Now(),
		RefererDomains: make(map[string]int64),
		RefererPaths:   make(map[string]int64),
		TopReferers:    make(map[string]int64),
	}

	// Read log files
	folder := "logs"
	files, err := filepath.Glob(fmt.Sprintf("%s/log-*.jsonl", folder))
	if err != nil || len(files) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "no log files found"})
	}

	totalRequests := 0
	directTraffic := int64(0)

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

			if entry.Type != "redirect" && entry.Type != "pre-sale" {
				continue
			}

			totalRequests++

			if entry.Referer == "" {
				directTraffic++
			} else {
				// Extract domain from referer
				referer := entry.Referer
				if strings.HasPrefix(referer, "http://") {
					referer = strings.TrimPrefix(referer, "http://")
				} else if strings.HasPrefix(referer, "https://") {
					referer = strings.TrimPrefix(referer, "https://")
				}

				parts := strings.Split(referer, "/")
				if len(parts) > 0 {
					domain := parts[0]
					analytics.RefererDomains[domain]++

					if len(parts) > 1 {
						path := "/" + strings.Join(parts[1:], "/")
						if len(path) > 50 {
							path = path[:50] + "..."
						}
						analytics.RefererPaths[path]++
					}
				}

				// Truncate full referer for top list
				if len(entry.Referer) > 60 {
					referer = entry.Referer[:60] + "..."
				} else {
					referer = entry.Referer
				}
				analytics.TopReferers[referer]++
			}
		}
		file.Close()
	}

	analytics.TotalRequests = totalRequests
	analytics.DirectTraffic = directTraffic

	// Get top 20 for each category
	analytics.RefererDomains = getTopValues(analytics.RefererDomains, 20)
	analytics.RefererPaths = getTopValues(analytics.RefererPaths, 15)
	analytics.TopReferers = getTopValues(analytics.TopReferers, 20)

	return c.JSON(analytics)
}

// getTopValues returns top N values from a map
func getTopValues(values map[string]int64, limit int) map[string]int64 {
	type kv struct {
		Key   string
		Value int64
	}

	var sorted []kv
	for k, v := range values {
		sorted = append(sorted, kv{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	result := make(map[string]int64)
	for i, item := range sorted {
		if i >= limit {
			break
		}
		result[item.Key] = item.Value
	}

	return result
}
