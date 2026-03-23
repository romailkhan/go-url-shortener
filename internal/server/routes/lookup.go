package routes

import (
	"errors"
	"strings"

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

	res, err := svc.Lookup(c.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "short link not found",
				"code":  code,
			})
		}
		return err
	}

	return c.JSON(fiber.Map{
		"code":       res.Code,
		"target_url": res.TargetURL,
		"short_path": res.ShortPath,
		"short_url":  publicShortURL(c, res.ShortPath),
	})
}
