package fibergun

import (
	"sync"

	"github.com/gofiber/fiber/v2"
)

// Config defines the config for GunDB middleware
type Config struct {
	// Next defines a function to skip this middleware when returned true.
	Next func(c *fiber.Ctx) bool

	// WebSocketEndpoint is the endpoint where GunDB websocket connections will be handled
	// Default: "/gun"
	WebSocketEndpoint string

	// StaticPath is the path to serve the GunDB client files
	// Default: "./public"
	StaticPath string

	// PeerList stores active peer connections
	peers *sync.Map
}

// ConfigDefault is the default config
var ConfigDefault = &Config{
	WebSocketEndpoint: "/gun",
	StaticPath:        "./public",
	peers:             &sync.Map{},
}

// Helper function to set default config
func configDefault(config ...*Config) *Config {
	// Return default config if nothing provided
	if len(config) < 1 {
		return ConfigDefault
	}

	// Override default config
	cfg := config[0]

	// Set default values
	if cfg.WebSocketEndpoint == "" {
		cfg.WebSocketEndpoint = ConfigDefault.WebSocketEndpoint
	}
	if cfg.StaticPath == "" {
		cfg.StaticPath = ConfigDefault.StaticPath
	}
	if cfg.peers == nil {
		cfg.peers = &sync.Map{}
	}

	return cfg
}
