package fibergun

import (
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// GunMessage represents the different types of messages GunDB might send
type GunMessage struct {
	Get   interface{} `json:"get,omitempty"`
	Put   interface{} `json:"put,omitempty"`
	Hash  string      `json:"#,omitempty"`
	Soul  string      `json:"#,omitempty"`
	Key   string      `json:".,omitempty"`
	Value interface{} `json:">",omitempty"`
}

// New creates a new middleware handler
func New(config ...*Config) fiber.Handler {
	// Set default config
	cfg := configDefault(config...)

	// Return new handler
	return func(c *fiber.Ctx) error {
		// Don't execute middleware if Next returns true
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		// Handle WebSocket connections
		if c.Path() == cfg.WebSocketEndpoint {
			return websocket.New(handleWebSocket(cfg))(c)
		}

		// Serve GunDB client template on root path
		if c.Path() == "/" {
			return c.SendFile(cfg.StaticPath + "/index.html")
		}

		// Store GunDB instance in locals for use in routes
		c.Locals("gundb", &GunDB{config: cfg})

		return c.Next()
	}
}

func handleWebSocket(config *Config) func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		// Store new peer connection
		peerID := c.Query("id", "anonymous")
		config.peers.Store(peerID, c)

		log.Printf("New peer connected: %s", peerID)

		defer func() {
			config.peers.Delete(peerID)
			c.Close()
			log.Printf("Peer disconnected: %s", peerID)
		}()

		for {
			messageType, msg, err := c.ReadMessage()
			if err != nil {
				log.Printf("Error reading message from peer %s: %v", peerID, err)
				break
			}

			if messageType == websocket.TextMessage {
				// Try to unmarshal as a regular JSON object first
				var gunMsg GunMessage
				if err := json.Unmarshal(msg, &gunMsg); err != nil {
					// If that fails, try to handle it as a raw message
					var rawMsg interface{}
					if err := json.Unmarshal(msg, &rawMsg); err != nil {
						log.Printf("Error parsing message: %v", err)
						continue
					}

					// Log the raw message structure for debugging
					log.Printf("Raw message structure: %+v", rawMsg)
				} else {
					// Log the parsed message
					log.Printf("Received GunDB message from %s: %+v", peerID, gunMsg)
				}

				// Broadcast message to all peers regardless of type
				config.peers.Range(func(key, value interface{}) bool {
					if peer, ok := value.(*websocket.Conn); ok && peer != c {
						if err := peer.WriteMessage(websocket.TextMessage, msg); err != nil {
							log.Printf("Error broadcasting to peer %v: %v", key, err)
						} else {
							log.Printf("Broadcasted message to peer %v", key)
						}
					}
					return true
				})
			}
		}
	}
}

// GunDB represents a GunDB instance
type GunDB struct {
	config *Config
}

// Put stores data in GunDB
func (g *GunDB) Put(key string, value interface{}) error {
	msg := map[string]interface{}{
		"#": key,
		"put": map[string]interface{}{
			"key": value,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Broadcast to all peers
	g.config.peers.Range(func(key, value interface{}) bool {
		if peer, ok := value.(*websocket.Conn); ok {
			peer.WriteMessage(websocket.TextMessage, data)
		}
		return true
	})

	return nil
}

// Get retrieves data from GunDB
func (g *GunDB) Get(key string) (interface{}, error) {
	// Implementation would depend on your storage backend
	return nil, nil
}
