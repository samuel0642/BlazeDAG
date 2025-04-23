package core

import (
	"fmt"
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// DAG represents a directed acyclic graph of blocks
type DAG struct {
	blocks     map[string]*types.Block
	references map[string][]types.Reference
	mu         sync.RWMutex
	height     types.Height
}

// NewDAG creates a new DAG
func NewDAG() *DAG {
	return &DAG{
		blocks:     make(map[string]*types.Block),
		references: make(map[string][]types.Reference),
		height:     0,
	}
}

// AddBlock adds a block to the DAG
func (d *DAG) AddBlock(block *types.Block) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	hash := block.ComputeHash()
	if _, exists := d.blocks[hash]; exists {
		return fmt.Errorf("block already exists")
	}

	d.blocks[hash] = block
	if block.Header.Height > d.height {
		d.height = block.Header.Height
	}

	// Add references
	if block.Body != nil {
		d.references[hash] = block.Body.References
	}

	return nil
}

// GetBlock returns a block by its hash
func (d *DAG) GetBlock(hash string) (*types.Block, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	block, exists := d.blocks[hash]
	if !exists {
		return nil, fmt.Errorf("block not found")
	}

	return block, nil
}

// GetBlockByHeight returns a block by its height
func (d *DAG) GetBlockByHeight(height types.Height) (*types.Block, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, block := range d.blocks {
		if block.Header.Height == height {
			return block, nil
		}
	}

	return nil, fmt.Errorf("block not found at height %d", height)
}

// GetLatestHeight returns the latest block height
func (d *DAG) GetLatestHeight() types.Height {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.height
}

// GetBlocks returns all blocks in the DAG
func (d *DAG) GetBlocks() []*types.Block {
	d.mu.RLock()
	defer d.mu.RUnlock()

	blocks := make([]*types.Block, 0, len(d.blocks))
	for _, block := range d.blocks {
		blocks = append(blocks, block)
	}

	return blocks
}

// GetHeight returns the current height of the DAG
func (d *DAG) GetHeight() types.Height {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.height
}

// GetMaxHeight gets the maximum height in the DAG
func (d *DAG) GetMaxHeight() types.Height {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var maxHeight types.Height
	for _, block := range d.blocks {
		if block.Header.Height > maxHeight {
			maxHeight = block.Header.Height
		}
	}

	return maxHeight
}

// GetBlocksByHeight gets all blocks at a given height
func (d *DAG) GetBlocksByHeight(height types.Height) []*types.Block {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var blocks []*types.Block
	for _, block := range d.blocks {
		if block.Header.Height == height {
			blocks = append(blocks, block)
		}
	}

	return blocks
}

// GetAllBlocks gets all blocks in the DAG
func (d *DAG) GetAllBlocks() []*types.Block {
	d.mu.RLock()
	defer d.mu.RUnlock()

	blocks := make([]*types.Block, 0, len(d.blocks))
	for _, block := range d.blocks {
		blocks = append(blocks, block)
	}

	return blocks
}

// GetBlockCount gets the number of blocks in the DAG
func (d *DAG) GetBlockCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.blocks)
}

// GetReferences gets the references for a block
func (d *DAG) GetReferences(hash string) []types.Reference {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.references[hash]
}

// GetRecentBlocks returns the most recent blocks
func (d *DAG) GetRecentBlocks(count int) []*types.Block {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Get all blocks
	blocks := make([]*types.Block, 0, len(d.blocks))
	for _, block := range d.blocks {
		blocks = append(blocks, block)
	}

	// Sort by height in descending order
	for i := 0; i < len(blocks); i++ {
		for j := i + 1; j < len(blocks); j++ {
			if blocks[i].Header.Height < blocks[j].Header.Height {
				blocks[i], blocks[j] = blocks[j], blocks[i]
			}
		}
	}

	// Return the most recent blocks
	if count > len(blocks) {
		count = len(blocks)
	}
	return blocks[:count]
} 