package types

import (
	"time"
)

// Block represents a block in the DAG
type Block struct {
	Header       BlockHeader
	Body         BlockBody
	Certificate  *Certificate
	References   []Reference
	Signature    []byte
	Timestamp    time.Time
}

// BlockHeader contains the block's metadata
type BlockHeader struct {
	Version        uint32
	Round          uint64
	Wave           uint64
	Height         uint64
	ParentHash     []byte
	StateRoot      []byte
	TransactionRoot []byte
	ReceiptRoot    []byte
	Validator      []byte
}

// BlockBody contains the block's transactions and receipts
type BlockBody struct {
	Transactions []Transaction
	Receipts     []Receipt
	Events       []Event
}

// Reference represents a reference to another block
type Reference struct {
	BlockHash []byte
	Round     uint64
	Wave      uint64
	Type      ReferenceType
}

// Certificate contains the block's certificate
type Certificate struct {
	BlockHash    []byte
	Signatures   [][]byte
	Round        uint64
	Wave         uint64
	ValidatorSet [][]byte
	Timestamp    time.Time
}

// ReferenceType represents the type of reference
type ReferenceType uint8

const (
	ReferenceTypeStandard ReferenceType = iota
	ReferenceTypeVote
	ReferenceTypeComplaint
) 