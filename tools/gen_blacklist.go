package main

import (
	"fmt"
	"go-redirect/utils"
	"net"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

func main() {
	asnDB, err := geoip2.Open("GeoLite2-ASN.mmdb")
	if err != nil {
		utils.LogFatal(utils.LogEntry{
			Type:  "geoip_asn_open_error",
			Extra: map[string]interface{}{"error": err.Error(), "db": "GeoLite2-ASN.mmdb"},
		}, 1)
	}
	defer asnDB.Close()

	ips := []string{
		"212.93.128.85",
		"104.28.245.127",
		"13.217.146.223",
		"103.111.96.70",
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		rec, err := asnDB.ASN(ip)
		if err != nil {
			continue
		}

		utils.LogInfo(utils.LogEntry{
			Type: "asn_lookup",
			Extra: map[string]interface{}{
				"ip":       ipStr,
				"asn":      rec.AutonomousSystemNumber,
				"provider": rec.AutonomousSystemOrganization,
			},
		})

		if containsAny(rec.AutonomousSystemOrganization, []string{"Amazon", "Cloudflare", "IPXO", "OVH"}) {
			parts := ip.To4()
			if parts != nil {
				utils.LogInfo(utils.LogEntry{
					Type: "asn_blacklist_prefix",
					Extra: map[string]interface{}{
						"ip":       ipStr,
						"prefix":   fmt.Sprintf("%d.%d.", parts[0], parts[1]),
						"provider": rec.AutonomousSystemOrganization,
						"asn":      rec.AutonomousSystemNumber,
						"reason":   "cloud_provider",
					},
				})
			}
		}
	}
}

func containsAny(s string, list []string) bool {
	for _, v := range list {
		if containsIgnoreCase(s, v) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (strings.Contains(strings.ToLower(s), strings.ToLower(substr)))
}
