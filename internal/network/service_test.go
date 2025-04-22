package network

import (
	"sync"
	"testing"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	config := &Config{
		ListenAddr:    "localhost:8080",
		Seeds:         []string{"seed1:8080", "seed2:8080"},
		MaxPeers:      10,
		MinPeers:      5,
		MessageBuffer: 100,
		SyncInterval:  time.Second,
	}

	service := NewService(config)
	assert.NotNil(t, service)
	assert.Equal(t, config, service.config)
	assert.NotNil(t, service.peers)
	assert.NotNil(t, service.messages)
	assert.NotNil(t, service.layer)
}

func TestStartStop(t *testing.T) {
	config := &Config{
		ListenAddr:    "localhost:8080",
		Seeds:         []string{"seed1:8080", "seed2:8080"},
		MaxPeers:      10,
		MinPeers:      5,
		MessageBuffer: 100,
		SyncInterval:  time.Second,
	}

	service := NewService(config)
	assert.NotNil(t, service)

	// Test Start
	err := service.Start()
	assert.NoError(t, err)

	// Test double start
	err = service.Start()
	assert.Error(t, err)
	assert.Equal(t, ErrServiceAlreadyRunning, err)

	// Test Stop
	err = service.Stop()
	assert.NoError(t, err)
}

func TestPeerHandling(t *testing.T) {
	config := &Config{
		ListenAddr:    "localhost:8080",
		Seeds:         []string{"seed1:8080", "seed2:8080"},
		MaxPeers:      10,
		MinPeers:      5,
		MessageBuffer: 100,
		SyncInterval:  time.Second,
	}

	service := NewService(config)
	assert.NotNil(t, service)

	// Add a peer
	peer := &Peer{
		ID:        "peer1",
		Addr:      "peer1:8080",
		Connected: true,
		LastSeen:  time.Now(),
	}

	service.AddPeer(peer)

	// Verify peer was added
	assert.Equal(t, 1, len(service.peers))
	assert.Equal(t, peer, service.peers[peer.ID])
}

func TestMessageHandling(t *testing.T) {
	config := &Config{
		ListenAddr:    "localhost:8080",
		Seeds:         []string{"seed1:8080", "seed2:8080"},
		MaxPeers:      10,
		MinPeers:      5,
		MessageBuffer: 100,
		SyncInterval:  time.Second,
	}

	service := NewService(config)
	assert.NotNil(t, service)

	// Create a test message
	msg := &types.NetworkMessage{
		Type:      types.MessageTypeBlock,
		Payload:   []byte("test payload"),
		From:      "peer1",
		To:        "peer2",
		Timestamp: time.Now(),
		TTL:       10,
		Signature: []byte("test signature"),
		Priority:  1,
	}

	// Send message
	err := service.SendMessage("peer1", msg)
	assert.NoError(t, err)

	// Verify message was stored
	assert.Equal(t, 1, len(service.messages))
	assert.Equal(t, msg, service.messages[string(msg.Signature)])
}

func TestService_SendMessage(t *testing.T) {
	config := &Config{
		ListenAddr:    "localhost:8080",
		Seeds:         []string{"seed1:8080", "seed2:8080"},
		MaxPeers:      10,
		MinPeers:      5,
		MessageBuffer: 100,
		SyncInterval:  time.Second,
	}

	service := NewService(config)
	err := service.Start()
	assert.NoError(t, err)
	defer service.Stop()

	// Add a peer
	peer := &Peer{
		ID:        "peer1",
		Addr:      "127.0.0.1:8081",
		Connected: true,
		LastSeen:  time.Now(),
	}
	service.AddPeer(peer)

	// Create a test message
	msg := &types.NetworkMessage{
		Type:      types.MessageTypeBlock,
		Payload:   []byte("test payload"),
		Signature: []byte("test signature"),
		Timestamp: time.Now(),
		From:      "peer1",
		To:        "peer2",
		Priority:  1,
		TTL:       100,
	}

	// Test sending to existing peer
	err = service.SendMessage(peer.ID, msg)
	assert.NoError(t, err)

	// Test sending to non-existent peer
	err = service.SendMessage("nonexistent", msg)
	assert.Equal(t, ErrPeerNotFound, err)
}

func TestService_BroadcastMessage(t *testing.T) {
	config := &Config{
		ListenAddr:    "localhost:8080",
		Seeds:         []string{"seed1:8080", "seed2:8080"},
		MaxPeers:      10,
		MinPeers:      5,
		MessageBuffer: 100,
		SyncInterval:  time.Second,
	}

	service := NewService(config)
	err := service.Start()
	assert.NoError(t, err)
	defer service.Stop()

	// Add multiple peers
	peers := []*Peer{
		{
			ID:        "peer1",
			Addr:      "127.0.0.1:8081",
			Connected: true,
			LastSeen:  time.Now(),
		},
		{
			ID:        "peer2",
			Addr:      "127.0.0.1:8082",
			Connected: true,
			LastSeen:  time.Now(),
		},
	}

	for _, peer := range peers {
		service.AddPeer(peer)
	}

	// Create a test message
	msg := &types.NetworkMessage{
		Type:      types.MessageTypeBlock,
		Payload:   []byte("test payload"),
		Signature: []byte("test signature"),
		Timestamp: time.Now(),
		From:      "peer1",
		To:        "peer2",
		Priority:  1,
		TTL:       100,
	}

	// Test broadcasting
	err = service.BroadcastMessage(msg)
	assert.NoError(t, err)
}

func TestService_Concurrency(t *testing.T) {
	config := &Config{
		ListenAddr:    "localhost:8080",
		Seeds:         []string{"seed1:8080", "seed2:8080"},
		MaxPeers:      10,
		MinPeers:      5,
		MessageBuffer: 100,
		SyncInterval:  time.Second,
	}

	service := NewService(config)
	err := service.Start()
	assert.NoError(t, err)
	defer service.Stop()

	// Use a WaitGroup to wait for goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Channel to collect results
	results := make(chan bool, 2)

	// Concurrently add peer and send message
	go func() {
		defer wg.Done()
		peer := &Peer{
			ID:        "peer1",
			Addr:      "127.0.0.1:8081",
			Connected: true,
			LastSeen:  time.Now(),
		}
		service.AddPeer(peer)
		results <- true
	}()

	go func() {
		defer wg.Done()
		msg := &types.NetworkMessage{
			Type:      types.MessageTypeBlock,
			Payload:   []byte("test payload"),
			Signature: []byte("test signature"),
			Timestamp: time.Now(),
			From:      "peer1",
			To:        "peer2",
			Priority:  1,
			TTL:       100,
		}
		err := service.BroadcastMessage(msg)
		assert.NoError(t, err)
		results <- true
	}()

	// Wait for goroutines to complete
	wg.Wait()
	close(results)

	// Check results
	count := 0
	for result := range results {
		assert.True(t, result)
		count++
	}
	assert.Equal(t, 2, count)
}