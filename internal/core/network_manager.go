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
	config *Config
	state  *State

	// Network state
	listener net.Listener
	peers    map[types.Address]*Peer
	peerLock sync.RWMutex

	// Message channels
	messageChan chan *types.NetworkMessage
	stopChan    chan struct{}
}

// NewNetworkManager creates a new network manager
func NewNetworkManager(config *Config, state *State) *NetworkManager {
	return &NetworkManager{
		config:      config,
		state:       state,
		peers:       make(map[types.Address]*Peer),
		messageChan: make(chan *types.NetworkMessage, 100),
		stopChan:    make(chan struct{}),
	}
}

// Start starts the network manager
func (nm *NetworkManager) Start() error {
	// Start listening for connections
	listener, err := net.Listen("tcp", ":3000")
	if err != nil {
		return err
	}
	nm.listener = listener

	// Start accepting connections
	go nm.acceptConnections()

	// Start message processing
	go nm.processMessages()

	return nil
}

// Stop stops the network manager
func (nm *NetworkManager) Stop() {
	close(nm.stopChan)
	if nm.listener != nil {
		nm.listener.Close()
	}
}

// acceptConnections accepts incoming connections
func (nm *NetworkManager) acceptConnections() {
	for {
		select {
		case <-nm.stopChan:
			return
		default:
			conn, err := nm.listener.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v", err)
				continue
			}

			// Handle new connection
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

// processMessages processes incoming messages
func (nm *NetworkManager) processMessages() {
	for {
		select {
		case <-nm.stopChan:
			return
		case msg := <-nm.messageChan:
			nm.handleMessage(msg)
		}
	}
}

// handleMessage handles an incoming message
func (nm *NetworkManager) handleMessage(msg *types.NetworkMessage) {
	// Update peer last seen
	nm.updatePeerLastSeen(msg.Sender)

	// Handle message based on type
	switch msg.Type {
	case types.MessageTypeBlock:
		nm.handleBlockMessage(msg)
	case types.MessageTypeVote:
		nm.handleVoteMessage(msg)
	case types.MessageTypeProposal:
		nm.handleProposalMessage(msg)
	case types.MessageTypeComplaint:
		nm.handleComplaintMessage(msg)
	}
}

// handleBlockMessage handles a block message
func (nm *NetworkManager) handleBlockMessage(msg *types.NetworkMessage) {
	// TODO: Implement block message handling
	log.Printf("Received block message from %s", msg.Sender)
}

// handleVoteMessage handles a vote message
func (nm *NetworkManager) handleVoteMessage(msg *types.NetworkMessage) {
	// TODO: Implement vote message handling
	log.Printf("Received vote message from %s", msg.Sender)
}

// handleProposalMessage handles a proposal message
func (nm *NetworkManager) handleProposalMessage(msg *types.NetworkMessage) {
	// TODO: Implement proposal message handling
	log.Printf("Received proposal message from %s", msg.Sender)
}

// handleComplaintMessage handles a complaint message
func (nm *NetworkManager) handleComplaintMessage(msg *types.NetworkMessage) {
	// TODO: Implement complaint message handling
	log.Printf("Received complaint message from %s", msg.Sender)
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