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

	target, err := svc.Resolve(c.Context(), code, linkCredential(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "short link not found",
				"code":  code,
			})
		case errors.Is(err, service.ErrExpired):
			return c.Status(fiber.StatusGone).JSON(fiber.Map{
				"error": "short link has expired",
				"code":  code,
			})
		case errors.Is(err, service.ErrPasswordRequired):
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "password required",
				"code":  code,
				"hint":  "send header X-Link-Password or query password=",
			})
		case errors.Is(err, service.ErrPasswordInvalid):
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid password",
				"code":  code,
			})
		default:
			return err
		}
	}

	svc.RecordClick(c.Context(), code)

	return c.Redirect().Status(fiber.StatusFound).To(target)
}
