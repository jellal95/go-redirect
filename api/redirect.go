package handler

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
)

type Product struct {
	URL        string  `json:"url"`
	Percentage float64 `json:"percentage"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	// load JSON
	data, err := os.ReadFile("config/links.json")
	if err != nil {
		http.Error(w, "Config error", http.StatusInternalServerError)
		return
	}

	var products []Product
	json.Unmarshal(data, &products)

	// weighted random
	total := 0.0
	for _, p := range products {
		total += p.Percentage
	}
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

	// redirect
	http.Redirect(w, r, selected.URL, http.StatusFound)
}
