package types

import (
	"crypto/sha256"
	"encoding/json"
)

// Transaction represents a transaction in the BlazeDAG system
type Transaction struct {
	From     []byte `json:"from"`
	To       []byte `json:"to"`
	Value    uint64 `json:"value"`
	Nonce    uint64 `json:"nonce"`
	GasLimit uint64 `json:"gasLimit"`
	GasPrice uint64 `json:"gasPrice"`
	Data     []byte `json:"data"`
}

// Hash returns the hash of the transaction
func (t *Transaction) Hash() []byte {
	data, _ := json.Marshal(t)
	hash := sha256.Sum256(data)
	return hash[:]
}

// TransactionResult represents the result of executing a transaction
type TransactionResult struct {
	Success          bool   `json:"success"`
	TransactionHash  []byte `json:"transactionHash"`
	GasUsed          uint64 `json:"gasUsed"`
	SenderBalance    uint64 `json:"senderBalance"`
	RecipientBalance uint64 `json:"recipientBalance"`
	SenderNonce      uint64 `json:"senderNonce"`
}

// TransactionReceipt represents a transaction receipt
type TransactionReceipt struct {
	TransactionHash []byte `json:"transactionHash"`
	Success         bool   `json:"success"`
	GasUsed         uint64 `json:"gasUsed"`
	BlockHash       []byte `json:"blockHash"`
	BlockNumber     uint64 `json:"blockNumber"`
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
	Transaction1 []byte       `json:"transaction1"`
	Transaction2 []byte       `json:"transaction2"`
	Type         ConflictType `json:"type"`
}

// Receipt represents a transaction receipt
type Receipt struct {
	TransactionHash []byte
	BlockHash       []byte
	BlockNumber     uint64
	From            []byte
	To              []byte
	ContractAddress []byte
	GasUsed         uint64
	Status          uint8
	Logs            []Log
}

// Log represents a log entry
type Log struct {
	Address         []byte
	Topics          [][]byte
	Data            []byte
	BlockNumber     uint64
	TransactionHash []byte
	LogIndex        uint64
	BlockHash       []byte
	TransactionIndex uint64
}

// Event represents an event
type Event struct {
	Address         []byte
	Topics          [][]byte
	Data            []byte
	BlockNumber     uint64
	TransactionHash []byte
	LogIndex        uint64
	BlockHash       []byte
	TransactionIndex uint64
} 