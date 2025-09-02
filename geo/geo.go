package geo

import (
	"net"
	"strings"

	"go-redirect/models"

	"github.com/oschwald/geoip2-golang"
)

var GeoDB *geoip2.Reader

func InitGeoDB(path string) error {
	var err error
	GeoDB, err = geoip2.Open(path)
	return err
}

// ipStr = bisa IP asli atau dari X-Forwarded-For
func GetGeoInfo(ipStr string) models.GeoInfo {
	// ambil IP pertama kalau ada list (X-Forwarded-For bisa banyak dipisah koma)
	if strings.Contains(ipStr, ",") {
		parts := strings.Split(ipStr, ",")
		ipStr = strings.TrimSpace(parts[0])
	}

	ip := net.ParseIP(ipStr)
	if ip == nil || GeoDB == nil {
		return models.GeoInfo{
			Country:  "Unknown",
			Region:   "Unknown",
			City:     "Unknown",
			Timezone: "Unknown",
		}
	}

	record, err := GeoDB.City(ip)
	if err != nil {
		return models.GeoInfo{
			Country:  "Unknown",
			Region:   "Unknown",
			City:     "Unknown",
			Timezone: "Unknown",
		}
	}

	geo := models.GeoInfo{
		Country:   record.Country.Names["en"],
		City:      record.City.Names["en"],
		Timezone:  record.Location.TimeZone,
		Latitude:  record.Location.Latitude,
		Longitude: record.Location.Longitude,
	}
	if len(record.Subdivisions) > 0 {
		geo.Region = record.Subdivisions[0].Names["en"]
	}
	return geo
}
