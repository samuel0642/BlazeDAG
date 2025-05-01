package core

import (
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/storage"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// BlockCreator handles block creation
type BlockCreator struct {
	config  *Config
	state   *types.State
	storage *storage.Storage
}

// NewBlockCreator creates a new block creator
func NewBlockCreator(config *Config, state *types.State, storage *storage.Storage) *BlockCreator {
	return &BlockCreator{
		config:  config,
		state:   state,
		storage: storage,
	}
}

// CreateBlock creates a new block
func (bc *BlockCreator) CreateBlock() (*types.Block, error) {
	// Create block header
	header := &types.BlockHeader{
		Version:    1,
		Timestamp:  time.Now(),
		Round:      types.Round(bc.state.CurrentRound),
		Wave:       types.Wave(bc.state.CurrentWave),
		Height:     types.BlockNumber(bc.getNextHeight()),
		ParentHash: bc.getParentHash(),
		References: bc.selectReferences(),
		StateRoot:  bc.calculateStateRoot(),
		Validator:  bc.config.NodeID,
	}

	// Create block body
	body := &types.BlockBody{
		Transactions: bc.collectTransactions(),
		Receipts:     make([]*types.Receipt, 0),
		Events:       make([]*types.Event, 0),
	}

	// Create block
	block := &types.Block{
		Header: header,
		Body:   body,
	}

	// Sign block
	if err := bc.signBlock(block); err != nil {
		return nil, err
	}

	// Save block to storage
	if err := bc.storage.SaveBlock(block); err != nil {
		return nil, err
	}

	return block, nil
}

// getNextHeight returns the next block height
func (bc *BlockCreator) getNextHeight() uint64 {
	if bc.state.LatestBlock == nil {
		return 0
	}
	return uint64(bc.state.LatestBlock.Header.Height + 1)
}

// getParentHash returns the parent block hash
func (bc *BlockCreator) getParentHash() types.Hash {
	if bc.state.LatestBlock == nil {
		return types.Hash{}
	}
	return bc.state.LatestBlock.ComputeHash()
}

// selectReferences selects references for the new block
func (bc *BlockCreator) selectReferences() []*types.Reference {
	references := make([]*types.Reference, 0)
	
	// Get the latest blocks from storage
	latestBlocks, err := bc.storage.GetLatestBlocks(5) // Get last 5 blocks
	if err != nil {
		return references
	}

	// Create references to the latest blocks
	for _, block := range latestBlocks {
		ref := &types.Reference{
			BlockHash: block.ComputeHash(),
			Round:     block.Header.Round,
			Wave:      block.Header.Wave,
		}
		references = append(references, ref)
	}

	return references
}

// calculateStateRoot calculates the state root
func (bc *BlockCreator) calculateStateRoot() types.Hash {
	return bc.state.ComputeRootHash()
}

// collectTransactions collects transactions for the new block
func (bc *BlockCreator) collectTransactions() []*types.Transaction {
	// Get transactions from mempool
	txs, err := bc.storage.LoadMempool()
	if err != nil {
		return make([]*types.Transaction, 0)
	}

	// Clear mempool after collecting transactions
	if err := bc.storage.SaveMempool(make([]*types.Transaction, 0)); err != nil {
		return txs
	}

	return txs
}

// signBlock signs the block
func (bc *BlockCreator) signBlock(block *types.Block) error {
	// TODO: Implement proper block signing with validator key
	// For now, just set a dummy signature
	block.Header.Signature = types.Signature{
		Validator:  bc.config.NodeID,
		Signature:  []byte("dummy_signature"),
		Timestamp:  time.Now(),
	}
	return nil
} 