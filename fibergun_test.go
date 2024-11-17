package fibergun

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// TestSetup provides test environment setup
type TestSetup struct {
	app    *fiber.App
	config *Config
	tmpDir string
}

// setupTest creates a test environment
func setupTest(t *testing.T) *TestSetup {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "fibergun-test-*")
	assert.NoError(t, err)

	// Create test index.html
	indexPath := filepath.Join(tmpDir, "index.html")
	err = os.WriteFile(indexPath, []byte("<html><body>Test</body></html>"), 0644)
	assert.NoError(t, err)

	// Create config with test directory
	config := &Config{
		WebSocketEndpoint: "/gun",
		StaticPath:        tmpDir,
		HeartbeatInterval: 1 * time.Second,
		PeerTimeout:       5 * time.Second,
	}

	// Create app with config
	app := fiber.New()
	app.Use(New(config))

	return &TestSetup{
		app:    app,
		config: config,
		tmpDir: tmpDir,
	}
}

// cleanup removes test files
func (ts *TestSetup) cleanup() {
	os.RemoveAll(ts.tmpDir)
}

// TestNew tests middleware initialization
func TestNew(t *testing.T) {
	ts := setupTest(t)
	defer ts.cleanup()

	// Test root endpoint
	req := httptest.NewRequest("GET", "/", nil)
	resp, err := ts.app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Test WebSocket endpoint
	req = httptest.NewRequest("GET", "/gun", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")

	resp, err = ts.app.Test(req)
	assert.NoError(t, err)
	// WebSocket upgrade will return 400 in test environment
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// TestConfig tests configuration
func TestConfig(t *testing.T) {
	// Test default config
	cfg := ConfigDefault()
	assert.NotNil(t, cfg)
	assert.Equal(t, "/gun", cfg.WebSocketEndpoint)

	// Test custom config
	custom := &Config{
		WebSocketEndpoint: "/custom",
		StaticPath:        "./custom",
		HeartbeatInterval: 5 * time.Second,
	}
	validated := custom.Validate()
	assert.Equal(t, "/custom", validated.WebSocketEndpoint)
	assert.Equal(t, "./custom", validated.StaticPath)
}

// TestMessageHandling tests GunDB message processing
func TestMessageHandling(t *testing.T) {
	gundb := &GunDB{
		config: ConfigDefault(),
		peers:  &sync.Map{},
	}

	// Test PUT message
	putMsg := GunMessage{
		Put: &GunPut{
			Chat: map[string]interface{}{
				"test": map[string]interface{}{
					"text":      "hello",
					"timestamp": time.Now().Unix(),
				},
			},
		},
		Hash: "test-hash",
	}

	msgBytes, err := json.Marshal(putMsg)
	assert.NoError(t, err)

	peer := &PeerConnection{
		ID:       "test-peer",
		Store:    &sync.Map{},
		Active:   true,
		LastSeen: time.Now(),
	}

	// Process message
	gundb.handleMessage(peer, msgBytes)

	// Verify message was stored
	value, ok := globalStore.Load("test")
	assert.True(t, ok)
	assert.NotNil(t, value)
}

func TestPeerManagement(t *testing.T) {
	gundb := &GunDB{
		config: ConfigDefault(),
		peers:  &sync.Map{},
	}

	// Add test peer
	peer := &PeerConnection{
		ID:       "test-peer",
		Conn:     &websocket.Conn{}, // Use a non-nil value
		Store:    &sync.Map{},
		Active:   true,
		LastSeen: time.Now(),
	}
	gundb.peers.Store(peer.ID, peer)

	// Test peer cleanup
	peer.LastSeen = time.Now().Add(-2 * gundb.config.PeerTimeout)
	gundb.cleanupInactivePeers()

	// Verify peer was removed
	_, exists := gundb.peers.Load(peer.ID)
	assert.False(t, exists)
}

// TestBroadcast tests message broadcasting
func TestBroadcast(t *testing.T) {
	gundb := &GunDB{
		config: ConfigDefault(),
		peers:  &sync.Map{},
	}

	// Create test peers
	peers := []*PeerConnection{
		createTestPeer("peer1"),
		createTestPeer("peer2"),
	}

	for _, peer := range peers {
		gundb.peers.Store(peer.ID, peer)
	}

	// Test broadcast
	msg := []byte(`{"type":"test","content":"hello"}`)
	gundb.broadcast(peers[0], msg)

	// Verify peers are still active
	for _, peer := range peers {
		p, ok := gundb.peers.Load(peer.ID)
		assert.True(t, ok)
		assert.True(t, p.(*PeerConnection).Active)
	}
}

// TestDataReplication tests data replication between peers
func TestDataReplication(t *testing.T) {
	gundb := &GunDB{
		config: &Config{
			DataReplication: DataReplicationConfig{
				Enabled:      true,
				SyncInterval: time.Second,
				MaxRetries:   3,
				BatchSize:    100,
			},
		},
		peers: &sync.Map{},
	}

	// Create and add test peers
	peers := make([]*PeerConnection, 2)
	for i := range peers {
		peers[i] = &PeerConnection{
			ID:       fmt.Sprintf("peer-%d", i),
			Store:    &sync.Map{},
			Active:   true,
			LastSeen: time.Now(),
		}
		gundb.peers.Store(peers[i].ID, peers[i])
	}

	// Test data
	testData := map[string]interface{}{
		"key": "value",
	}

	// Store in first peer
	peers[0].Store.Store("test-key", testData)

	// Trigger replication
	err := gundb.Put("test-key", testData)
	assert.NoError(t, err)

	// Verify replication
	value, exists := peers[1].Store.Load("test-key")
	assert.True(t, exists)
	assert.Equal(t, testData, value)
}

// Helper function to create test peer
func createTestPeer(id string) *PeerConnection {
	return &PeerConnection{
		ID:       id,
		Store:    &sync.Map{},
		Active:   true,
		LastSeen: time.Now(),
	}
}

// Helper function for creating test requests
func createTestRequest(t *testing.T, method, path string, body io.Reader) *fasthttp.Request {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(path)
	req.Header.SetMethod(method)

	if body != nil {
		bodyBytes, err := io.ReadAll(body)
		assert.NoError(t, err)
		req.SetBody(bodyBytes)
	}

	return req
}
