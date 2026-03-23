package routes

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"url-shortener/internal/service"
)

// MountLinkLookup registers link metadata API routes on the given router.
func MountLinkLookup(r fiber.Router, svc *service.Shortener) {
	r.Get("/links/:code", func(c fiber.Ctx) error {
		return linkMetadata(c, svc)
	})
}

func linkMetadata(c fiber.Ctx, svc *service.Shortener) error {
	code := strings.TrimSpace(c.Params("code"))
	if code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code is required")
	}

	res, err := svc.Lookup(c.Context(), code, linkCredential(c))
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
				"error":     "password required",
				"code":      code,
				"hint":      "send header X-Link-Password or query password=",
				"protected": true,
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

	out := fiber.Map{
		"code":               res.Code,
		"short_path":         res.ShortPath,
		"short_url":          publicShortURL(c, res.ShortPath),
		"click_count":        res.ClickCount,
		"password_protected": res.PasswordProtected,
		"created_at":         res.CreatedAt.UTC().Format(time.RFC3339),
	}
	if res.ExpiresAt != nil {
		out["expires_at"] = res.ExpiresAt.UTC().Format(time.RFC3339)
	} else {
		out["expires_at"] = nil
	}
	if res.TargetURL != "" {
		out["target_url"] = res.TargetURL
	}

	return c.JSON(out)
}
