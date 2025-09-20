package handlers

import (
	"go-redirect/models"
	"go-redirect/utils"
	"net/http"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
)

var PostbackLogs []map[string]string
var PropellerConfig models.Propeller
var GalaksionConfig models.Galaksion
var PopcashConfig models.Popcash
var ClickAdillaConfig models.ClickAdilla

// --- Public Endpoints ---
func GetPostbacks(c *fiber.Ctx) error {
	return c.JSON(PostbackLogs)
}

func PostbackHandler(c *fiber.Ctx) error {
	data := map[string]string{}
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		data[string(k)] = string(v)
	})

	data["timestamp"] = time.Now().Format(time.RFC3339)
	PostbackLogs = append(PostbackLogs, data)

	utils.LogInfo(utils.LogEntry{
		Type:  "postback_received",
		Extra: stringMapToInterfaceMap(data),
	})

	subID := data["sub_id"]
	payout := data["payout"]
	typeAds := data["type_ads"]

	switch typeAds {
	case models.AdTypePropeller:
		go forwardPostbackWithBreaker("propeller", "PropellerAds", subID, payout, PropellerConfig.PostbackURL, map[string]string{
			"aid":        PropellerConfig.Aid,
			"tid":        PropellerConfig.Tid,
			"visitor_id": subID,
			"payout":     payout,
		})
	case models.AdTypeGalaksion:
		go forwardPostbackWithBreaker("galaksion", "Galaksion", subID, "", GalaksionConfig.PostbackURL, map[string]string{
			"cid":      GalaksionConfig.Cid,
			"click_id": subID,
		})
	case models.AdTypePopcash:
		go forwardPostbackWithBreaker("popcash", "Popcash", subID, payout, PopcashConfig.PostbackURL, map[string]string{
			"aid":     PopcashConfig.Aid,
			"type":    PopcashConfig.Type,
			"clickid": subID,
			"payout":  payout,
		})
	case models.AdTypeClickAdilla:
		go forwardPostbackWithBreaker("clickadilla", "ClickAdilla", subID, payout, ClickAdillaConfig.PostbackURL, map[string]string{
			"token":       ClickAdillaConfig.Token,
			"campaign_id": data["campaign_id"],
			"click_id":    subID,
			"payout":      payout,
		})
	default:
		utils.LogInfo(utils.LogEntry{
			Type: "postback_unknown_type",
			Extra: map[string]interface{}{
				"type_ads": typeAds,
				"data":     stringMapToInterfaceMap(data),
			},
		})
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"data":   data,
	})
}

// --- Forward Helper with Circuit Breaker ---
func forwardPostbackWithBreaker(networkKey, product, subID, payout, baseURL string, params map[string]string) {
	if subID == "" {
		utils.LogInfo(utils.LogEntry{
			Type: "postback_error",
			Extra: map[string]interface{}{
				"product": product,
				"reason":  "missing_subID",
				"payout":  payout,
			},
		})
		return
	}

	if baseURL == "" {
		utils.LogInfo(utils.LogEntry{
			Type: "postback_error",
			Extra: map[string]interface{}{
				"product": product,
				"reason":  "missing_postback_url",
			},
		})
		return
	}

	q := url.Values{}
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}

	fullURL := baseURL
	if len(q) > 0 {
		if containsQuery(baseURL) {
			fullURL += "&" + q.Encode()
		} else {
			fullURL += "?" + q.Encode()
		}
	}

	// Simple HTTP request without circuit breaker
	resp, err := http.Get(fullURL)
	if err != nil {
		utils.LogInfo(utils.LogEntry{
			Type: "postback_forward_error",
			Extra: map[string]interface{}{
				"product": product,
				"sub_id":  subID,
				"fullURL": fullURL,
				"error":   err.Error(),
				"network": networkKey,
			},
		})
		return
	}
	defer resp.Body.Close()

	// Check for HTTP error status codes
	if resp.StatusCode >= 400 {
		utils.LogInfo(utils.LogEntry{
			Type: "postback_forward_error",
			Extra: map[string]interface{}{
				"product":     product,
				"sub_id":      subID,
				"fullURL":     fullURL,
				"status_code": resp.StatusCode,
				"network":     networkKey,
			},
		})
		return
	}

	utils.LogInfo(utils.LogEntry{
		Type: "postback_forwarded",
		Extra: map[string]interface{}{
			"product":     product,
			"sub_id":      subID,
			"fullURL":     fullURL,
			"params":      stringMapToInterfaceMap(params),
			"status_code": resp.StatusCode,
		},
	})
}

// --- Helper Functions ---
func stringMapToInterfaceMap(m map[string]string) map[string]interface{} {
	res := make(map[string]interface{}, len(m))
	for k, v := range m {
		res[k] = v
	}
	return res
}

func containsQuery(u string) bool {
	return len(u) > 0 && (u[len(u)-1] == '?' || u[len(u)-1] == '&' || contains(u, "?"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (substr != "" && s != "")
}
