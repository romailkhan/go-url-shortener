package routes

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type shortenRequest struct {
	URL         string `json:"url"`
	CustomAlias string `json:"custom_alias,omitempty"`
}

// MountShorten registers URL shortening API routes on the given router (e.g. /api/v1 group).
func MountShorten(r fiber.Router) {
	r.Post("/shorten", shortenCreate)
}

// shortenCreate accepts a long URL and will persist a short code once storage is wired.
func shortenCreate(c fiber.Ctx) error {
	var req shortenRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body")
	}

	raw := strings.TrimSpace(req.URL)
	if raw == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url is required")
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url must include scheme and host")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fiber.NewError(fiber.StatusBadRequest, "only http and https URLs are allowed")
	}

	// TODO: persist mapping via repository / service; return 201 with short_url + code.
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error": "short link persistence is not implemented yet",
		"url":   u.String(),
	})
}
