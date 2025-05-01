package core

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// NetworkManager handles network communication
type NetworkManager struct {
	config       *Config
	stateManager *StateManager

	// Network state
	peers       map[types.Address]*Peer
	peerLock    sync.RWMutex
	messageChan chan *types.NetworkMessage
	stopChan    chan struct{}
}

// NewNetworkManager creates a new network manager
func NewNetworkManager(config *Config, stateManager *StateManager) *NetworkManager {
	return &NetworkManager{
		config:       config,
		stateManager: stateManager,
		peers:        make(map[types.Address]*Peer),
		messageChan:  make(chan *types.NetworkMessage, 100),
		stopChan:     make(chan struct{}),
	}
}

// Start starts the network manager
func (nm *NetworkManager) Start() error {
	// Start listening for connections
	listener, err := net.Listen("tcp", ":3000")
	if err != nil {
		return err
	}

	// Start accepting connections
	go nm.acceptConnections(listener)

	// Start message handler
	go nm.handleMessages()

	return nil
}

// Stop stops the network manager
func (nm *NetworkManager) Stop() {
	close(nm.stopChan)
}

// acceptConnections accepts incoming connections
func (nm *NetworkManager) acceptConnections(listener net.Listener) {
	for {
		select {
		case <-nm.stopChan:
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			go nm.handleConnection(conn)
		}
	}
}

// handleConnection handles a new connection
func (nm *NetworkManager) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read peer information
	peer := &Peer{
		Address:    types.Address(conn.RemoteAddr().String()),
		LastSeen:   time.Now(),
		Connection: conn,
	}

	// Add peer
	nm.addPeer(peer)

	// Handle messages from peer
	for {
		select {
		case <-nm.stopChan:
			return
		default:
			// Read message
			msg, err := nm.readMessage(conn)
			if err != nil {
				log.Printf("Failed to read message: %v", err)
				nm.removePeer(peer.Address)
				return
			}

			// Process message
			nm.messageChan <- msg
		}
	}
}

// handleMessages handles incoming messages
func (nm *NetworkManager) handleMessages() {
	for {
		select {
		case msg := <-nm.messageChan:
			nm.processMessage(msg)
		case <-nm.stopChan:
			return
		}
	}
}

// processMessage processes a network message
func (nm *NetworkManager) processMessage(msg *types.NetworkMessage) {
	// TODO: Implement message processing
}

// addPeer adds a new peer
func (nm *NetworkManager) addPeer(peer *Peer) {
	nm.peerLock.Lock()
	defer nm.peerLock.Unlock()
	nm.peers[peer.Address] = peer
}

// removePeer removes a peer
func (nm *NetworkManager) removePeer(address types.Address) {
	nm.peerLock.Lock()
	defer nm.peerLock.Unlock()
	delete(nm.peers, address)
}

// updatePeerLastSeen updates a peer's last seen timestamp
func (nm *NetworkManager) updatePeerLastSeen(address types.Address) {
	nm.peerLock.Lock()
	defer nm.peerLock.Unlock()
	if peer, exists := nm.peers[address]; exists {
		peer.LastSeen = time.Now()
	}
}

// readMessage reads a message from a connection
func (nm *NetworkManager) readMessage(conn net.Conn) (*types.NetworkMessage, error) {
	// TODO: Implement message reading
	// For now, return a dummy message
	return &types.NetworkMessage{
		Type:    types.MessageTypeBlock,
		Sender:  types.Address(conn.RemoteAddr().String()),
		Payload: []byte("dummy_message"),
	}, nil
}

// Peer represents a connected peer
type Peer struct {
	Address    types.Address
	LastSeen   time.Time
	Connection net.Conn
} 