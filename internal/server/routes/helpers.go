package routes

import "github.com/gofiber/fiber/v3"

// publicShortURL builds an absolute URL for a path using the incoming request (supports X-Forwarded-*).
func publicShortURL(c fiber.Ctx, path string) string {
	scheme := c.Protocol()
	if xf := c.Get("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	}
	host := c.Host()
	if host == "" {
		return path
	}
	return scheme + "://" + host + path
}
