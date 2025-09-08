package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"go-redirect/models"

	"github.com/gofiber/fiber/v2"
)

func setupFiber() *fiber.App {
	app := fiber.New()
	Products = []models.Product{
		{
			ID:   "1",
			Name: "Eiger",
			URL:  "https://eiger.com?sub_id1={siteid}&sub_id2={sub_id}&sub_id3={type_ads}&sub_aff_id={sub_id}",
		},
		{
			ID:   "2",
			Name: "Blibli",
			URL:  "https://blibli.com?sub_id_1={siteid}&sub_id_2={sub_id}&sub_id_3={type_ads}&sub_aff_id={sub_id}",
		},
	}
	app.Get("/", RedirectHandler)
	return app
}

func TestRedirectHandler(t *testing.T) {
	app := setupFiber()

	tests := []struct {
		name       string
		url        string
		expectSubs []string
		notExpect  []string
	}{
		{
			name: "With sub_id + type_ads + siteid",
			url:  "/?product=1&sub_id=prop123&type_ads=1&siteid=AFF111",
			expectSubs: []string{
				"sub_id2=prop123",
				"sub_id3=1",
				"sub_id1=AFF111",
				"sub_aff_id=prop123",
			},
		},
		{
			name: "No type_ads (only sub_id + siteid)",
			url:  "/?product=2&sub_id=mainX&siteid=AFF222",
			expectSubs: []string{
				"sub_id_2=mainX",
				"sub_aff_id=mainX",
			},
			notExpect: []string{
				"sub_id3=", // jangan ada kosong
			},
		},
		{
			name: "With type_ads only (no siteid)",
			url:  "/?product=1&sub_id=abc999&type_ads=3",
			expectSubs: []string{
				"sub_id2=abc999",
				"sub_id3=3",
				"sub_aff_id=abc999",
			},
		},
		{
			name: "With sub_id_1 alias and type_ads",
			url:  "/?product=1&sub_id=CLICK123&sub_id_1=AFF111&type_ads=2",
			expectSubs: []string{
				"sub_id1=AFF111",
				"sub_id2=CLICK123",
				"sub_id3=2",
				"sub_aff_id=CLICK123",
			},
		},
		{
			name: "Extra utm params",
			url:  "/?product=2&sub_id=MAIN999&utm_campaign=CAMP88&utm_source=Tiktok",
			expectSubs: []string{
				"sub_id_2=MAIN999",
				"sub_aff_id=MAIN999",
				"utm_campaign=CAMP88",
				"utm_source=Tiktok",
			},
			notExpect: []string{
				"sub_id3=", // gak ada type_ads, jangan kosong
			},
		},
		{
			name: "Fallback when only sub_id",
			url:  "/?product=1&sub_id=ONLYID",
			expectSubs: []string{
				"sub_id2=ONLYID",
				"sub_aff_id=ONLYID",
			},
			notExpect: []string{
				"sub_id1=",
				"sub_id3=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("[%s] app.Test failed: %v", tt.name, err)
			}
			if resp == nil {
				t.Fatalf("[%s] got nil response", tt.name)
			}

			if resp.StatusCode != 302 {
				t.Errorf("[%s] expected 302, got %d", tt.name, resp.StatusCode)
			}
			loc := resp.Header.Get("Location")

			for _, exp := range tt.expectSubs {
				if !strings.Contains(loc, exp) {
					t.Errorf("[%s] expected %s in redirect URL, got: %s", tt.name, exp, loc)
				}
			}
			for _, ne := range tt.notExpect {
				if strings.Contains(loc, ne) {
					t.Errorf("[%s] did not expect %s in redirect URL, got: %s", tt.name, ne, loc)
				}
			}
		})
	}
}
