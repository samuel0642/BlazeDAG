package network

import (
	"encoding/json"
	"fmt"
	"sync"
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
	Type      MessageType `json:"type"`
	Payload   []byte     `json:"payload"`
	Timestamp int64      `json:"timestamp"`
	Sender    PeerID     `json:"sender"`
}

// MessageHandler handles network messages
type MessageHandler struct {
	node     *P2PNode
	handlers map[MessageType]func(*Message) error
	running  bool
	mu       sync.RWMutex
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(node *P2PNode) *MessageHandler {
	return &MessageHandler{
		node:     node,
		handlers: make(map[MessageType]func(*Message) error),
	}
}

// RegisterHandler registers a handler for a message type
func (mh *MessageHandler) RegisterHandler(msgType MessageType, handler func(*Message) error) {
	mh.mu.Lock()
	defer mh.mu.Unlock()
	mh.handlers[msgType] = handler
}

// Start starts the message handler
func (mh *MessageHandler) Start() error {
	mh.mu.Lock()
	defer mh.mu.Unlock()

	if mh.running {
		return nil
	}

	mh.running = true
	return nil
}

// Stop stops the message handler
func (mh *MessageHandler) Stop() error {
	mh.mu.Lock()
	defer mh.mu.Unlock()

	if !mh.running {
		return nil
	}

	mh.running = false
	return nil
}

// HandleMessage handles an incoming message
func (mh *MessageHandler) HandleMessage(msg *Message) error {
	mh.mu.RLock()
	handler, exists := mh.handlers[msg.Type]
	mh.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for message type %d", msg.Type)
	}

	return handler(msg)
}

// UnmarshalPayload unmarshals the message payload into the provided interface
func (m *Message) UnmarshalPayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}

// SendMessage sends a message to a specific peer
func (mh *MessageHandler) SendMessage(peerID PeerID, msgType MessageType, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	msg := &Message{
		Type:      msgType,
		Payload:   payloadBytes,
		Timestamp: 0, // TODO: Add proper timestamp
		Sender:    mh.node.GetID(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	return mh.node.Broadcast(msgBytes)
}

// BroadcastMessage broadcasts a message to all peers
func (mh *MessageHandler) BroadcastMessage(msgType MessageType, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	msg := &Message{
		Type:      msgType,
		Payload:   payloadBytes,
		Timestamp: 0, // TODO: Add proper timestamp
		Sender:    mh.node.GetID(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	return mh.node.Broadcast(msgBytes)
} 