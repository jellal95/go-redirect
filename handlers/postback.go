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
var PropAdsConfig models.PropAds
var GalaksionConfig models.Galaksion

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
	case "1": // PropellerAds
		go ForwardPostbackToPropeller(subID, payout)
	case "2": // Galaksion
		go ForwardPostbackToGalaksion(subID)
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
	q.Set("aid", PropAdsConfig.Aid)
	q.Set("tid", PropAdsConfig.Tid)
	q.Set("visitor_id", subID)
	if payout != "" {
		q.Set("payout", payout)
	}
	fullURL := PropAdsConfig.PostbackURL
	if strings.Contains(fullURL, "?") {
		fullURL += "&" + q.Encode()
	} else {
		fullURL += "?" + q.Encode()
	}

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

	_, err := http.Get(fullURL)
	if err != nil {
		log.Println("Failed to send postback to Galaksion:", err)
	} else {
		log.Println("Forwarded postback to Galaksion for subID:", subID)
	}
}
