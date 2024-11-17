package fibergun

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// GunMessage represents various GunDB message types
type GunMessage struct {
	Get  *GunGet `json:"get,omitempty"`
	Put  *GunPut `json:"put,omitempty"`
	Hash string  `json:"#,omitempty"`
	Ok   *GunOk  `json:"ok,omitempty"`
	Err  string  `json:"err,omitempty"`
}

// GunGet represents a GET request in GunDB
type GunGet struct {
	Hash string `json:"#,omitempty"`
	Key  string `json:".,omitempty"`
	Soul string `json:"#,omitempty"`
}

type GunPut struct {
	Chat        map[string]interface{} `json:"chat,omitempty"`
	ChatMessage map[string]interface{} `json:"chat/message,omitempty"`
	Messages    map[string]interface{} `json:"messages,omitempty"`
	Ok          map[string]interface{} `json:"ok,omitempty"`
}

// GunOk represents an acknowledgment in GunDB
type GunOk struct {
	Hash string                 `json:"@,omitempty"`
	Data map[string]interface{} `json:"/,omitempty"`
}

// PeerConnection represents a connected peer
type PeerConnection struct {
	ID       string
	Conn     *websocket.Conn
	LastSeen time.Time
	Store    *sync.Map
	Active   bool
}

// GunDB represents the main handler structure
type GunDB struct {
	config *Config
	peers  *sync.Map
}

// New creates a new GunDB middleware instance
func New(config ...*Config) fiber.Handler {
	var cfg *Config
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = ConfigDefault()
	}

	gundb := &GunDB{
		config: cfg,
		peers:  &sync.Map{},
	}

	// Start cleanup goroutine
	go gundb.cleanupInactivePeers()

	return func(c *fiber.Ctx) error {
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		if c.Path() == cfg.WebSocketEndpoint {
			if websocket.IsWebSocketUpgrade(c) {
				c.Locals("allowed", true)
				return websocket.New(gundb.handleWebSocket(), websocket.Config{
					EnableCompression: true,
					ReadBufferSize:    1024 * 1024,
					WriteBufferSize:   1024 * 1024,
				})(c)
			}
			return fiber.ErrUpgradeRequired
		}

		if c.Path() == "/" {
			return c.SendFile(cfg.StaticPath + "/index.html")
		}

		c.Locals("gundb", gundb)
		return c.Next()
	}
}

func (g *GunDB) cleanupInactivePeers() {
	ticker := time.NewTicker(g.config.HeartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		g.peers.Range(func(key, value interface{}) bool {
			peer := value.(*PeerConnection)
			if !peer.Active || now.Sub(peer.LastSeen) > g.config.PeerTimeout {
				log.Printf("Cleaning up inactive peer: %s", peer.ID)
				g.peers.Delete(key)
				peer.Conn.Close()
			}
			return true
		})
	}
}

func (g *GunDB) handleWebSocket() func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		peerID := fmt.Sprintf("peer_%d", time.Now().UnixNano())
		peer := &PeerConnection{
			ID:       peerID,
			Conn:     c,
			LastSeen: time.Now(),
			Store:    &sync.Map{},
			Active:   true,
		}

		g.peers.Store(peerID, peer)
		log.Printf("New peer connected: %s", peerID)

		g.sendAck(c, peerID)

		defer func() {
			peer.Active = false
			g.peers.Delete(peerID)
			c.Close()
			log.Printf("Peer disconnected: %s", peerID)
		}()

		// Start heartbeat
		go g.maintainHeartbeat(peer)

		for {
			messageType, msg, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Error reading message from peer %s: %v", peerID, err)
				}
				break
			}

			peer.LastSeen = time.Now()
			if messageType == websocket.TextMessage {
				g.handleMessage(peer, msg)
			}
		}
	}
}

func (g *GunDB) maintainHeartbeat(peer *PeerConnection) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !peer.Active {
			return
		}
		if err := peer.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			peer.Active = false
			return
		}
	}
}

// Add this at the package level
var globalStore = &sync.Map{}

func (g *GunDB) handleMessage(peer *PeerConnection, msg []byte) {
	log.Printf("Raw message from peer %s: %s", peer.ID, string(msg))

	// Try to parse as array first
	var messages []json.RawMessage
	if err := json.Unmarshal(msg, &messages); err == nil {
		for _, rawMsg := range messages {
			// Look for PUT message in array
			var message map[string]interface{}
			if err := json.Unmarshal(rawMsg, &message); err == nil {
				if putData, ok := message["put"]; ok {
					log.Printf("Found PUT in array: %v", putData)
					// Create GunMessage and process
					gunMsg := GunMessage{
						Put: &GunPut{
							Chat: putData.(map[string]interface{}),
						},
						Hash: message["#"].(string),
					}
					g.processGunMessage(peer, &gunMsg, rawMsg)
				}
			}
			// Also try as GunMessage
			var arrayMsg GunMessage
			if err := json.Unmarshal(rawMsg, &arrayMsg); err == nil {
				g.processGunMessage(peer, &arrayMsg, rawMsg)
			}
		}
		return
	}

	// Try as single message if not array
	var gunMsg GunMessage
	if err := json.Unmarshal(msg, &gunMsg); err == nil {
		g.processGunMessage(peer, &gunMsg, msg)
	}
}

// Update processGunMessage to handle rawMsg
func (g *GunDB) processGunMessage(peer *PeerConnection, msg *GunMessage, rawMsg []byte) {
	log.Printf("Processing message from peer %s: %+v", peer.ID, msg)

	if msg.Get != nil {
		g.handleGet(peer, msg)
	}
	if msg.Put != nil {
		log.Printf("Found PUT message: %+v", msg.Put)
		g.handlePut(peer, msg, rawMsg)
	}
	if msg.Ok != nil {
		log.Printf("OK acknowledgment from %s: %+v", peer.ID, msg.Ok)
	}
}

// Remove the standalone broadcast function and update handlePut to handle broadcasting directly
func (g *GunDB) handlePut(peer *PeerConnection, msg *GunMessage, rawMsg []byte) {
	log.Printf("Processing PUT from peer %s: %+v", peer.ID, msg.Put)

	// Extract data from the nested structure
	if msg.Put.Chat != nil {
		for k, v := range msg.Put.Chat {
			if messageData, ok := v.(map[string]interface{}); ok {
				log.Printf("Storing chat message: %v", messageData)
				globalStore.Store(k, messageData)
				peer.Store.Store(k, messageData)
			}
		}
		// Broadcast to other peers
		g.broadcast(peer, rawMsg)
	}

	// Handle direct messages if present
	if msg.Put.Messages != nil {
		for k, v := range msg.Put.Messages {
			globalStore.Store(k, v)
			peer.Store.Store(k, v)
		}
		g.broadcast(peer, rawMsg)
	}

	// Send acknowledgment
	ack := map[string]interface{}{
		"#":  msg.Hash,
		"@":  msg.Hash,
		"ok": true,
	}
	g.sendJSON(peer.Conn, ack)
}

// Update handleGet to check globalStore first
func (g *GunDB) handleGet(peer *PeerConnection, msg *GunMessage) {
	log.Printf("Processing GET from peer %s: %+v", peer.ID, msg.Get)

	if msg.Get.Soul != "" {
		// Check global store first
		if data, ok := globalStore.Load(msg.Get.Soul); ok {
			response := map[string]interface{}{
				"#":   time.Now().UnixNano(),
				"@":   msg.Hash,
				"put": data,
			}
			g.sendJSON(peer.Conn, response)
			return
		}

		// Check peer stores if not in global store
		g.peers.Range(func(_, value interface{}) bool {
			if p := value.(*PeerConnection); p.Active {
				if data, ok := p.Store.Load(msg.Get.Soul); ok {
					response := map[string]interface{}{
						"#":   time.Now().UnixNano(),
						"@":   msg.Hash,
						"put": data,
					}
					g.sendJSON(peer.Conn, response)
					return false
				}
			}
			return true
		})
	}
}

func (g *GunDB) sendJSON(c *websocket.Conn, data interface{}) {
	if msg, err := json.Marshal(data); err == nil {
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("Error sending JSON: %v", err)
		}
	}
}

func (g *GunDB) sendAck(c *websocket.Conn, peerID string) {
	ack := map[string]interface{}{
		"#":   time.Now().UnixNano(),
		"ok":  true,
		"pid": peerID,
	}
	g.sendJSON(c, ack)
}

func (g *GunDB) broadcast(sender *PeerConnection, msg []byte) {
	log.Printf("Broadcasting message from peer %s: %+v", sender.ID, string(msg))

	g.peers.Range(func(_, value interface{}) bool {
		peer := value.(*PeerConnection)
		if peer.Active && peer.ID != sender.ID {
			for attempts := 0; attempts < 3; attempts++ {
				if err := peer.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					log.Printf("Error broadcasting to peer %s (attempt %d): %v", peer.ID, attempts+1, err)
					if attempts == 2 {
						peer.Active = false
					}
					time.Sleep(100 * time.Millisecond)
				} else {
					log.Printf("Successfully broadcast to peer %s", peer.ID)
					break
				}
			}
		}
		return true
	})
}

func (g *GunDB) Put(key string, value interface{}) error {
	// Store in global store first
	globalStore.Store(key, value)

	msg := GunMessage{
		Hash: time.Now().Format("20060102150405.000000000"),
		Put: &GunPut{
			Messages: map[string]interface{}{
				key: value,
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Store and broadcast to all peers
	g.peers.Range(func(_, value interface{}) bool {
		if peer := value.(*PeerConnection); peer.Active {
			// Store data
			peer.Store.Store(key, value)

			// Send with retry
			for attempts := 0; attempts < 3; attempts++ {
				if err := peer.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
					log.Printf("Error sending to peer %s (attempt %d): %v", peer.ID, attempts+1, err)
					if attempts == 2 {
						peer.Active = false
					}
					time.Sleep(100 * time.Millisecond)
				} else {
					log.Printf("Successfully sent to peer %s", peer.ID)
					break
				}
			}
		}
		return true
	})

	return nil
}

// Update Get to check globalStore first
func (g *GunDB) Get(key string) (interface{}, error) {
	// Check global store first
	if value, ok := globalStore.Load(key); ok {
		return value, nil
	}

	// Check peer stores if not found in global store
	var result interface{}
	g.peers.Range(func(_, value interface{}) bool {
		if peer := value.(*PeerConnection); peer.Active {
			if data, ok := peer.Store.Load(key); ok {
				result = data
				return false
			}
		}
		return true
	})
	return result, nil
}
