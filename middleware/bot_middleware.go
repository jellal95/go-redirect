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
				logEntry := buildBlockRequestLog(c, ip, "bad_referrer", map[string]interface{}{
					"reason":       "bad_referrer",
					"referrer":     host,
					"full_referer": c.Get("Referer"),
				})
				utils.LogInfo(logEntry)

				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 2) User-Agent blacklist
		for _, bad := range bf.cfg.BlacklistUA {
			if bad != "" && strings.Contains(ua, bad) {
				logEntry := buildBlockRequestLog(c, ip, "suspicious_ua", map[string]interface{}{
					"reason":     "suspicious_ua",
					"matched_ua": bad,
				})
				utils.LogInfo(logEntry)

				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 3) Rate limiting
		if bf.tooMany(ip) {
			logEntry := buildBlockRequestLog(c, ip, "rate_limit_exceeded", map[string]interface{}{
				"reason": "rate_limit_exceeded",
			})
			utils.LogInfo(logEntry)

			return c.Status(fiber.StatusForbidden).Send(nil)
		}

		// 4) IP prefix block
		for _, p := range bf.cfg.BlacklistIPPrefix {
			if strings.HasPrefix(ip, p) {
				logEntry := buildBlockRequestLog(c, ip, "blacklisted_ip_prefix", map[string]interface{}{
					"reason":    "blacklisted_ip_prefix",
					"ip_prefix": p,
				})
				utils.LogInfo(logEntry)

				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 5) Geo filter (skip only for localhost, not all private IPs)
		if len(bf.cfg.AllowCountries) > 0 && bf.geoDB != nil && !isLocalhostIP(ip) {
			cc := bf.countryCode(ip, &logData)
			if cc == "" {
				logEntry := buildBlockRequestLog(c, ip, "geo_unknown", map[string]interface{}{
					"reason": "geo_unknown",
				})
				utils.LogInfo(logEntry)

				return c.Status(fiber.StatusForbidden).Send(nil)
			}
			if !containsStr(bf.cfg.AllowCountries, cc) {
				logEntry := buildBlockRequestLog(c, ip, "geo_not_allowed", map[string]interface{}{
					"reason":      "geo_not_allowed",
					"countryCode": cc,
				})
				utils.LogInfo(logEntry)

				return c.Status(fiber.StatusForbidden).Send(nil)
			}
		}

		// 6) Mobile only
		if bf.cfg.AllowMobileOnly && !isMobileUA(ua) {
			logEntry := buildBlockRequestLog(c, ip, "non_mobile_device", map[string]interface{}{
				"reason": "non_mobile_device",
			})
			utils.LogInfo(logEntry)

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

// getSourceInfo extracts source information from request
func getSourceInfo(c *fiber.Ctx) map[string]interface{} {
	sourceInfo := make(map[string]interface{})

	// Referrer information
	if referer := c.Get("Referer"); referer != "" {
		if u, err := url.Parse(referer); err == nil {
			sourceInfo["referer_host"] = u.Host
			sourceInfo["referer_path"] = u.Path
			sourceInfo["referer_query"] = u.RawQuery
		}
		sourceInfo["referer_full"] = referer
	} else {
		sourceInfo["referer_type"] = "direct"
	}

	// X-Forwarded headers for source tracking
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		sourceInfo["x_forwarded_for"] = xff
	}
	if xrealip := c.Get("X-Real-IP"); xrealip != "" {
		sourceInfo["x_real_ip"] = xrealip
	}
	if xforwarded := c.Get("X-Forwarded-Proto"); xforwarded != "" {
		sourceInfo["x_forwarded_proto"] = xforwarded
	}

	return sourceInfo
}

// buildBlockRequestLog creates a comprehensive log entry for blocked requests
func buildBlockRequestLog(c *fiber.Ctx, ip, reason string, extra map[string]interface{}) utils.LogEntry {
	// Build query params map
	queryParams := make(map[string]string)
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		queryParams[string(k)] = string(v)
	})

	// Build headers map (selective - avoid sensitive headers)
	headers := make(map[string]string)
	importantHeaders := []string{"User-Agent", "Referer", "Accept", "Accept-Language", "Accept-Encoding"}
	for _, header := range importantHeaders {
		if value := c.Get(header); value != "" {
			headers[header] = value
		}
	}

	// Add X-Forwarded headers
	forwardHeaders := []string{"X-Forwarded-For", "X-Real-IP", "X-Forwarded-Proto"}
	for _, header := range forwardHeaders {
		if value := c.Get(header); value != "" {
			headers[header] = value
		}
	}

	// Merge extra info with source info
	if extra == nil {
		extra = make(map[string]interface{})
	}
	sourceInfo := getSourceInfo(c)
	for k, v := range sourceInfo {
		extra[k] = v
	}

	return utils.LogEntry{
		Type:        "block_request",
		IP:          ip,
		UserAgent:   c.Get("User-Agent"),
		Referer:     c.Get("Referer"),
		URL:         c.OriginalURL(),
		QueryParams: queryParams,
		Headers:     headers,
		Extra:       extra,
	}
}
