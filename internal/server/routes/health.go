package routes

import "github.com/gofiber/fiber/v3"

func MountHealth(app *fiber.App) {
	app.Get("/health", health)
}

func health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}
