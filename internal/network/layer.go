package network

import (
    "errors"
    "sync"
    "time"

    "github.com/CrossDAG/BlazeDAG/internal/types"
)

// NetworkLayer represents the network communication layer
type NetworkLayer struct {
    peers    map[string]*Peer
    messages chan *types.NetworkMessage
    mu       sync.RWMutex
}

// NewNetworkLayer creates a new network layer
func NewNetworkLayer() *NetworkLayer {
    return &NetworkLayer{
        peers:    make(map[string]*Peer),
        messages: make(chan *types.NetworkMessage, 1000),
    }
}

// Start starts the network layer
func (l *NetworkLayer) Start() error {
    return nil
}

// Stop stops the network layer
func (l *NetworkLayer) Stop() error {
    close(l.messages)
    return nil
}

// AddPeer adds a peer to the network layer
func (l *NetworkLayer) AddPeer(peer *Peer) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.peers[peer.ID] = peer
}

// RemovePeer removes a peer from the network layer
func (l *NetworkLayer) RemovePeer(peerID string) {
    l.mu.Lock()
    defer l.mu.Unlock()
    delete(l.peers, peerID)
}

// GetPeers returns all peers
func (l *NetworkLayer) GetPeers() []*Peer {
    l.mu.RLock()
    defer l.mu.RUnlock()

    peers := make([]*Peer, 0, len(l.peers))
    for _, peer := range l.peers {
        peers = append(peers, peer)
    }
    return peers
}

// SendMessage sends a message to a peer
func (l *NetworkLayer) SendMessage(peer *Peer, msg *types.NetworkMessage) error {
    if !peer.Connected {
        return errors.New("peer is not connected")
    }

    // Update peer's last seen time
    peer.LastSeen = time.Now()

    // Send message through the network layer
    // This is a placeholder for actual network communication
    return nil
}

// Broadcast sends a message to all connected peers
func (l *NetworkLayer) Broadcast(msg *types.NetworkMessage) error {
    l.mu.RLock()
    defer l.mu.RUnlock()

    for _, peer := range l.peers {
        if peer.Connected {
            if err := l.SendMessage(peer, msg); err != nil {
                // Log error but continue broadcasting
                continue
            }
        }
    }
    return nil
} 