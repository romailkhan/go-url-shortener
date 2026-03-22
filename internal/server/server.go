package server

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"url-shortener/internal/config"
	"url-shortener/internal/server/routes"
)

// Server wraps the Fiber application and listen configuration.
type Server struct {
	App  *fiber.App
	Port string
}

// New builds the Fiber app, applies middleware, and registers routes.
func New() (*Server, error) {
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
	routes.MountShorten(api)
	routes.MountLinkLookup(api)
	routes.MountPublicRedirects(app)

	return &Server{
		App:  app,
		Port: port,
	}, nil
}

// Listen starts the HTTP server on the configured port.
func (s *Server) Listen() error {
	return s.App.Listen(":" + s.Port)
}
