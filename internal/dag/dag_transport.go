package dag

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// DAGTransport handles the DAG transport layer - manages DAG rounds independently
type DAGTransport struct {
	dag           *DAG
	currentRound  types.Round
	roundDuration time.Duration
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	
	// Independent DAG operation
	blockBroadcast chan *types.Block
	roundAdvancer  *time.Ticker
	
	// Reference management
	recentBlocks   map[types.Round][]*types.Block
	recentMu       sync.RWMutex
}

// NewDAGTransport creates a new DAG transport that operates independently
func NewDAGTransport(dag *DAG, roundDuration time.Duration) *DAGTransport {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DAGTransport{
		dag:            dag,
		currentRound:   0,
		roundDuration:  roundDuration,
		ctx:            ctx,
		cancel:         cancel,
		blockBroadcast: make(chan *types.Block, 100),
		roundAdvancer:  time.NewTicker(roundDuration),
		recentBlocks:   make(map[types.Round][]*types.Block),
	}
}

// Start begins the independent DAG transport operation
func (dt *DAGTransport) Start() {
	go dt.runDAGTransport()
	log.Printf("DAG Transport started - operating independently at network speed")
}

// Stop stops the DAG transport
func (dt *DAGTransport) Stop() {
	dt.cancel()
	dt.roundAdvancer.Stop()
	close(dt.blockBroadcast)
}

// runDAGTransport runs the DAG transport independently from consensus
func (dt *DAGTransport) runDAGTransport() {
	for {
		select {
		case <-dt.ctx.Done():
			return
		case <-dt.roundAdvancer.C:
			dt.advanceDAGRound()
		case block := <-dt.blockBroadcast:
			dt.processDAGBlock(block)
		}
	}
}

// BroadcastBlock adds a block to the DAG transport for processing
func (dt *DAGTransport) BroadcastBlock(block *types.Block) {
	select {
	case dt.blockBroadcast <- block:
	default:
		log.Printf("DAG Transport: Block broadcast channel full, dropping block")
	}
}

// CreateDAGBlock creates a new DAG block with proper DAG round (independent of consensus wave)
func (dt *DAGTransport) CreateDAGBlock(validator types.Address, transactions []*types.Transaction) *types.Block {
	dt.mu.RLock()
	currentRound := dt.currentRound
	dt.mu.RUnlock()
	
	// Get references to 2f+1 blocks from previous round (Narwhal structure)
	references := dt.getDAGReferences(currentRound)
	
	block := &types.Block{
		Header: &types.BlockHeader{
			Version:    1,
			Timestamp:  time.Now(),
			Round:      currentRound,      // DAG round - progresses independently
			Wave:       0,                 // Wave is set separately by consensus layer
			Height:     dt.dag.GetHeight() + 1,
			References: references,
			Validator:  validator,
		},
		Body: &types.BlockBody{
			Transactions: transactions,
			Receipts:     make([]*types.Receipt, 0),
			Events:       make([]*types.Event, 0),
		},
	}
	
	return block
}

// getDAGReferences gets references to 2f+1 blocks from previous DAG round
func (dt *DAGTransport) getDAGReferences(currentRound types.Round) []*types.Reference {
	dt.recentMu.RLock()
	defer dt.recentMu.RUnlock()
	
	if currentRound == 0 {
		return []*types.Reference{}
	}
	
	prevRound := currentRound - 1
	prevBlocks := dt.recentBlocks[prevRound]
	
	references := make([]*types.Reference, 0)
	// Reference up to 2f+1 blocks from previous round
	maxRefs := len(prevBlocks)
	if maxRefs > 5 { // assuming f=2, so 2f+1=5
		maxRefs = 5
	}
	
	for i := 0; i < maxRefs; i++ {
		ref := &types.Reference{
			BlockHash: prevBlocks[i].ComputeHash(),
			Round:     prevBlocks[i].Header.Round,
			Wave:      prevBlocks[i].Header.Wave,
			Type:      types.ReferenceTypeStandard,
		}
		references = append(references, ref)
	}
	
	return references
}

// processDAGBlock processes a block in the DAG transport layer
func (dt *DAGTransport) processDAGBlock(block *types.Block) {
	// Add to DAG
	if err := dt.dag.AddBlock(block); err != nil {
		log.Printf("DAG Transport: Failed to add block to DAG: %v", err)
		return
	}
	
	// Track block for references
	dt.recentMu.Lock()
	round := block.Header.Round
	if dt.recentBlocks[round] == nil {
		dt.recentBlocks[round] = make([]*types.Block, 0)
	}
	dt.recentBlocks[round] = append(dt.recentBlocks[round], block)
	
	// Clean old rounds (keep last 10 rounds)
	for r := range dt.recentBlocks {
		if r < round-10 {
			delete(dt.recentBlocks, r)
		}
	}
	dt.recentMu.Unlock()
	
	log.Printf("DAG Transport: Added block to round %d - Hash: %s", round, string(block.ComputeHash()))
}

// advanceDAGRound advances the DAG round independently
func (dt *DAGTransport) advanceDAGRound() {
	dt.mu.Lock()
	dt.currentRound++
	newRound := dt.currentRound
	dt.mu.Unlock()
	
	log.Printf("DAG Transport: Advanced to round %d (independent of consensus waves)", newRound)
}

// GetCurrentDAGRound returns the current DAG round
func (dt *DAGTransport) GetCurrentDAGRound() types.Round {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.currentRound
}

// GetUncommittedBlocks returns blocks that haven't been committed by consensus yet
func (dt *DAGTransport) GetUncommittedBlocks() []*types.Block {
	// Return recent blocks from last few rounds that may not be committed
	dt.recentMu.RLock()
	defer dt.recentMu.RUnlock()
	
	currentRound := dt.GetCurrentDAGRound()
	uncommitted := make([]*types.Block, 0)
	
	// Get blocks from last 5 rounds
	for r := currentRound; r > 0 && r >= currentRound-5; r-- {
		if blocks := dt.recentBlocks[r]; blocks != nil {
			uncommitted = append(uncommitted, blocks...)
		}
	}
	
	return uncommitted
}

// GetBlocksInCausalHistory returns all blocks in causal history of given blocks
func (dt *DAGTransport) GetBlocksInCausalHistory(topBlocks []*types.Block) []*types.Block {
	visited := make(map[string]bool)
	result := make([]*types.Block, 0)
	
	var traverse func(*types.Block)
	traverse = func(block *types.Block) {
		hash := string(block.ComputeHash())
		if visited[hash] {
			return
		}
		visited[hash] = true
		result = append(result, block)
		
		// Traverse references
		for _, ref := range block.Header.References {
			if refBlock, err := dt.dag.GetBlock(ref.BlockHash); err == nil {
				traverse(refBlock)
			}
		}
	}
	
	for _, block := range topBlocks {
		traverse(block)
	}
	
	return result
}

// GetBlock returns a block by its hash from the DAG
func (dt *DAGTransport) GetBlock(hash types.Hash) (*types.Block, error) {
	return dt.dag.GetBlock(hash)
} 