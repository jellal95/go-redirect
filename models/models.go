package models

import (
	"time"

	"gorm.io/datatypes"
)

// Ad network type identifiers used in query param `type_ads` and postbacks.
const (
	AdTypePropeller   = "1" // PropellerAds
	AdTypeGalaksion   = "2" // Galaksion
	AdTypePopcash     = "3" // Popcash
	AdTypeClickAdilla = "4"
	TypeRouteRedirect = "redirect"
	TypeRoutePreSale  = "pre-sale"
	TypePostback      = "postback"
)

type Product struct {
	ID           string  `json:"id" yaml:"id"`
	Name         string  `json:"name" yaml:"name"`
	Description  string  `json:"description" yaml:"description"`
	URL          string  `json:"url" yaml:"url"`
	Image        string  `json:"image" yaml:"image"`
	Percentage   float64 `json:"percentage" yaml:"percentage"`
	Komisi       string  `json:"komisi" yaml:"komisi"`
	KomisiHingga string  `json:"komisi_hingga" yaml:"komisi_hingga"`
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
	ID          uint      `gorm:"primaryKey"`
	Type        string    `gorm:"index"` // "redirect" / "pre-sale"
	Timestamp   time.Time `gorm:"index"`
	ProductName string    `gorm:"index"`
	URL         string
	IP          string
	UserAgent   string
	Browser     string
	OS          string
	Device      string
	Referer     string
	QueryRaw    string
	QueryParams datatypes.JSON `gorm:"type:jsonb"`
	Headers     datatypes.JSON `gorm:"type:jsonb"`
	Geo         datatypes.JSON `gorm:"type:jsonb"`
}
