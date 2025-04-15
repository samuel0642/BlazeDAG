package network

import (
    "time"
)

// Peer represents a network peer
type Peer struct {
    ID        string
    Addr      string
    Connected bool
    LastSeen  time.Time
}

// NewPeer creates a new peer
func NewPeer(id, addr string) *Peer {
    return &Peer{
        ID:        id,
        Addr:      addr,
        Connected: false,
        LastSeen:  time.Now(),
    }
}

// IsActive checks if the peer is active
func (p *Peer) IsActive() bool {
    return p.Connected && time.Since(p.LastSeen) < 5*time.Minute
}

// UpdateLastSeen updates the peer's last seen timestamp
func (p *Peer) UpdateLastSeen() {
    p.LastSeen = time.Now()
}

// Connect marks the peer as connected
func (p *Peer) Connect() {
    p.Connected = true
    p.UpdateLastSeen()
}

// Disconnect marks the peer as disconnected
func (p *Peer) Disconnect() {
    p.Connected = false
} 