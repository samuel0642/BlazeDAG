package dag

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"time"
)

// Block represents a block in the DAG
type Block struct {
	Hash       string
	ParentHash string
	References []string
	Timestamp  time.Time
	Data       []byte
	Signature  []byte
}

// DAG represents the directed acyclic graph
type DAG struct {
	blocks     map[string]*Block
	references map[string][]string
	mu         sync.RWMutex
}

// NewDAG creates a new DAG
func NewDAG() *DAG {
	return &DAG{
		blocks:     make(map[string]*Block),
		references: make(map[string][]string),
	}
}

// AddBlock adds a block to the DAG
func (d *DAG) AddBlock(block *Block) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.blocks[block.Hash] = block
	d.references[block.Hash] = block.References
}

// GetBlock returns a block by hash
func (d *DAG) GetBlock(hash string) *Block {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.blocks[hash]
}

// GetReferences returns the references of a block
func (d *DAG) GetReferences(hash string) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.references[hash]
}

// HasBlock checks if a block exists
func (d *DAG) HasBlock(hash string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, exists := d.blocks[hash]
	return exists
}

// GetTips returns the tips of the DAG
func (d *DAG) GetTips() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tips := make([]string, 0)
	for hash := range d.blocks {
		isTip := true
		for _, refs := range d.references {
			for _, ref := range refs {
				if ref == hash {
					isTip = false
					break
				}
			}
			if !isTip {
				break
			}
		}
		if isTip {
			tips = append(tips, hash)
		}
	}
	return tips
}

// GetAncestors returns all ancestors of a block
func (d *DAG) GetAncestors(hash string) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ancestors := make([]string, 0)
	visited := make(map[string]bool)

	var getAncestors func(string)
	getAncestors = func(h string) {
		if visited[h] {
			return
		}
		visited[h] = true

		block := d.blocks[h]
		if block == nil {
			return
		}

		// Add parent
		if block.ParentHash != "" {
			ancestors = append(ancestors, block.ParentHash)
			getAncestors(block.ParentHash)
		}

		// Add references
		for _, ref := range block.References {
			ancestors = append(ancestors, ref)
			getAncestors(ref)
		}
	}

	getAncestors(hash)
	return ancestors
}

// GetDescendants returns all descendants of a block
func (d *DAG) GetDescendants(hash string) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	descendants := make([]string, 0)
	visited := make(map[string]bool)

	var getDescendants func(string)
	getDescendants = func(h string) {
		if visited[h] {
			return
		}
		visited[h] = true

		// Check all blocks for references to this block
		for blockHash, refs := range d.references {
			for _, ref := range refs {
				if ref == h {
					descendants = append(descendants, blockHash)
					getDescendants(blockHash)
					break
				}
			}
		}
	}

	getDescendants(hash)
	return descendants
}

// CalculateBlockHash calculates the hash of a block
func CalculateBlockHash(block *Block) string {
	h := sha256.New()

	// Hash parent hash
	h.Write([]byte(block.ParentHash))

	// Hash references
	for _, ref := range block.References {
		h.Write([]byte(ref))
	}

	// Hash timestamp
	binary.Write(h, binary.LittleEndian, block.Timestamp.UnixNano())

	// Hash data
	h.Write(block.Data)

	return string(h.Sum(nil))
}

// VerifyBlock verifies a block
func (d *DAG) VerifyBlock(block *Block) bool {
	// Verify hash
	if CalculateBlockHash(block) != block.Hash {
		return false
	}

	// Verify parent exists
	if block.ParentHash != "" && !d.HasBlock(block.ParentHash) {
		return false
	}

	// Verify references exist
	for _, ref := range block.References {
		if !d.HasBlock(ref) {
			return false
		}
	}

	return true
} 