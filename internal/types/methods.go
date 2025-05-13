package types

import (
	// "bytes"
	"crypto/sha256"
	// "encoding/hex"
	// "encoding/gob"
	"encoding/json"
	// "fmt"
	"encoding/binary"
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
	// Create a deterministic byte slice for hashing
	data := make([]byte, 0)
	data = append(data, []byte(t.From)...)
	data = append(data, []byte(t.To)...)
	data = append(data, uint64ToBytes(uint64(t.Nonce))...)
	data = append(data, uint64ToBytes(uint64(t.Value))...)
	data = append(data, uint64ToBytes(t.GasLimit)...)
	data = append(data, uint64ToBytes(t.GasPrice)...)
	data = append(data, t.Data...)
	
	// Add timestamp as bytes
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(t.Timestamp.UnixNano()))
	data = append(data, timestampBytes...)

	// Compute SHA-256 hash
	hash := sha256.Sum256(data)
	t.hash = hash[:]
	return hash[:]
} 