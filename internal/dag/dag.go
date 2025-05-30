package dag

import (
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// DAG represents a directed acyclic graph of blocks
type DAG struct {
	blocks     map[string]*types.Block
	references map[string][]types.Hash
	mu         sync.RWMutex
	height     types.BlockNumber
}

// NewDAG creates a new DAG
func NewDAG() *DAG {
	return &DAG{
		blocks:     make(map[string]*types.Block),
		references: make(map[string][]types.Hash),
		height:     0,
	}
}

// AddBlock adds a block to the DAG
func (d *DAG) AddBlock(block *types.Block) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if block already exists
	blockHash := block.ComputeHash()
	if _, exists := d.blocks[string(blockHash)]; exists {
		return fmt.Errorf("block already exists")
	}

	// Add block to DAG
	d.blocks[string(blockHash)] = block

	// Update height if needed
	if block.Header.Height > d.height {
		d.height = block.Header.Height
	}

	// Add references
	for _, ref := range block.Header.References {
		refHash := string(ref.BlockHash)
		if _, exists := d.references[refHash]; !exists {
			d.references[refHash] = make([]types.Hash, 0)
		}
		d.references[refHash] = append(d.references[refHash], blockHash)
	}

	// Log block addition
	log.Printf("Added block to DAG - Hash: %s, Height: %d, Validator: %s, References: %d",
		string(blockHash), block.Header.Height, block.Header.Validator, len(block.Header.References))

	return nil
}

// GetBlock returns a block by its hash
func (d *DAG) GetBlock(hash types.Hash) (*types.Block, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	block, exists := d.blocks[string(hash)]
	if !exists {
		return nil, fmt.Errorf("block not found")
	}

	return block, nil
}

// GetRecentBlocks gets the most recent blocks in the DAG
func (d *DAG) GetRecentBlocks(count int) []*types.Block {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Get all blocks
	blocks := make([]*types.Block, 0, len(d.blocks))
	for _, block := range d.blocks {
		blocks = append(blocks, block)
	}

	// Sort by height, wave, and round in descending order
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].Header.Height != blocks[j].Header.Height {
			return blocks[i].Header.Height > blocks[j].Header.Height
		}
		if blocks[i].Header.Wave != blocks[j].Header.Wave {
			return blocks[i].Header.Wave > blocks[j].Header.Wave
		}
		if blocks[i].Header.Round != blocks[j].Header.Round {
			return blocks[i].Header.Round > blocks[j].Header.Round
		}
		// If all else is equal, sort by validator to ensure consistent ordering
		return blocks[i].Header.Validator > blocks[j].Header.Validator
	})

	// Return the most recent blocks
	if count > len(blocks) {
		count = len(blocks)
	}
	return blocks[:count]
}

// GetBlocksByHeight gets all blocks at a given height
func (d *DAG) GetBlocksByHeight(height types.BlockNumber) []*types.Block {
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

// GetBlockCount gets the number of blocks in the DAG
func (d *DAG) GetBlockCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.blocks)
}

// GetHeight returns the current height of the DAG
func (d *DAG) GetHeight() types.BlockNumber {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.height
}
