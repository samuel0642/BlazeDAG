package types

import (
	"time"
)

// Address represents a blockchain address
type Address string

// Hash represents a 32-byte hash
type Hash []byte

// BlockNumber represents a block number
type BlockNumber uint64

// Round represents a consensus round number
type Round uint64

// Block represents a block in the blockchain
type Block struct {
	Header      *BlockHeader
	Body        *BlockBody
	Certificate *Certificate
	Timestamp   time.Time
	hash        Hash // cached hash
}

// BlockHeader represents the header of a block
type BlockHeader struct {
	Version     uint32
	Wave        Wave
	Round       Round
	Height      Height
	ParentHash  []byte
	StateRoot   []byte
	Validator   []byte
	Timestamp   time.Time
}

// BlockBody represents the body of a block
type BlockBody struct {
	Transactions []Transaction
	Receipts     []Receipt
	Events       []Event
	References   []Reference
}

// Transaction represents a blockchain transaction
type Transaction struct {
	From   Address
	To     Address
	Value  Value
	Nonce  Nonce
	Data   []byte
}

// Receipt represents a transaction receipt
type Receipt struct {
	TxHash          []byte
	Status          uint64
	GasUsed         uint64
	ContractAddress []byte
	Logs           []Log
}

// Event represents a blockchain event
type Event struct {
	Address []byte
	Topics  [][]byte
	Data    []byte
}

// Reference represents a block reference
type Reference struct {
	BlockHash []byte
	Type      ReferenceType
}

// ReferenceType represents the type of block reference
type ReferenceType int

const (
	ReferenceTypeParent ReferenceType = iota
	ReferenceTypeUncle
	ReferenceTypeWave
)

// Certificate represents a block certificate
type Certificate struct {
	BlockHash   []byte
	Wave       Wave
	Round      Round
	Timestamp  time.Time
	Signatures []Signature
	Validators []Address
}

// Log represents a transaction log
type Log struct {
	Address []byte
	Topics  [][]byte
	Data    []byte
}

// Value represents a numeric value
type Value uint64

// Nonce represents a transaction nonce
type Nonce uint64

// Signature represents a digital signature
type Signature []byte

// Gas represents the amount of gas used in a transaction
type Gas uint64

// Timestamp represents a block timestamp
type Timestamp int64

// TransactionIndex represents the index of a transaction in a block
type TransactionIndex uint64

// LogIndex represents the index of a log in a transaction
type LogIndex uint64

// Status represents the status of a transaction
type Status uint8

// Version represents the version of a block
type Version uint32

// Height represents the height of a block
type Height uint64

// TransactionResult represents the result of executing a transaction
type TransactionResult struct {
	Success          bool    `json:"success"`
	TransactionHash  Hash    `json:"transactionHash"`
	GasUsed          Gas     `json:"gasUsed"`
	SenderBalance    Value   `json:"senderBalance"`
	RecipientBalance Value   `json:"recipientBalance"`
	SenderNonce      Nonce   `json:"senderNonce"`
}

// TransactionReceipt represents a transaction receipt
type TransactionReceipt struct {
	TransactionHash Hash        `json:"transactionHash"`
	Success         bool        `json:"success"`
	GasUsed         Gas         `json:"gasUsed"`
	BlockHash       Hash        `json:"blockHash"`
	BlockNumber     BlockNumber `json:"blockNumber"`
}

// ConflictType represents the type of transaction conflict
type ConflictType int

const (
	ConflictTypeReadWrite ConflictType = iota
	ConflictTypeWriteWrite
	ConflictTypeNonce
	ConflictTypeBalance
)

// TransactionConflict represents a conflict between transactions
type TransactionConflict struct {
	Transaction1 Hash         `json:"transaction1"`
	Transaction2 Hash         `json:"transaction2"`
	Type         ConflictType `json:"type"`
}

// Proposal represents a block proposal
type Proposal struct {
	ID        []byte
	BlockHash []byte
	Wave      Wave
	Round     Round
	Block     *Block
	Proposer  Address
	Timestamp time.Time
	Status    WaveStatus
}

// Vote represents a vote on a block proposal
type Vote struct {
	ProposalID []byte
	BlockHash  []byte
	Wave       Wave
	Round      Round
	Validator  Address
	Timestamp  time.Time
} 