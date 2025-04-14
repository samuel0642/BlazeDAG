package types

import "time"

// NetworkMessage represents a network message
type NetworkMessage struct {
	Type      MessageType
	Payload   []byte
	Signature []byte
	Timestamp time.Time
	Sender    []byte
	Recipient []byte
	Priority  uint8
	TTL       uint64
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
type MessageType uint8

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
type SyncRequestType uint8

const (
	SyncRequestTypeFull SyncRequestType = iota
	SyncRequestTypeIncremental
	SyncRequestTypeFast
)

// SyncResponseType represents the type of sync response
type SyncResponseType uint8

const (
	SyncResponseTypeFull SyncResponseType = iota
	SyncResponseTypeIncremental
	SyncResponseTypeFast
)

// RecoveryRequestType represents the type of recovery request
type RecoveryRequestType uint8

const (
	RecoveryRequestTypeBlock RecoveryRequestType = iota
	RecoveryRequestTypeState
	RecoveryRequestTypeCertificate
)

// RecoveryResponseType represents the type of recovery response
type RecoveryResponseType uint8

const (
	RecoveryResponseTypeBlock RecoveryResponseType = iota
	RecoveryResponseTypeState
	RecoveryResponseTypeCertificate
) 