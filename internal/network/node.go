package network

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// PeerID represents a unique identifier for a peer
type PeerID string

// P2PNode represents a P2P network node
type P2PNode struct {
	id      PeerID
	peers   map[PeerID]net.Conn
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	mu      sync.RWMutex
	port    int
	server  net.Listener
}

// NewP2PNode creates a new P2P node
func NewP2PNode(id PeerID) *P2PNode {
	ctx, cancel := context.WithCancel(context.Background())
	return &P2PNode{
		id:      id,
		peers:   make(map[PeerID]net.Conn),
		ctx:     ctx,
		cancel:  cancel,
		running: false,
	}
}

// Start starts the P2P node
func (n *P2PNode) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return nil
	}

	n.running = true
	return nil
}

// Stop stops the P2P node
func (n *P2PNode) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.running {
		return nil
	}

	// Close all peer connections
	for _, conn := range n.peers {
		conn.Close()
	}

	n.cancel()
	n.running = false
	return nil
}

// Connect adds a peer to the node's peer list
func (n *P2PNode) Connect(peerID PeerID) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// For now, just store the peer ID
	n.peers[peerID] = nil
	return nil
}

// Disconnect removes a peer from the node's peer list
func (n *P2PNode) Disconnect(peerID PeerID) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if conn, exists := n.peers[peerID]; exists {
		if conn != nil {
			conn.Close()
		}
		delete(n.peers, peerID)
	}
	return nil
}

// GetActivePeers returns a list of active peers
func (n *P2PNode) GetActivePeers() []PeerID {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]PeerID, 0, len(n.peers))
	for peer := range n.peers {
		peers = append(peers, peer)
	}
	return peers
}

// Broadcast sends a message to all connected peers
func (n *P2PNode) Broadcast(msg []byte) error {
	// For now, just log the broadcast
	fmt.Printf("Broadcasting message to %d peers\n", len(n.peers))
	return nil
}

// GetID returns the node's ID
func (n *P2PNode) GetID() PeerID {
	return n.id
} 