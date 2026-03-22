package main

import (
	"log"

	"github.com/gofiber/fiber/v3"

	"url-shortener/internal/config"
)

func main() {
	if _, err := config.LoadConfig(); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello, World 👋!")
	})

	log.Fatal(app.Listen(":" + config.GetPort()))
}
