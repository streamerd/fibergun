package fibergun

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func BenchmarkMessageHandling(b *testing.B) {
	gundb := &GunDB{
		config: ConfigDefault(),
		peers:  &sync.Map{},
	}

	peer := &PeerConnection{
		ID:       "bench-peer",
		Store:    &sync.Map{},
		Active:   true,
		LastSeen: time.Now(),
	}

	msg := []byte(`{"put":{"chat":{"test":{"text":"hello","timestamp":1234567890}}}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gundb.handleMessage(peer, msg)
	}
}

func BenchmarkBroadcast(b *testing.B) {
	gundb := &GunDB{
		config: ConfigDefault(),
		peers:  &sync.Map{},
	}

	// Setup test peers
	for i := 0; i < 100; i++ {
		peer := &PeerConnection{
			ID:       fmt.Sprintf("peer-%d", i),
			Store:    &sync.Map{},
			Active:   true,
			LastSeen: time.Now(),
		}
		gundb.peers.Store(peer.ID, peer)
	}

	msg := []byte(`{"type":"test","content":"hello"}`)
	sender := &PeerConnection{ID: "sender", Active: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gundb.broadcast(sender, msg)
	}
}
