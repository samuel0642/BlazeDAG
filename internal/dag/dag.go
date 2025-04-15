package dag

import (
	"errors"

	"github.com/samuel0642/BlazeDAG/internal/types"
)

// DAG represents the Directed Acyclic Graph structure
type DAG struct {
	blocks          map[string]*types.Block
	tips            map[string]bool
	references      map[string][]types.Reference
	children        map[string][]string
	committedBlocks map[string]bool
	pendingBlocks   map[string]bool
	causalOrder     map[string]uint64
}

// NewDAG creates a new DAG instance
func NewDAG() *DAG {
	return &DAG{
		blocks:          make(map[string]*types.Block),
		tips:            make(map[string]bool),
		references:      make(map[string][]types.Reference),
		children:        make(map[string][]string),
		committedBlocks: make(map[string]bool),
		pendingBlocks:   make(map[string]bool),
		causalOrder:     make(map[string]uint64),
	}
}

// AddBlock adds a new block to the DAG
func (d *DAG) AddBlock(block *types.Block) error {
	blockHash := string(block.Hash())
	
	// Store the block
	d.blocks[blockHash] = block
	
	// Store references
	d.references[blockHash] = block.References
	
	// Update children references and tips
	for _, ref := range block.References {
		refHash := string(ref.BlockHash)
		d.children[refHash] = append(d.children[refHash], blockHash)
		// Remove referenced blocks from tips
		delete(d.tips, refHash)
	}
	
	// Add to pending blocks
	d.pendingBlocks[blockHash] = true
	
	// Add to tips if it has no children
	d.tips[blockHash] = true
	
	return nil
}

// GetBlock retrieves a block by its hash
func (d *DAG) GetBlock(hash []byte) (*types.Block, bool) {
	block, exists := d.blocks[string(hash)]
	return block, exists
}

// GetReferences returns the references of a block
func (d *DAG) GetReferences(hash []byte) []types.Reference {
	return d.references[string(hash)]
}

// GetChildren returns the children of a block
func (d *DAG) GetChildren(hash []byte) [][]byte {
	children := d.children[string(hash)]
	result := make([][]byte, len(children))
	for i, child := range children {
		result[i] = []byte(child)
	}
	return result
}

// GetTips returns the current tips of the DAG
func (d *DAG) GetTips() [][]byte {
	tips := make([][]byte, 0, len(d.tips))
	for tip := range d.tips {
		tips = append(tips, []byte(tip))
	}
	return tips
}

// CommitBlock marks a block as committed
func (d *DAG) CommitBlock(hash []byte) error {
	blockHash := string(hash)
	
	// Check if block exists
	if _, exists := d.blocks[blockHash]; !exists {
		return ErrBlockNotFound
	}
	
	// Mark as committed
	d.committedBlocks[blockHash] = true
	delete(d.pendingBlocks, blockHash)
	
	// Update causal order
	d.updateCausalOrder(blockHash)
	
	return nil
}

// IsCommitted checks if a block is committed
func (d *DAG) IsCommitted(hash []byte) bool {
	return d.committedBlocks[string(hash)]
}

// GetCausalOrder returns the causal order of a block
func (d *DAG) GetCausalOrder(hash []byte) (uint64, bool) {
	order, exists := d.causalOrder[string(hash)]
	return order, exists
}

// updateCausalOrder updates the causal ordering of blocks
func (d *DAG) updateCausalOrder(blockHash string) {
	// Get the maximum causal order of references
	maxOrder := uint64(0)
	for _, ref := range d.references[blockHash] {
		if order, exists := d.causalOrder[string(ref.BlockHash)]; exists && order > maxOrder {
			maxOrder = order
		}
	}
	
	// Set the causal order of this block
	d.causalOrder[blockHash] = maxOrder + 1
}

// Errors
var (
	ErrBlockNotFound = errors.New("block not found")
) 