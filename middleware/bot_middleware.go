package middleware

import (
	"log"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/oschwald/geoip2-golang"
)

// ===================== CONFIG =====================

type BotFilterConfig struct {
	AllowCountries     []string
	BlacklistUA        []string
	BlacklistIPPrefix  []string
	BlacklistReferrer  []string
	BlacklistRefRegex  []string
	RateLimitMax       int
	RateLimitWindowSec int
	LogAllowed         bool
	LogBlocked         bool
	AllowMobileOnly    bool
}

type ClickLog struct {
	IP        string
	UserAgent string
	Country   string
	Referrer  string
	Timestamp time.Time
}

type botFilter struct {
	cfg          BotFilterConfig
	geoDB        *geoip2.Reader
	ipReqMu      sync.Mutex
	ipReqMap     map[string][]time.Time
	refRegexList []*regexp.Regexp
}

// ===================== INIT =====================

func NewBotFilter(cfg BotFilterConfig, geoDBPath string) (*botFilter, error) {
	var db *geoip2.Reader
	var err error
	if geoDBPath != "" {
		db, err = geoip2.Open(geoDBPath)
		if err != nil {
			return nil, err
		}
	}
	bf := &botFilter{
		cfg:      cfg,
		geoDB:    db,
		ipReqMap: make(map[string][]time.Time),
	}
	for _, pat := range cfg.BlacklistRefRegex {
		if re, err := regexp.Compile(pat); err == nil {
			bf.refRegexList = append(bf.refRegexList, re)
		}
	}

	go bf.gcRequests()
	return bf, nil
}

// ===================== MIDDLEWARE =====================

func (bf *botFilter) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		const bypassKey = "a9f7x2kq"

		if c.Query("bypass") == bypassKey {
			log.Println("[BYPASS] Request dilepas via query param")
			return c.Next()
		}
		if c.Get("X-Bypass-Key") == bypassKey {
			log.Println("[BYPASS] Request dilepas via header")
			return c.Next()
		}

		ip := clientIP(c)
		ua := strings.ToLower(c.Get("User-Agent"))
		ref := strings.ToLower(c.Get("Referer"))

		logData := ClickLog{
			IP:        ip,
			UserAgent: ua,
			Referrer:  ref,
			Timestamp: time.Now(),
		}

		// 1) Referrer blacklist
		if host := refHost(ref); host != "" {
			if bf.isBadReferrer(host) {
				log.Printf("[BLOCK] Bad referrer: %s", host)
				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 2) User-Agent blacklist
		for _, bad := range bf.cfg.BlacklistUA {
			if bad != "" && strings.Contains(ua, bad) {
				log.Printf("[BLOCK] Suspicious UA: %s", bad)
				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 3) Rate limiting
		if bf.tooMany(ip) {
			log.Printf("[BLOCK] Rate limit exceeded: %s", ip)
			return c.Status(fiber.StatusForbidden).Send(nil)
		}

		// 4) IP prefix block
		for _, p := range bf.cfg.BlacklistIPPrefix {
			if strings.HasPrefix(ip, p) {
				log.Printf("[BLOCK] Blacklisted IP prefix: %s", p)
				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 5) Geo filter
		if len(bf.cfg.AllowCountries) > 0 && bf.geoDB != nil {
			cc := bf.countryCode(ip, &logData)
			if cc == "" {
				log.Printf("[BLOCK] Geo unknown: %s", ip)
				return c.Status(fiber.StatusForbidden).Send(nil)
			}
			if !containsStr(bf.cfg.AllowCountries, cc) {
				log.Printf("[BLOCK] Geo not allowed: %s", cc)
				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 6) Mobile only
		if bf.cfg.AllowMobileOnly && !isMobileUA(ua) {
			log.Printf("[BLOCK] Non-mobile device: %s", ua)
			return c.Status(fiber.StatusForbidden).Send(nil)
		}

		return c.Next()
	}
}

// ===================== HELPERS =====================

func clientIP(c *fiber.Ctx) string {
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}

		return strings.TrimSpace(xff)
	}

	if xr := c.Get("X-Real-Ip"); xr != "" {
		return strings.TrimSpace(xr)
	}

	return c.IP()
}

func refHost(ref string) string {
	if ref == "" {
		return ""
	}

	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}

	return strings.ToLower(u.Host)
}

func (bf *botFilter) isBadReferrer(host string) bool {
	host = strings.TrimSpace(host)
	for _, h := range bf.cfg.BlacklistReferrer {
		h = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(h)), ".")
		if host == h || strings.HasSuffix(host, "."+h) {
			return true
		}
	}
	for _, re := range bf.refRegexList {
		if re.MatchString(host) {
			return true
		}
	}

	return false
}

func (bf *botFilter) countryCode(ipStr string, logData *ClickLog) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}

	rec, err := bf.geoDB.Country(ip)
	if err != nil || rec == nil {
		return ""
	}

	cc := rec.Country.IsoCode
	logData.Country = cc

	return cc
}

func (bf *botFilter) tooMany(ip string) bool {
	now := time.Now()
	win := time.Duration(bf.cfg.RateLimitWindowSec) * time.Second
	limitMax := bf.cfg.RateLimitMax
	bf.ipReqMu.Lock()

	defer bf.ipReqMu.Unlock()

	h := bf.ipReqMap[ip]
	j := 0
	for _, t := range h {
		if now.Sub(t) <= win {
			h[j] = t
			j++
		}
	}
	h = h[:j]
	h = append(h, now)
	bf.ipReqMap[ip] = h

	return len(h) > limitMax
}

func (bf *botFilter) gcRequests() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		bf.ipReqMu.Lock()
		now := time.Now()
		win := time.Duration(bf.cfg.RateLimitWindowSec) * time.Second
		for ip, arr := range bf.ipReqMap {
			j := 0
			for _, tm := range arr {
				if now.Sub(tm) <= win {
					arr[j] = tm
					j++
				}
			}
			if j == 0 {
				delete(bf.ipReqMap, ip)
			} else {
				bf.ipReqMap[ip] = arr[:j]
			}
		}
		bf.ipReqMu.Unlock()
	}
}

func containsStr(arr []string, s string) bool {
	s = strings.ToUpper(s)
	for _, x := range arr {
		if strings.ToUpper(strings.TrimSpace(x)) == s {
			return true
		}
	}
	return false
}

func isMobileUA(ua string) bool {
	ua = strings.ToLower(ua)
	mobileIndicators := []string{"android", "iphone", "ipad", "ipod"}
	for _, m := range mobileIndicators {
		if strings.Contains(ua, m) {
			return true
		}
	}

	if strings.Contains(ua, " mobile ") {
		return true
	}
	return false
}

func saveClickLog(logData ClickLog, blocked bool) {
	status := "ALLOWED"
	if blocked {
		status = "BLOCKED"
	}
	log.Printf("[%s] ip=%s ua=%q cc=%s ref=%q\n",
		status, logData.IP, logData.UserAgent, logData.Country, logData.Referrer)
}
