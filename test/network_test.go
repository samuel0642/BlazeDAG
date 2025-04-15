package test

import (
	"context"
	"testing"
	"time"

	"github.com/samuel0642/BlazeDAG/internal/network"
	"github.com/samuel0642/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNetworkLayer(t *testing.T) {
	// Create network layer
	nl := network.NewNetworkLayer()
	assert.NotNil(t, nl)

	// Start network layer
	err := nl.Start()
	assert.NoError(t, err)
	defer nl.Stop()

	// Add test peers
	peer1 := &network.Peer{
		ID:     "peer1",
		Addr:   "localhost:8001",
		Active: true,
	}
	peer2 := &network.Peer{
		ID:     "peer2",
		Addr:   "localhost:8002",
		Active: true,
	}
	nl.AddPeer(peer1)
	nl.AddPeer(peer2)

	// Test broadcasting a block message
	blockMsg := &types.NetworkMessage{
		Type:      types.MessageTypeBlock,
		Payload:   []byte("test block payload"),
		From:      "peer1",
		To:        "peer2",
		Timestamp: time.Now(),
		TTL:       10,
		Signature: []byte("test-signature"),
		Priority:  1,
	}
	err = nl.Broadcast(blockMsg)
	assert.NoError(t, err)

	// Test broadcasting a transaction message
	txMsg := &types.NetworkMessage{
		Type:      types.MessageTypeTransaction,
		Payload:   []byte("test tx payload"),
		From:      "peer1",
		To:        "peer2",
		Timestamp: time.Now(),
		TTL:       10,
		Signature: []byte("test-signature-2"),
		Priority:  1,
	}
	err = nl.Broadcast(txMsg)
	assert.NoError(t, err)

	// Test removing a peer
	nl.RemovePeer("peer1")

	// Wait for messages to be processed
	time.Sleep(100 * time.Millisecond)
}

func TestNetworkService(t *testing.T) {
	// Create network service
	config := &network.Config{
		ListenAddr:    "localhost:8080",
		Seeds:         []string{"seed1:8080", "seed2:8080"},
		MaxPeers:      10,
		MinPeers:      5,
		MessageBuffer: 100,
		SyncInterval:  time.Second,
	}

	service := network.NewService(config)
	assert.NotNil(t, service)

	// Start service
	err := service.Start()
	assert.NoError(t, err)
	defer service.Stop()

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

	// Broadcast message
	err = service.BroadcastMessage(msg)
	assert.NoError(t, err)

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify message was stored
	assert.Equal(t, msg, service.Messages[string(msg.Signature)])
}

func TestNetworkServiceSync(t *testing.T) {
	config := &network.Config{
		ListenAddr:    ":8001",
		Seeds:         []string{"seed1:8001"},
		MaxPeers:      10,
		MinPeers:      3,
		MessageBuffer: 100,
		SyncInterval:  time.Second,
	}

	service := network.NewService(config)
	assert.NotNil(t, service)

	err := service.Start()
	assert.NoError(t, err)

	// Test sync request message
	msg := &types.NetworkMessage{
		Type:      types.MessageTypeSyncRequest,
		Payload:   []byte("sync-payload"),
		Signature: []byte("sync-signature"),
		Timestamp: time.Now(),
		From:      "test-sender",
		Priority:  1,
		TTL:       10,
	}

	err = service.BroadcastMessage(msg)
	assert.NoError(t, err)

	// Stop service
	err = service.Stop()
	assert.NoError(t, err)
}

func TestNetworkServicePeerHandling(t *testing.T) {
	config := &network.Config{
		ListenAddr:    ":8002",
		Seeds:         []string{"seed1:8002"},
		MaxPeers:      10,
		MinPeers:      3,
		MessageBuffer: 100,
		SyncInterval:  time.Second,
	}

	service := network.NewService(config)
	assert.NotNil(t, service)

	err := service.Start()
	assert.NoError(t, err)

	// Test peer handling
	peers := service.GetPeers()
	assert.NotNil(t, peers)

	// Stop service
	err = service.Stop()
	assert.NoError(t, err)
}