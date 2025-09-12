package utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"go-redirect/models"
)

// LoadProductsCSV reads products from a CSV file with Indonesian headers as used in config/config.csv.
// Mappings per requirement:
// - "Link Produk" => image
// - "Link Komisi Ekstra" => url
// Weighting: Prefer numeric value from "Komisi" (e.g., "Rp13.680" => 13680). If missing/zero, fallback to percentage from "Komisi hingga" (e.g., "12,5%" => 12.5). If both missing, fallback to 1.
func LoadProductsCSV(path string) ([]models.Product, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // allow variable columns
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("empty csv: %s", path)
	}

	// Build header map
	head := rows[0]
	idx := map[string]int{}
	for i, h := range head {
		// Strip UTF-8 BOM from the very first header if present to avoid lookup misses (e.g., \ufeffID Produk)
		if i == 0 {
			h = strings.TrimPrefix(h, "\ufeff")
		}
		key := strings.TrimSpace(strings.ToLower(h))
		idx[key] = i
	}

	get := func(row []string, header string) string {
		if i, ok := idx[strings.ToLower(header)]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}

	// Helper to parse currency like "Rp13.680" or "13.680" into float64 weight
	numOnly := regexp.MustCompile(`[^0-9]`)
	parseKomisi := func(s string) float64 {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0.0
		}
		clean := numOnly.ReplaceAllString(s, "")
		if clean == "" {
			return 0.0
		}
		v, err := strconv.ParseFloat(clean, 64)
		if err != nil {
			return 0.0
		}
		if v <= 0 {
			return 0.0
		}
		return v
	}

	// Helper to parse percent like "12%" or "12,5%" -> 12 or 12.5
	parseKomisiHingga := func(s string) float64 {
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "%"))
		if s == "" {
			return 0.0
		}
		// Replace comma with dot for decimal
		s = strings.ReplaceAll(s, ",", ".")
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v <= 0 {
			return 0.0
		}
		return v
	}

	var products []models.Product
	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		id := get(row, "ID Produk")
		if id == "" {
			id = get(row, "ID")
		}
		desc := get(row, "Nama Produk")
		name := get(row, "Nama Toko")
		image := get(row, "URL")
		url := get(row, "Link Komisi Ekstra")
		komisiStr := get(row, "Komisi")
		komisiHinggaStr := get(row, "Komisi hingga")
		if id == "" && desc == "" && url == "" && image == "" {
			continue
		}
		weight := parseKomisi(komisiStr)
		if weight <= 0 {
			weight = parseKomisiHingga(komisiHinggaStr)
			if weight <= 0 {
				weight = 1.0
			}
		}
		// Default name if empty, per requirement
		p := models.Product{
			ID:           id,
			Name:         name,
			Description:  desc,
			URL:          url,
			Image:        image,
			Percentage:   weight,
			Komisi:       komisiStr,
			KomisiHingga: komisiHinggaStr,
		}
		products = append(products, p)
	}

	return products, nil
}
