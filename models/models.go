package models

// Ad network type identifiers used in query param `type_ads` and postbacks.
const (
	AdTypePropeller = "1" // PropellerAds
	AdTypeGalaksion = "2" // Galaksion
	AdTypePopcash   = "3" // Popcash
)

type Product struct {
	ID         string  `json:"id" yaml:"id"`
	Name       string  `json:"name" yaml:"name"`
	URL        string  `json:"url" yaml:"url"`
	Image      string  `json:"image" yaml:"image"`
	Percentage float64 `json:"percentage" yaml:"percentage"`
}

type GeoInfo struct {
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
}

type LogEntry struct {
	Timestamp   string            `json:"timestamp"`
	ProductName string            `json:"product_name"`
	URL         string            `json:"url"`
	IP          string            `json:"ip"`
	UserAgent   string            `json:"user_agent"`
	Browser     string            `json:"browser"`
	OS          string            `json:"os"`
	Device      string            `json:"device"`
	Referer     string            `json:"referer"`
	QueryRaw    string            `json:"query_raw"`
	QueryParams map[string]string `json:"query_params"`
	Headers     map[string]string `json:"headers"`
	Geo         GeoInfo           `json:"geo"`
}
