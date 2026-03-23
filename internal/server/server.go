package server

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"url-shortener/internal/config"
	"url-shortener/internal/server/routes"
	"url-shortener/internal/service"
)

// Server wraps the Fiber application and listen configuration.
type Server struct {
	App  *fiber.App
	Port string
}

// New builds the Fiber app and registers routes.
func New(shortener *service.Shortener) (*Server, error) {
	port := config.GetPort()
	if port == "" {
		return nil, fmt.Errorf("PORT is not configured")
	}

	app := fiber.New(fiber.Config{
		AppName:      "url-shortener",
		ServerHeader: "url-shortener",
	})

	routes.MountHealth(app)
	api := app.Group("/api/v1")
	routes.MountShorten(api, shortener)
	routes.MountLinkLookup(api, shortener)
	routes.MountPublicRedirects(app, shortener)

	return &Server{
		App:  app,
		Port: port,
	}, nil
}

// Listen starts the HTTP server on the configured port.
func (s *Server) Listen() error {
	return s.App.Listen(":" + s.Port)
}
