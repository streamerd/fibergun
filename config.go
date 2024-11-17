package fibergun

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Config defines the config for GunDB middleware
type Config struct {
	// Next defines a function to skip this middleware when returned true
	Next func(c *fiber.Ctx) bool

	// WebSocketEndpoint is the endpoint where GunDB websocket connections will be handled
	WebSocketEndpoint string

	// StaticPath is the path to serve the GunDB client files
	StaticPath string

	// HeartbeatInterval is the interval for sending heartbeat pings
	HeartbeatInterval time.Duration

	// PeerTimeout is the duration after which a peer is considered inactive
	PeerTimeout time.Duration

	// MaxMessageSize is the maximum size of a WebSocket message in bytes
	MaxMessageSize int64

	// EnableCompression enables WebSocket compression
	EnableCompression bool

	// BufferSize sets the read/write buffer size for WebSocket connections
	BufferSize int

	// Debug enables detailed logging
	Debug bool

	// ReconnectAttempts is the number of times to attempt reconnection
	ReconnectAttempts int

	// ReconnectInterval is the time to wait between reconnection attempts
	ReconnectInterval time.Duration

	// DataReplication configures how data is replicated between peers
	DataReplication DataReplicationConfig

	// PeerList stores active peer connections (internal use)
	peers *sync.Map

	SharedStore *sync.Map
}

// DataReplicationConfig defines how data is replicated between peers
type DataReplicationConfig struct {
	// Enabled determines if data should be replicated between peers
	Enabled bool

	// SyncInterval is how often to sync data between peers
	SyncInterval time.Duration

	// MaxRetries is the maximum number of sync retries
	MaxRetries int

	// BatchSize is the maximum number of items to sync at once
	BatchSize int
}

// defaultConfig is the default config (private)
var defaultConfig = &Config{
	WebSocketEndpoint: "/gun",
	StaticPath:        "./public",
	HeartbeatInterval: 15 * time.Second,
	PeerTimeout:       60 * time.Second,
	MaxMessageSize:    1024 * 1024, // 1MB
	EnableCompression: true,
	BufferSize:        1024 * 16, // 16KB
	Debug:             false,
	ReconnectAttempts: 5,
	ReconnectInterval: 2 * time.Second,
	DataReplication: DataReplicationConfig{
		Enabled:      true,
		SyncInterval: 30 * time.Second,
		MaxRetries:   3,
		BatchSize:    100,
	},
	peers: &sync.Map{},
}

// NewConfig creates a new config with default values
func NewConfig() *Config {
	return &Config{
		WebSocketEndpoint: defaultConfig.WebSocketEndpoint,
		StaticPath:        defaultConfig.StaticPath,
		HeartbeatInterval: defaultConfig.HeartbeatInterval,
		PeerTimeout:       defaultConfig.PeerTimeout,
		MaxMessageSize:    defaultConfig.MaxMessageSize,
		EnableCompression: defaultConfig.EnableCompression,
		BufferSize:        defaultConfig.BufferSize,
		Debug:             defaultConfig.Debug,
		ReconnectAttempts: defaultConfig.ReconnectAttempts,
		ReconnectInterval: defaultConfig.ReconnectInterval,
		DataReplication:   defaultConfig.DataReplication,
		peers:             &sync.Map{},
	}
}

func ConfigDefault() *Config {
	return &Config{
		WebSocketEndpoint: "/gun",
		StaticPath:        "./public",
		HeartbeatInterval: 15 * time.Second,
		PeerTimeout:       60 * time.Second,
		peers:             &sync.Map{},
		SharedStore:       &sync.Map{}, // Add this
	}
}

// Validate checks and corrects the configuration values
func (c *Config) Validate() *Config {
	if c.WebSocketEndpoint == "" {
		c.WebSocketEndpoint = defaultConfig.WebSocketEndpoint
	}
	if c.StaticPath == "" {
		c.StaticPath = defaultConfig.StaticPath
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = defaultConfig.HeartbeatInterval
	}
	if c.PeerTimeout <= 0 {
		c.PeerTimeout = defaultConfig.PeerTimeout
	}
	if c.MaxMessageSize <= 0 {
		c.MaxMessageSize = defaultConfig.MaxMessageSize
	}
	if c.BufferSize <= 0 {
		c.BufferSize = defaultConfig.BufferSize
	}
	if c.ReconnectAttempts <= 0 {
		c.ReconnectAttempts = defaultConfig.ReconnectAttempts
	}
	if c.ReconnectInterval <= 0 {
		c.ReconnectInterval = defaultConfig.ReconnectInterval
	}
	if c.peers == nil {
		c.peers = &sync.Map{}
	}
	return c
}

// WithHeartbeat sets the heartbeat interval
func (c *Config) WithHeartbeat(interval time.Duration) *Config {
	c.HeartbeatInterval = interval
	return c
}

// WithTimeout sets the peer timeout
func (c *Config) WithTimeout(timeout time.Duration) *Config {
	c.PeerTimeout = timeout
	return c
}

// WithDebug enables or disables debug logging
func (c *Config) WithDebug(enabled bool) *Config {
	c.Debug = enabled
	return c
}

// WithCompression enables or disables WebSocket compression
func (c *Config) WithCompression(enabled bool) *Config {
	c.EnableCompression = enabled
	return c
}

// WithBufferSize sets the WebSocket buffer size
func (c *Config) WithBufferSize(size int) *Config {
	c.BufferSize = size
	return c
}

// WithDataReplication configures data replication
func (c *Config) WithDataReplication(config DataReplicationConfig) *Config {
	c.DataReplication = config
	return c
}
