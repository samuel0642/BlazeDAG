package network

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

// P2PNode represents a P2P node
type P2PNode struct {
	host    host.Host
	peers   map[peer.ID]*PeerInfo
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// PeerInfo represents information about a peer
type PeerInfo struct {
	ID        peer.ID
	Address   string
	LastSeen  time.Time
	Latency   time.Duration
	IsActive  bool
}

// NewP2PNode creates a new P2P node
func NewP2PNode(port int) (*P2PNode, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a new libp2p host
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Transport(tcp.NewTCPTransport),
	)
	if err != nil {
		cancel()
		return nil, err
	}

	node := &P2PNode{
		host:   h,
		peers:  make(map[peer.ID]*PeerInfo),
		ctx:    ctx,
		cancel: cancel,
	}

	// Start peer discovery
	go node.discoverPeers()

	return node, nil
}

// discoverPeers discovers new peers
func (n *P2PNode) discoverPeers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.mu.Lock()
			for _, peerInfo := range n.peers {
				if time.Since(peerInfo.LastSeen) > 5*time.Minute {
					peerInfo.IsActive = false
				}
			}
			n.mu.Unlock()
		}
	}
}

// Connect connects to a peer
func (n *P2PNode) Connect(addr string) error {
	peerInfo, err := peer.AddrInfoFromString(addr)
	if err != nil {
		return err
	}

	if err := n.host.Connect(n.ctx, *peerInfo); err != nil {
		return err
	}

	n.mu.Lock()
	n.peers[peerInfo.ID] = &PeerInfo{
		ID:       peerInfo.ID,
		Address:  addr,
		LastSeen: time.Now(),
		IsActive: true,
	}
	n.mu.Unlock()

	return nil
}

// Disconnect disconnects from a peer
func (n *P2PNode) Disconnect(peerID peer.ID) {
	n.mu.Lock()
	delete(n.peers, peerID)
	n.mu.Unlock()
	n.host.Network().ClosePeer(peerID)
}

// GetPeers returns all connected peers
func (n *P2PNode) GetPeers() []*PeerInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]*PeerInfo, 0, len(n.peers))
	for _, peer := range n.peers {
		peers = append(peers, peer)
	}
	return peers
}

// GetActivePeers returns all active peers
func (n *P2PNode) GetActivePeers() []*PeerInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]*PeerInfo, 0)
	for _, peer := range n.peers {
		if peer.IsActive {
			peers = append(peers, peer)
		}
	}
	return peers
}

// UpdatePeerLatency updates a peer's latency
func (n *P2PNode) UpdatePeerLatency(peerID peer.ID, latency time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if peer, exists := n.peers[peerID]; exists {
		peer.Latency = latency
		peer.LastSeen = time.Now()
	}
}

// Close closes the P2P node
func (n *P2PNode) Close() error {
	n.cancel()
	return n.host.Close()
} 