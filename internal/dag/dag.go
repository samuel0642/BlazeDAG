package dag

import (
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// DAG is a wrapper for core.DAG
type DAG struct {
	coreDag *core.DAG
}

// NewDAG returns a new DAG instance that uses the singleton core.DAG
func NewDAG() *DAG {
	return &DAG{
		coreDag: core.GetDAG(),
	}
}

// AddBlock adds a block to the DAG
func (d *DAG) AddBlock(block *types.Block) error {
	return d.coreDag.AddBlock(block)
}

// GetBlock returns a block by its hash
func (d *DAG) GetBlock(hash types.Hash) (*types.Block, error) {
	return d.coreDag.GetBlock(hash)
}

// GetRecentBlocks gets the most recent blocks in the DAG
func (d *DAG) GetRecentBlocks(count int) []*types.Block {
	return d.coreDag.GetRecentBlocks(count)
}

// GetBlocksByHeight gets all blocks at a given height
func (d *DAG) GetBlocksByHeight(height types.BlockNumber) []*types.Block {
	return d.coreDag.GetBlocksByHeight(height)
}

// GetBlockCount gets the number of blocks in the DAG
func (d *DAG) GetBlockCount() int {
	return d.coreDag.GetBlockCount()
}

// GetHeight returns the current height of the DAG
func (d *DAG) GetHeight() types.BlockNumber {
	return d.coreDag.GetHeight()
}
