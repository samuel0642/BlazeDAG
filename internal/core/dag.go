package core

import (
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// DAG represents the directed acyclic graph
type DAG struct {
	blocks     map[string]*types.Block
	references map[string][]types.Reference
	mu         sync.RWMutex
}

// NewDAG creates a new DAG instance
func NewDAG() *DAG {
	return &DAG{
		blocks:     make(map[string]*types.Block),
		references: make(map[string][]types.Reference),
	}
}

// AddBlock adds a block to the DAG
func (d *DAG) AddBlock(block *types.Block) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	blockHash := block.Hash()
	d.blocks[blockHash] = block
	d.references[blockHash] = block.References
	return nil
}

// GetBlock retrieves a block from the DAG
func (d *DAG) GetBlock(hash string) (*types.Block, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	block, exists := d.blocks[hash]
	if !exists {
		return nil, types.ErrBlockNotFound
	}
	return block, nil
}

// GetReferences returns the references for a block
func (d *DAG) GetReferences(hash string) ([]types.Reference, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	refs, exists := d.references[hash]
	if !exists {
		return nil, types.ErrBlockNotFound
	}
	return refs, nil
} 