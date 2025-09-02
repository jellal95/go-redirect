package handlers

import (
	"fmt"
	"go-redirect/models"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

var PostbackLogs []map[string]string

var PropAdsConfig models.PropAds

func PostbackHandler(c *fiber.Ctx) error {
	data := map[string]string{}
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		data[string(k)] = string(v)
	})

	data["timestamp"] = time.Now().Format(time.RFC3339)
	PostbackLogs = append(PostbackLogs, data)
	log.Println("Postback:", data)

	subID := data["sub_id"]
	payout := data["payout"]

	go ForwardPostbackToPropeller(subID, payout)

	return c.JSON(fiber.Map{
		"status": "ok",
		"data":   data,
	})
}

func GetPostbacks(c *fiber.Ctx) error {
	return c.JSON(PostbackLogs)
}

func ForwardPostbackToPropeller(subID, payout string) {
	if subID == "" {
		return
	}

	url := fmt.Sprintf(
		"%s?aid=%s&tid=%s&visitor_id=%s&payout=%s",
		PropAdsConfig.PostbackURL,
		PropAdsConfig.Aid,
		PropAdsConfig.Tid,
		subID,
		payout,
	)

	_, err := http.Get(url)
	if err != nil {
		log.Println("Failed to send postback to PropellerAds:", err)
	} else {
		log.Println("Forwarded postback to PropellerAds for subID:", subID)
	}
}
