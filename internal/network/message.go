package network

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// MessageType represents the type of message
type MessageType int

const (
	MessageTypeBlock MessageType = iota
	MessageTypeVote
	MessageTypeCertificate
	MessageTypeComplaint
	MessageTypeSyncRequest
	MessageTypeSyncResponse
	MessageTypeRecoveryRequest
	MessageTypeRecoveryResponse
)

// Message represents a network message
type Message struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
	Sender    peer.ID         `json:"sender"`
}

// MessageHandler handles network messages
type MessageHandler struct {
	node      *P2PNode
	handlers  map[MessageType]MessageHandlerFunc
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// MessageHandlerFunc is a function that handles a message
type MessageHandlerFunc func(*Message) error

// NewMessageHandler creates a new message handler
func NewMessageHandler(node *P2PNode) *MessageHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &MessageHandler{
		node:     node,
		handlers: make(map[MessageType]MessageHandlerFunc),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// RegisterHandler registers a message handler
func (mh *MessageHandler) RegisterHandler(msgType MessageType, handler MessageHandlerFunc) {
	mh.mu.Lock()
	defer mh.mu.Unlock()
	mh.handlers[msgType] = handler
}

// HandleMessage handles a message
func (mh *MessageHandler) HandleMessage(msg *Message) error {
	mh.mu.RLock()
	handler, exists := mh.handlers[msg.Type]
	mh.mu.RUnlock()

	if !exists {
		return nil
	}

	return handler(msg)
}

// SendMessage sends a message to a peer
func (mh *MessageHandler) SendMessage(peerID peer.ID, msgType MessageType, payload interface{}) error {
	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Create message
	msg := &Message{
		Type:      msgType,
		Payload:   payloadBytes,
		Timestamp: time.Now(),
		Sender:    mh.node.host.ID(),
	}

	// Marshal message
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Send message
	stream, err := mh.node.host.NewStream(mh.ctx, peerID, "/blazedag/1.0.0")
	if err != nil {
		return err
	}
	defer stream.Close()

	_, err = stream.Write(msgBytes)
	return err
}

// BroadcastMessage broadcasts a message to all peers
func (mh *MessageHandler) BroadcastMessage(msgType MessageType, payload interface{}) error {
	peers := mh.node.GetActivePeers()
	for _, peer := range peers {
		if err := mh.SendMessage(peer.ID, msgType, payload); err != nil {
			// Log error but continue with other peers
			continue
		}
	}
	return nil
}

// Start starts the message handler
func (mh *MessageHandler) Start() {
	// Set up stream handler
	mh.node.host.SetStreamHandler("/blazedag/1.0.0", func(stream Stream) {
		defer stream.Close()

		// Read message
		var msg Message
		if err := json.NewDecoder(stream).Decode(&msg); err != nil {
			return
		}

		// Handle message
		if err := mh.HandleMessage(&msg); err != nil {
			return
		}
	})
}

// Stop stops the message handler
func (mh *MessageHandler) Stop() {
	mh.cancel()
} 