package main

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

func main() {
	asnDB, err := geoip2.Open("GeoLite2-ASN.mmdb")
	if err != nil {
		log.Fatal(err)
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
		fmt.Printf("%s -> ASN %d (%s)\n", ipStr, rec.AutonomousSystemNumber, rec.AutonomousSystemOrganization)

		// Simple rule: kalau provider mengandung AWS / Cloudflare / IPXO → block prefix
		if containsAny(rec.AutonomousSystemOrganization, []string{"Amazon", "Cloudflare", "IPXO", "OVH"}) {
			// ambil 2 oktet pertama → jadi prefix
			parts := ip.To4()
			if parts != nil {
				fmt.Printf("Blacklist prefix: \"%d.%d.\"\n", parts[0], parts[1])
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
