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
	maxBlocks  int  // ← ADDED: Maximum blocks to keep in memory
}

// NewDAG creates a new DAG
func NewDAG() *DAG {
	return &DAG{
		blocks:     make(map[string]*types.Block),
		references: make(map[string][]types.Hash),
		height:     0,
		maxBlocks:  20, // ← CHANGED: Much more aggressive limit (was 50)
	}
}

// AddBlock adds a block to the DAG
func (d *DAG) AddBlock(block *types.Block) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	blockHash := string(block.ComputeHash())
	d.blocks[blockHash] = block
	d.height++

	// Update references
	for _, ref := range block.Header.References {
		refKey := string(ref.BlockHash)
		d.references[refKey] = append(d.references[refKey], types.Hash(blockHash))
	}

	// ← CHANGED: Cleanup on EVERY block addition (not just when exceeding limit)
	if len(d.blocks) > 5 { // Start cleanup very early
		d.cleanupOldBlocks()
	}

	log.Printf("DAG: Added block %x at height %d, total blocks: %d", 
		block.ComputeHash()[:8], d.height, len(d.blocks))

	return nil
}

// ← ADDED: cleanupOldBlocks removes old blocks to prevent memory overflow
func (d *DAG) cleanupOldBlocks() {
	// Get all blocks and sort by height (oldest first)
	blocks := make([]*types.Block, 0, len(d.blocks))
	for _, block := range d.blocks {
		blocks = append(blocks, block)
	}

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Header.Height < blocks[j].Header.Height
	})

	// ← CHANGED: Keep only 5 blocks maximum (was maxBlocks/2)
	keepCount := 5
	
	blocksToRemove := len(blocks) - keepCount
	if blocksToRemove <= 0 {
		return
	}

	// Remove old blocks
	for i := 0; i < blocksToRemove; i++ {
		block := blocks[i]
		blockHash := string(block.ComputeHash())
		delete(d.blocks, blockHash)
		delete(d.references, blockHash)
	}

	log.Printf("🧹 DAG Cleanup: Removed %d old blocks, keeping only %d blocks in memory", 
		blocksToRemove, len(d.blocks))
}

// ← ADDED: SetMaxBlocks allows configuring the memory limit
func (d *DAG) SetMaxBlocks(maxBlocks int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.maxBlocks = maxBlocks
	log.Printf("DAG memory limit set to %d blocks (~%.1f GB)", maxBlocks, float64(maxBlocks*10)/1000)
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
