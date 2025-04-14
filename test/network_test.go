package test

import (
	"testing"
	"time"

	"BlazeDAG/internal/network"
	"BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNetworkLayer(t *testing.T) {
	// Create network layer
	net := network.NewNetworkLayer()

	// Test peer discovery
	peers, err := net.DiscoverPeers()
	assert.NoError(t, err)
	assert.NotEmpty(t, peers)

	// Test message handling
	msg := &types.NetworkMessage{
		Type: types.MessageTypeBlock,
		Payload: &types.BlockMessage{
			Block: &types.Block{
				Header: &types.BlockHeader{
					ParentHash: []byte("test"),
				},
			},
		},
	}

	// Test message broadcasting
	err = net.BroadcastMessage(msg)
	assert.NoError(t, err)

	// Test message receiving
	received := make(chan *types.NetworkMessage)
	go func() {
		msg, err := net.ReceiveMessage()
		assert.NoError(t, err)
		received <- msg
	}()

	// Wait for message
	select {
	case <-received:
		// Message received successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message")
	}

	// Test peer management
	peerID := "test-peer"
	err = net.AddPeer(peerID)
	assert.NoError(t, err)

	peers = net.GetPeers()
	assert.Contains(t, peers, peerID)

	err = net.RemovePeer(peerID)
	assert.NoError(t, err)

	peers = net.GetPeers()
	assert.NotContains(t, peers, peerID)
}

func TestNetworkSynchronization(t *testing.T) {
	net := network.NewNetworkLayer()

	// Test state synchronization
	stateReq := &types.SyncRequest{
		Type: types.SyncRequestTypeFull,
		StartBlock: []byte("start"),
		EndBlock: []byte("end"),
	}

	err := net.RequestStateSync(stateReq)
	assert.NoError(t, err)

	// Test block synchronization
	blockReq := &types.SyncRequest{
		Type: types.SyncRequestTypeIncremental,
		StartBlock: []byte("start"),
		EndBlock: []byte("end"),
	}

	err = net.RequestBlockSync(blockReq)
	assert.NoError(t, err)

	// Test certificate synchronization
	certReq := &types.SyncRequest{
		Type: types.SyncRequestTypeCertificate,
		StartBlock: []byte("start"),
		EndBlock: []byte("end"),
	}

	err = net.RequestCertificateSync(certReq)
	assert.NoError(t, err)
}

func TestNetworkSecurity(t *testing.T) {
	net := network.NewNetworkLayer()

	// Test message authentication
	msg := &types.NetworkMessage{
		Type: types.MessageTypeBlock,
		Payload: &types.BlockMessage{
			Block: &types.Block{
				Header: &types.BlockHeader{
					ParentHash: []byte("test"),
				},
			},
		},
	}

	// Sign message
	err := net.SignMessage(msg)
	assert.NoError(t, err)

	// Verify signature
	valid, err := net.VerifyMessage(msg)
	assert.NoError(t, err)
	assert.True(t, valid)

	// Test encryption
	encrypted, err := net.EncryptMessage(msg)
	assert.NoError(t, err)

	decrypted, err := net.DecryptMessage(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, msg, decrypted)
}

func TestNetworkPerformance(t *testing.T) {
	net := network.NewNetworkLayer()

	// Test message throughput
	start := time.Now()
	count := 1000

	for i := 0; i < count; i++ {
		msg := &types.NetworkMessage{
			Type: types.MessageTypeBlock,
			Payload: &types.BlockMessage{
				Block: &types.Block{
					Header: &types.BlockHeader{
						ParentHash: []byte("test"),
					},
				},
			},
		}
		err := net.BroadcastMessage(msg)
		assert.NoError(t, err)
	}

	duration := time.Since(start)
	throughput := float64(count) / duration.Seconds()
	assert.Greater(t, throughput, 100.0) // Expect at least 100 messages per second

	// Test latency
	msg := &types.NetworkMessage{
		Type: types.MessageTypeBlock,
		Payload: &types.BlockMessage{
			Block: &types.Block{
				Header: &types.BlockHeader{
					ParentHash: []byte("test"),
				},
			},
		},
	}

	start = time.Now()
	err := net.BroadcastMessage(msg)
	assert.NoError(t, err)

	received := make(chan *types.NetworkMessage)
	go func() {
		msg, err := net.ReceiveMessage()
		assert.NoError(t, err)
		received <- msg
	}()

	select {
	case <-received:
		latency := time.Since(start)
		assert.Less(t, latency, 100*time.Millisecond) // Expect latency less than 100ms
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
} 