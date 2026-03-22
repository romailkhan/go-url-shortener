package routes

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

// MountPublicRedirects registers public short-link redirect routes.
func MountPublicRedirects(app *fiber.App) {
	// Public short links; keep prefix so API and static paths stay separate.
	app.Get("/s/:code", redirectByCode)
}

// redirectByCode resolves a short code and redirects to the stored URL (302 once persistence exists).
func redirectByCode(c fiber.Ctx) error {
	code := strings.TrimSpace(c.Params("code"))
	if code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code is required")
	}

	// TODO: resolve code via store; on miss return 404.
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "short link not found",
		"code":  code,
	})
}
