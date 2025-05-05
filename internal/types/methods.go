package types

import (
	// "bytes"
	"crypto/sha256"
	// "encoding/hex"
	// "encoding/gob"
	"encoding/json"
	// "fmt"
)

func mustJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// ComputeHash returns the hash of the block header only
func (b *Block) ComputeHash() Hash {
	// Canonicalize header fields
	if b.Header != nil {
		if b.Header.ParentHash == nil {
			b.Header.ParentHash = []byte{}
		}
		if b.Header.References == nil {
			b.Header.References = []*Reference{}
		}
	}
	// fmt.Printf("BlockHeader to hash: %s\n", mustJSON(b.Header))
	data, _ := json.Marshal(b.Header)
	hash := sha256.Sum256(data)
	return hash[:]
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