package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-redirect/models"

	"github.com/gofiber/fiber/v2"
)

func setupFiber() *fiber.App {
	app := fiber.New()
	Products = []models.Product{
		{ID: "1", Name: "Eiger", URL: "https://eiger.com"},
		{ID: "2", Name: "Blibli", URL: "https://blibli.com"},
	}
	app.Get("/", RedirectHandler)
	return app
}

func TestRedirectHandler_Propeller(t *testing.T) {
	app := setupFiber()

	req := httptest.NewRequest("GET", "/?product=1&subid=prop123&type_ads=1", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "https://eiger.com") {
		t.Errorf("unexpected redirect URL: %s", loc)
	}
	if !strings.Contains(loc, "subid=prop123") {
		t.Errorf("expected subid in redirect URL, got: %s", loc)
	}
}

func TestRedirectHandler_Galaksion(t *testing.T) {
	app := setupFiber()

	req := httptest.NewRequest("GET", "/?product=2&clickid=galak456&type_ads=2", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "https://blibli.com") {
		t.Errorf("unexpected redirect URL: %s", loc)
	}
	if !strings.Contains(loc, "subid=galak456") {
		t.Errorf("expected subid from clickid, got: %s", loc)
	}
}

func TestRedirectHandler_NoTypeAds(t *testing.T) {
	app := setupFiber()

	req := httptest.NewRequest("GET", "/?product=1", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if strings.Contains(loc, "subid=") {
		t.Errorf("did not expect subid when no mapping present, got: %s", loc)
	}
}

func TestRedirectHandler_TypeAdsMappingOnly(t *testing.T) {
	app := setupFiber()

	// type_ads=2 should use clickid and forward as subid; ignore any unknown params
	req := httptest.NewRequest("GET", "/?product=2&clickid=galak456&type_ads=2&unknownParam=overrideX", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "https://blibli.com") {
		t.Errorf("unexpected redirect URL: %s", loc)
	}
	if !strings.Contains(loc, "subid=galak456") {
		t.Errorf("expected subid from clickid via type_ads mapping, got: %s", loc)
	}
}
