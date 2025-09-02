package models

type Product struct {
	URL        string  `json:"url"`
	Percentage float64 `json:"percentage"`
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
