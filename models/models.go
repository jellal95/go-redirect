package models

type Config struct {
	Products []Product `yaml:"products"`
	PropAds  PropAds   `yaml:"propellerAds"`
}

type PropAds struct {
	Aid         string `yaml:"aid"`
	Tid         string `yaml:"tid"`
	PostbackURL string `yaml:"postback_url"`
}

type Product struct {
	Name       string  `yaml:"name" json:"name"`
	URL        string  `yaml:"url" json:"url"`
	Percentage float64 `yaml:"percentage" json:"percentage"`
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
