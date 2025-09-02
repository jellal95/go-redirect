package handlers

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

// PostbackLog nyimpen conversion dari AccessTrade
var PostbackLogs []map[string]string

// Handler buat terima postback
func PostbackHandler(c *fiber.Ctx) error {
	data := map[string]string{}

	// ambil semua query params
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) {
		data[string(k)] = string(v)
	})

	// tambahin timestamp biar ketahuan kapan masuknya
	data["timestamp"] = time.Now().Format(time.RFC3339)

	PostbackLogs = append(PostbackLogs, data)
	log.Println("Postback:", data)

	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "postback received",
		"data":    data,
	})
}

// Endpoint untuk lihat semua postback
func GetPostbacks(c *fiber.Ctx) error {
	return c.JSON(PostbackLogs)
}
