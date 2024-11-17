package main

import (
	"log"

	"github.com/gofiber/contrib/fibergun"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	app := fiber.New()

	// Add logger middleware
	app.Use(logger.New())

	// Initialize GunDB middleware
	app.Use(fibergun.New(&fibergun.Config{
		StaticPath: "./public",
	}))

	// Serve static files from public directory
	app.Static("/", "./public")

	log.Fatal(app.Listen(":3000"))
}
