package routes

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

// linkCredential reads a link password from header (preferred) or query string.
func linkCredential(c fiber.Ctx) string {
	if p := strings.TrimSpace(c.Get("X-Link-Password")); p != "" {
		return p
	}
	return strings.TrimSpace(c.Query("password"))
}
