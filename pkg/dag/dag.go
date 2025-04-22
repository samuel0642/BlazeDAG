package dag

import (
	"errors"
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

var (
	ErrBlockExists     = errors.New("block already exists")
	ErrInvalidBlock    = errors.New("invalid block")
	ErrParentNotFound  = errors.New("parent block not found")
)

// DAG represents a directed acyclic graph of blocks
type DAG struct {
	blocks map[string]*types.Block
	tips   map[string]bool
	mu     sync.RWMutex

	// References
	references map[string][]string
	children   map[string][]string

	// State
	committedBlocks map[string]bool
	pendingBlocks   map[string]bool
	causalOrder    map[string]uint64
}

// NewDAG creates a new DAG
func NewDAG() *DAG {
	return &DAG{
		blocks:          make(map[string]*types.Block),
		tips:            make(map[string]bool),
		references:      make(map[string][]string),
		children:        make(map[string][]string),
		committedBlocks: make(map[string]bool),
		pendingBlocks:   make(map[string]bool),
		causalOrder:     make(map[string]uint64),
	}
}

// AddBlock adds a block to the DAG
func (d *DAG) AddBlock(block *types.Block) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	blockHash := string(block.Hash())

	// Check if block already exists
	if _, exists := d.blocks[blockHash]; exists {
		return ErrBlockExists
	}

	// Check parent exists (except for genesis block)
	parentHash := string(block.Header.ParentHash)
	if len(d.blocks) > 0 {
		if _, exists := d.blocks[parentHash]; !exists {
			return ErrParentNotFound
		}
		// Remove parent from tips
		delete(d.tips, parentHash)
	}

	// Add block
	d.blocks[blockHash] = block
	d.tips[blockHash] = true

	// Remove referenced blocks from tips
	for _, ref := range block.References {
		delete(d.tips, string(ref.BlockHash))
	}

	// Add references
	for _, ref := range block.References {
		refHash := string(ref.BlockHash)
		d.references[blockHash] = append(d.references[blockHash], refHash)
		d.children[refHash] = append(d.children[refHash], blockHash)
	}

	return nil
}

// GetBlock retrieves a block by its hash
func (d *DAG) GetBlock(hash []byte) (*types.Block, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	block, exists := d.blocks[string(hash)]
	return block, exists
}

// GetTips returns all tip blocks (blocks with no children)
func (d *DAG) GetTips() [][]byte {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tips := make([][]byte, 0, len(d.tips))
	for hash := range d.tips {
		tips = append(tips, []byte(hash))
	}
	return tips
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

// GetBlocksByHeight returns all blocks at a specific height
func (d *DAG) GetBlocksByHeight(height uint64) []*types.Block {
	d.mu.RLock()
	defer d.mu.RUnlock()

	blocks := make([]*types.Block, 0)
	for _, block := range d.blocks {
		if block.Header.Height == height {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// GetParent returns the parent block of a given block
func (d *DAG) GetParent(block *types.Block) (*types.Block, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	parent, exists := d.blocks[string(block.Header.ParentHash)]
	return parent, exists
}

// GetChildren returns all child blocks of a given block
func (d *DAG) GetChildren(block *types.Block) []*types.Block {
	d.mu.RLock()
	defer d.mu.RUnlock()

	children := make([]*types.Block, 0)
	blockHash := block.Hash()
	for _, b := range d.blocks {
		if string(b.Header.ParentHash) == string(blockHash) {
			children = append(children, b)
		}
	}
	return children
}

// GetReferences gets the references of a block
func (d *DAG) GetReferences(hash []byte) [][]byte {
	d.mu.RLock()
	defer d.mu.RUnlock()

	refs := d.references[string(hash)]
	refHashes := make([][]byte, len(refs))
	for i, ref := range refs {
		refHashes[i] = []byte(ref)
	}
	return refHashes
}

// CommitBlock commits a block
func (d *DAG) CommitBlock(hash []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	blockHash := string(hash)
	if _, ok := d.blocks[blockHash]; !ok {
		return ErrBlockNotFound
	}

	d.committedBlocks[blockHash] = true
	delete(d.pendingBlocks, blockHash)

	// Update causal order
	d.updateCausalOrder(blockHash)

	return nil
}

// IsCommitted checks if a block is committed
func (d *DAG) IsCommitted(hash []byte) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.committedBlocks[string(hash)]
}

// GetCausalOrder gets the causal order of a block
func (d *DAG) GetCausalOrder(hash []byte) (uint64, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	order, ok := d.causalOrder[string(hash)]
	return order, ok
}

// updateTips updates the tips of the DAG
func (d *DAG) updateTips(blockHash string) {
	// Remove block from tips if it was a tip
	delete(d.tips, blockHash)

	// Add block to tips if it has no children
	if len(d.children[blockHash]) == 0 {
		d.tips[blockHash] = true
	}

	// Remove parents from tips if they are no longer tips
	for _, ref := range d.references[blockHash] {
		if len(d.children[ref]) > 0 {
			delete(d.tips, ref)
		}
	}
}

// updateCausalOrder updates the causal order of blocks
func (d *DAG) updateCausalOrder(blockHash string) {
	// Get the maximum causal order of references
	maxOrder := uint64(0)
	for _, ref := range d.references[blockHash] {
		if order, ok := d.causalOrder[ref]; ok && order > maxOrder {
			maxOrder = order
		}
	}

	// Set causal order
	d.causalOrder[blockHash] = maxOrder + 1

	// Update children's causal order
	for _, child := range d.children[blockHash] {
		if d.committedBlocks[child] {
			d.updateCausalOrder(child)
		}
	}
}

// GetBlocks returns all blocks in the DAG
func (d *DAG) GetBlocks() map[string]*types.Block {
	d.mu.RLock()
	defer d.mu.RUnlock()
	blocks := make(map[string]*types.Block, len(d.blocks))
	for k, v := range d.blocks {
		blocks[k] = v
	}
	return blocks
}

// Errors
var (
	ErrBlockNotFound = errors.New("block not found")
) 