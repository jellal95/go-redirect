package handlers

import (
	"go-redirect/models"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

var PostbackLogs []map[string]string
var PropellerConfig models.Propeller
var GalaksionConfig models.Galaksion
var PopcashConfig models.Popcash
var ClickAdillaConfig models.ClickAdilla

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
	log.Println("Postback received:", data)

	subID := data["sub_id"]
	payout := data["payout"]
	typeAds := data["type_ads"]

	switch typeAds {
	case models.AdTypePropeller:
		go ForwardPostbackToPropeller(subID, payout)
	case models.AdTypeGalaksion:
		go ForwardPostbackToGalaksion(subID)
	case models.AdTypePopcash:
		go ForwardPostbackToPopcash(subID, payout)
	case models.AdTypeClickAdilla:
		go ForwardPostbackToClickAdilla(subID, payout, data)
	default:
		log.Println("Unknown type_ads, just logging:", typeAds)
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"data":   data,
	})
}

func ForwardPostbackToPropeller(subID, payout string) {
	if subID == "" {
		log.Println("PropellerAds postback missing subID")
		return
	}

	q := url.Values{}
	q.Set("aid", PropellerConfig.Aid)
	q.Set("tid", PropellerConfig.Tid)
	q.Set("visitor_id", subID)
	if payout != "" {
		q.Set("payout", payout)
	}
	fullURL := PropellerConfig.PostbackURL
	if strings.Contains(fullURL, "?") {
		fullURL += "&" + q.Encode()
	} else {
		fullURL += "?" + q.Encode()
	}

	// === Insert Log ===
	//params := map[string]string{"sub_id": subID, "payout": payout}
	//qp, _ := json.Marshal(params)

	//entry := models.LogEntry{
	//	Type:        models.TypePostback,
	//	Timestamp:   time.Now(),
	//	ProductName: "PropellerAds",
	//	URL:         fullURL,
	//	QueryParams: qp,
	//}
	//if err := utils.DB.Create(&entry).Error; err != nil {
	//	log.Println("Failed insert Propeller postback log:", err)
	//}

	// === Forward ===
	_, err := http.Get(fullURL)
	if err != nil {
		log.Println("Failed to send postback to PropellerAds:", err)
	} else {
		log.Println("Forwarded postback to PropellerAds for subID:", subID)
	}
}

func ForwardPostbackToGalaksion(subID string) {
	if subID == "" {
		log.Println("Galaksion postback missing subID")
		return
	}

	q := url.Values{}
	q.Set("cid", GalaksionConfig.Cid)
	q.Set("click_id", subID)
	fullURL := GalaksionConfig.PostbackURL
	if strings.Contains(fullURL, "?") {
		fullURL += "&" + q.Encode()
	} else {
		fullURL += "?" + q.Encode()
	}

	// === Insert Log ===
	//params := map[string]string{"click_id": subID}
	//qp, _ := json.Marshal(params)

	//entry := models.LogEntry{
	//	Type:        models.TypePostback,
	//	Timestamp:   time.Now(),
	//	ProductName: "Galaksion",
	//	URL:         fullURL,
	//	QueryParams: qp,
	//}
	//if err := utils.DB.Create(&entry).Error; err != nil {
	//	log.Println("Failed insert Galaksion postback log:", err)
	//}

	// === Forward ===
	_, err := http.Get(fullURL)
	if err != nil {
		log.Println("Failed to send postback to Galaksion:", err)
	} else {
		log.Println("Forwarded postback to Galaksion for subID:", subID)
	}
}

func ForwardPostbackToPopcash(subID, payout string) {
	if subID == "" {
		log.Println("Popcash postback missing subID (clickid)")
		return
	}

	baseURL := PopcashConfig.PostbackURL
	if baseURL == "" {
		baseURL = "https://ct.popcash.net/click"
	}
	aid := PopcashConfig.Aid
	if aid == "" {
		aid = "494669"
	}
	typeVal := PopcashConfig.Type
	if typeVal == "" {
		typeVal = "1"
	}

	q := url.Values{}
	q.Set("aid", aid)
	q.Set("type", typeVal)
	q.Set("clickid", subID)
	if payout != "" {
		q.Set("payout", payout)
	}

	fullURL := baseURL
	if strings.Contains(fullURL, "?") {
		fullURL += "&" + q.Encode()
	} else {
		fullURL += "?" + q.Encode()
	}

	// === Insert Log ===
	//params := map[string]string{"clickid": subID, "payout": payout}
	//qp, _ := json.Marshal(params)

	//entry := models.LogEntry{
	//	Type:        models.TypePostback,
	//	Timestamp:   time.Now(),
	//	ProductName: "Popcash",
	//	URL:         fullURL,
	//	QueryParams: qp,
	//}
	//if err := utils.DB.Create(&entry).Error; err != nil {
	//	log.Println("Failed insert Popcash postback log:", err)
	//}

	// === Forward ===
	_, err := http.Get(fullURL)
	if err != nil {
		log.Println("Failed to send postback to Popcash:", err)
	} else {
		log.Println("Forwarded postback to Popcash for clickid:", subID)
	}
}

func ForwardPostbackToClickAdilla(subID, payout string, data map[string]string) {
	if subID == "" {
		log.Println("ClickAdilla postback missing click_id")
		return
	}

	baseURL := ClickAdillaConfig.PostbackURL
	if baseURL == "" {
		baseURL = "https://tracking.clickadilla.com/in/postbacks/"
	}

	q := url.Values{}
	q.Set("token", ClickAdillaConfig.Token)

	if val, ok := data["campaign_id"]; ok {
		q.Set("campaign_id", val)
	}
	q.Set("click_id", subID)

	if payout != "" {
		q.Set("payout", payout)
	}

	fullURL := baseURL
	if strings.Contains(fullURL, "?") {
		fullURL += "&" + q.Encode()
	} else {
		fullURL += "?" + q.Encode()
	}

	// === Simpan log ke DB ===
	//qp, _ := json.Marshal(data)

	//entry := models.LogEntry{
	//	Type:        models.TypePostback,
	//	Timestamp:   time.Now(),
	//	ProductName: "ClickAdilla",
	//	URL:         fullURL,
	//	QueryParams: qp,
	//}

	//if err := utils.DB.Create(&entry).Error; err != nil {
	//	log.Println("Failed insert ClickAdilla postback log:", err)
	//}

	// === Forward ke ClickAdilla ===
	_, err := http.Get(fullURL)
	if err != nil {
		log.Println("Failed to send postback to ClickAdilla:", err)
	} else {
		log.Println("Forwarded postback to ClickAdilla for click_id:", subID, "URL:", fullURL)
	}
}
