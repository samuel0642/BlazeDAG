package dag

import (
	"sync"

	"BlazeDAG/internal/types"
)

// DAG represents the directed acyclic graph structure
type DAG struct {
	mu sync.RWMutex

	// Blocks
	blocks map[string]*types.Block
	tips   map[string]bool

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
		causalOrder:    make(map[string]uint64),
	}
}

// AddBlock adds a block to the DAG
func (d *DAG) AddBlock(block *types.Block) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Convert block to string for map keys
	blockHash := string(block.Header.ParentHash)

	// Add block
	d.blocks[blockHash] = block

	// Add references
	for _, ref := range block.References {
		refHash := string(ref.BlockHash)
		d.references[blockHash] = append(d.references[blockHash], refHash)
		d.children[refHash] = append(d.children[refHash], blockHash)
	}

	// Update tips
	d.updateTips(blockHash)

	return nil
}

// GetBlock gets a block from the DAG
func (d *DAG) GetBlock(hash []byte) (*types.Block, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	block, ok := d.blocks[string(hash)]
	return block, ok
}

// GetTips gets the current tips of the DAG
func (d *DAG) GetTips() [][]byte {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tips := make([][]byte, 0, len(d.tips))
	for tip := range d.tips {
		tips = append(tips, []byte(tip))
	}
	return tips
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

// GetChildren gets the children of a block
func (d *DAG) GetChildren(hash []byte) [][]byte {
	d.mu.RLock()
	defer d.mu.RUnlock()

	children := d.children[string(hash)]
	childHashes := make([][]byte, len(children))
	for i, child := range children {
		childHashes[i] = []byte(child)
	}
	return childHashes
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

// Errors
var (
	ErrBlockNotFound = errors.New("block not found")
) 