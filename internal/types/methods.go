package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// ComputeHash returns the hash of the block
func (b *Block) ComputeHash() string {
	if b.hash == nil {
		data, _ := json.Marshal(b)
		hash := sha256.Sum256(data)
		b.hash = hash[:]
	}
	return hex.EncodeToString(b.hash)
}

// GetLatestBlock returns the latest block
func (b *Block) GetLatestBlock() *Block {
	// TODO: Implement
	return nil
}

// ComputeHash returns the hash of the transaction
func (t *Transaction) ComputeHash() Hash {
	data, _ := json.Marshal(t)
	hash := sha256.Sum256(data)
	return hash[:]
} 