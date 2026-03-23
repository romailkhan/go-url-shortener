package routes

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"url-shortener/internal/service"
)

type shortenRequest struct {
	URL         string `json:"url"`
	CustomAlias string `json:"custom_alias,omitempty"`
	Password    string `json:"password,omitempty"`
	ExpiresIn   string `json:"expires_in,omitempty"` // Go duration, e.g. "24h", "168h"
}

// MountShorten registers URL shortening API routes on the given router (e.g. /api/v1 group).
func MountShorten(r fiber.Router, svc *service.Shortener) {
	r.Post("/shorten", func(c fiber.Ctx) error {
		return shortenCreate(c, svc)
	})
}

func shortenCreate(c fiber.Ctx, svc *service.Shortener) error {
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

	var exp time.Duration
	if strings.TrimSpace(req.ExpiresIn) != "" {
		exp, err = time.ParseDuration(strings.TrimSpace(req.ExpiresIn))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "expires_in must be a Go duration (e.g. 24h, 168h)")
		}
	}

	res, err := svc.Create(c.Context(), service.CreateInput{
		TargetURL:   u.String(),
		CustomAlias: req.CustomAlias,
		Password:    req.Password,
		ExpiresIn:   exp,
	})
	if err != nil {
		var in *service.InputError
		if errors.As(err, &in) {
			return fiber.NewError(fiber.StatusBadRequest, in.Error())
		}
		if errors.Is(err, service.ErrConflict) {
			return fiber.NewError(fiber.StatusConflict, "short code already taken")
		}
		return err
	}

	out := fiber.Map{
		"code":               res.Code,
		"target_url":         res.TargetURL,
		"short_path":         res.ShortPath,
		"short_url":          publicShortURL(c, res.ShortPath),
		"password_protected": res.PasswordProtected,
	}
	if res.ExpiresAt != nil {
		out["expires_at"] = res.ExpiresAt.UTC().Format(time.RFC3339)
	} else {
		out["expires_at"] = nil
	}

	return c.Status(fiber.StatusCreated).JSON(out)
}
