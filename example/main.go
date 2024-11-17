package main

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/streamerd/fibergun"
)

func main() {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
	})

	// Add middleware
	app.Use(logger.New(logger.Config{
		Format:     "${time} ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "15:04:05",
		TimeZone:   "Local",
	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "*",
	}))

	// Initialize GunDB middleware with complete config
	app.Use(fibergun.New(&fibergun.Config{
		StaticPath:        "./public",
		WebSocketEndpoint: "/gun",
		HeartbeatInterval: 15 * time.Second,
		PeerTimeout:       60 * time.Second,
		EnableCompression: true,
		BufferSize:        1024 * 16,
		Debug:             true,
		DataReplication: fibergun.DataReplicationConfig{
			Enabled:      true,
			SyncInterval: 5 * time.Second, // Faster sync for testing
			MaxRetries:   5,
			BatchSize:    100,
		},
	}))

	app.Static("/", "./public")

	log.Printf("Starting server on :3000...")
	log.Fatal(app.Listen(":3000"))
}
