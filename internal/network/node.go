package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
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
	Port    int
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

	// Start listening for incoming connections
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", n.Port))
	if err != nil {
		return fmt.Errorf("failed to start listener: %v", err)
	}
	n.server = listener

	// Start accepting connections
	go n.acceptConnections()

	// Start peer discovery
	go n.discoverPeers()

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

	if n.server != nil {
		n.server.Close()
	}

	n.cancel()
	n.running = false
	return nil
}

// Connect adds a peer to the node's peer list
func (n *P2PNode) Connect(peerID PeerID, addr string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Connect to the peer
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %v", err)
	}

	n.peers[peerID] = conn
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
	n.mu.RLock()
	defer n.mu.RUnlock()

	for peerID, conn := range n.peers {
		if _, err := conn.Write(msg); err != nil {
			fmt.Printf("Failed to send message to peer %s: %v\n", peerID, err)
			// Don't return error, continue with other peers
		}
	}
	return nil
}

// GetID returns the node's ID
func (n *P2PNode) GetID() PeerID {
	return n.id
}




// discoverPeers periodically discovers new peers
func (n *P2PNode) discoverPeers() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			// Try to connect to known peers
			knownPeers := []struct {
				ID   PeerID
				Addr string
			}{
				{PeerID("node-3000"), "localhost:3000"},
				{PeerID("node-3001"), "localhost:3001"},
			}

			for _, peer := range knownPeers {
				if peer.ID != n.id {
					if err := n.Connect(peer.ID, peer.Addr); err != nil {
						fmt.Printf("Failed to connect to peer %s: %v\n", peer.ID, err)
					}
				}
			}
		}
	}
} 