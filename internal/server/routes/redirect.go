package routes

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

	"url-shortener/internal/service"
)

// MountPublicRedirects registers public short-link redirect routes.
func MountPublicRedirects(app *fiber.App, svc *service.Shortener) {
	app.Get("/s/:code", func(c fiber.Ctx) error {
		return redirectByCode(c, svc)
	})
}

func redirectByCode(c fiber.Ctx, svc *service.Shortener) error {
	code := strings.TrimSpace(c.Params("code"))
	if code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code is required")
	}

	target, err := svc.Resolve(c.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "short link not found",
				"code":  code,
			})
		}
		return err
	}

	return c.Redirect().Status(fiber.StatusFound).To(target)
}
