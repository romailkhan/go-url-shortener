package routes

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

// MountLinkLookup registers link metadata API routes on the given router.
func MountLinkLookup(r fiber.Router) {
	r.Get("/links/:code", linkMetadata)
}

// linkMetadata returns metadata for a short code.
func linkMetadata(c fiber.Ctx) error {
	code := strings.TrimSpace(c.Params("code"))
	if code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code is required")
	}

	// TODO: load from store; include created_at, click stats, original URL if authorized.
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error": "link lookup is not implemented yet",
		"code":  code,
	})
}
