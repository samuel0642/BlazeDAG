package network

import (
    "errors"
    "sync"
    "time"

    "github.com/CrossDAG/BlazeDAG/internal/types"
)

// NetworkLayer represents the network communication layer
type NetworkLayer struct {
    peers    map[types.Address]*types.Peer
    messages chan *types.NetworkMessage
    mu       sync.RWMutex
}

// NewNetworkLayer creates a new network layer
func NewNetworkLayer() *NetworkLayer {
    return &NetworkLayer{
        peers:    make(map[types.Address]*types.Peer),
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
func (l *NetworkLayer) AddPeer(peer *types.Peer) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.peers[peer.Address] = peer
}

// RemovePeer removes a peer from the network layer
func (l *NetworkLayer) RemovePeer(address types.Address) {
    l.mu.Lock()
    defer l.mu.Unlock()
    delete(l.peers, address)
}

// GetPeers returns all peers
func (l *NetworkLayer) GetPeers() []*types.Peer {
    l.mu.RLock()
    defer l.mu.RUnlock()

    peers := make([]*types.Peer, 0, len(l.peers))
    for _, peer := range l.peers {
        peers = append(peers, peer)
    }
    return peers
}

// SendMessage sends a message to a peer
func (l *NetworkLayer) SendMessage(peer *types.Peer, msg *types.NetworkMessage) error {
    if peer.Connection == nil {
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
        if peer.Connection != nil {
            if err := l.SendMessage(peer, msg); err != nil {
                // Log error but continue broadcasting
                continue
            }
        }
    }
    return nil
} 