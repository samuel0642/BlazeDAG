package types

import "time"

// Peer represents a network peer
type Peer struct {
	ID        string
	Address   string
	Active    bool
	LastSeen  time.Time
	Score     int
}

// NetworkMessage represents a message sent over the network
type NetworkMessage struct {
	Type      MessageType
	Payload   []byte
	From      string
	To        string
	Sender    string
	Recipient string
	Timestamp time.Time
	TTL       int
	Signature []byte
	Priority  int
}

// SyncRequest represents a sync request
type SyncRequest struct {
	Type       SyncRequestType
	StartBlock []byte
	EndBlock   []byte
	StateRoot  []byte
	Timestamp  time.Time
	Signature  []byte
}

// SyncResponse represents a sync response
type SyncResponse struct {
	Type      SyncResponseType
	Blocks    []Block
	StateRoot []byte
	Timestamp time.Time
	Signature []byte
}

// RecoveryRequest represents a recovery request
type RecoveryRequest struct {
	Type         RecoveryRequestType
	MissingBlocks [][]byte
	StateRoot    []byte
	Timestamp    time.Time
	Signature    []byte
}

// RecoveryResponse represents a recovery response
type RecoveryResponse struct {
	Type      RecoveryResponseType
	Blocks    []Block
	StateRoot []byte
	Timestamp time.Time
	Signature []byte
}

// MessageType represents the type of network message
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

// SyncRequestType represents the type of sync request
type SyncRequestType int

const (
	SyncRequestTypeFull SyncRequestType = iota
	SyncRequestTypeIncremental
	SyncRequestTypeFast
	SyncRequestTypeCertificate
)

// SyncResponseType represents the type of sync response
type SyncResponseType int

const (
	SyncResponseTypeFull SyncResponseType = iota
	SyncResponseTypeIncremental
	SyncResponseTypeFast
)

// RecoveryRequestType represents the type of recovery request
type RecoveryRequestType int

const (
	RecoveryRequestTypeBlock RecoveryRequestType = iota
	RecoveryRequestTypeState
	RecoveryRequestTypeCertificate
)

// RecoveryResponseType represents the type of recovery response
type RecoveryResponseType int

const (
	RecoveryResponseTypeBlock RecoveryResponseType = iota
	RecoveryResponseTypeState
	RecoveryResponseTypeCertificate
) 