package types

import (
	"bytes"
	"crypto/sha256"
	// "encoding/hex"
	"encoding/gob"
	"encoding/json"
)

// ComputeHash returns the hash of the block
func (b *Block) ComputeHash() Hash {
	if b.hash == nil {
		// Create a copy of the block without the hash field
		blockCopy := &Block{
			Header:      b.Header,
			Body:        b.Body,
			Certificate: b.Certificate,
		}
		var buf bytes.Buffer
		gob.NewEncoder(&buf).Encode(blockCopy)
		hash := sha256.Sum256(buf.Bytes())
		b.hash = hash[:]
	}
	return b.hash
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