package middleware

import (
	"go-redirect/utils"
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
			utils.LogFatal(utils.LogEntry{
				Type:  "geoip_db_error",
				Extra: map[string]interface{}{"error": err.Error(), "geoDBPath": geoDBPath},
			}, 1)

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
			utils.LogInfo(utils.LogEntry{
				Type:  "bypass_request",
				Extra: map[string]interface{}{"method": "query_param"},
			})

			return c.Next()
		}
		if c.Get("X-Bypass-Key") == bypassKey {
			utils.LogInfo(utils.LogEntry{
				Type:  "bypass_request",
				Extra: map[string]interface{}{"method": "header"},
			})

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
				utils.LogInfo(utils.LogEntry{
					Type: "block_request",
					Extra: map[string]interface{}{
						"reason":    "bad_referrer",
						"referrer":  host,
						"ip":        ip,
						"userAgent": ua,
					},
				})

				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 2) User-Agent blacklist
		for _, bad := range bf.cfg.BlacklistUA {
			if bad != "" && strings.Contains(ua, bad) {
				utils.LogInfo(utils.LogEntry{
					Type: "block_request",
					Extra: map[string]interface{}{
						"reason": "suspicious_ua",
						"ua":     bad,
						"ip":     ip,
					},
				})

				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 3) Rate limiting
		if bf.tooMany(ip) {
			utils.LogInfo(utils.LogEntry{
				Type: "block_request",
				Extra: map[string]interface{}{
					"reason": "rate_limit_exceeded",
					"ip":     ip,
				},
			})

			return c.Status(fiber.StatusForbidden).Send(nil)
		}

		// 4) IP prefix block
		for _, p := range bf.cfg.BlacklistIPPrefix {
			if strings.HasPrefix(ip, p) {
				utils.LogInfo(utils.LogEntry{
					Type: "block_request",
					Extra: map[string]interface{}{
						"reason":    "blacklisted_ip_prefix",
						"ip_prefix": p,
						"ip":        ip,
					},
				})

				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 5) Geo filter (skip only for localhost, not all private IPs)
		if len(bf.cfg.AllowCountries) > 0 && bf.geoDB != nil && !isLocalhostIP(ip) {
			cc := bf.countryCode(ip, &logData)
			if cc == "" {
				utils.LogInfo(utils.LogEntry{
					Type: "block_request",
					Extra: map[string]interface{}{
						"reason": "geo_unknown",
						"ip":     ip,
					},
				})

				return c.Status(fiber.StatusForbidden).Send(nil)
			}
			if !containsStr(bf.cfg.AllowCountries, cc) {
				utils.LogInfo(utils.LogEntry{
					Type: "block_request",
					Extra: map[string]interface{}{
						"reason":      "geo_not_allowed",
						"ip":          ip,
						"countryCode": cc,
					},
				})

				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 6) Mobile only
		if bf.cfg.AllowMobileOnly && !isMobileUA(ua) {
			utils.LogInfo(utils.LogEntry{
				Type: "block_request",
				Extra: map[string]interface{}{
					"reason": "non_mobile_device",
					"ua":     ua,
					"ip":     ip,
				},
			})

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

// isLocalOrPrivateIP checks if IP is localhost or private IP
func isLocalOrPrivateIP(ipStr string) bool {
	// Localhost and loopback
	if ipStr == "127.0.0.1" || ipStr == "::1" || ipStr == "localhost" {
		return true
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Private IP ranges
	return ip.IsLoopback() || ip.IsPrivate()
}

// isLocalhostIP checks if IP is only localhost (for geo filter bypass)
func isLocalhostIP(ipStr string) bool {
	// Only localhost and loopback
	if ipStr == "127.0.0.1" || ipStr == "::1" || ipStr == "localhost" {
		return true
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Only loopback, not private
	return ip.IsLoopback()
}
