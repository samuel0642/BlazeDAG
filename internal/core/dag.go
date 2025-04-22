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

// GetBlocks returns all blocks in the DAG
func (d *DAG) GetBlocks() map[string]*types.Block {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.blocks
}

// GetHeight returns the current height of the DAG
func (d *DAG) GetHeight() uint64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	maxHeight := uint64(0)
	for _, block := range d.blocks {
		if block.Header.Height > maxHeight {
			maxHeight = block.Header.Height
		}
	}
	return maxHeight
}

// GetLatestBlockHash returns the hash of the latest block
func (d *DAG) GetLatestBlockHash() []byte {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var latestBlock *types.Block
	var latestHeight uint64

	for _, block := range d.blocks {
		if block.Header.Height > latestHeight {
			latestHeight = block.Header.Height
			latestBlock = block
		}
	}

	if latestBlock == nil {
		return nil
	}

	hashStr := latestBlock.Hash()
	return []byte(hashStr)
} 