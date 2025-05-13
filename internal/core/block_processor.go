package core

import (
	"time"
	"sync"
	"crypto/sha256"
	"encoding/json"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// Mempool stores pending transactions
type Mempool struct {
	transactions map[string]*types.Transaction
	mu           sync.RWMutex
}

// NewMempool creates a new mempool
func NewMempool() *Mempool {
	return &Mempool{
		transactions: make(map[string]*types.Transaction),
	}
}

// AddTransaction adds a transaction to the mempool
func (mp *Mempool) AddTransaction(tx *types.Transaction) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	
	// Set timestamp if not set
	if tx.Timestamp.IsZero() {
		tx.Timestamp = time.Now()
	}
	
	txHash := string(tx.GetHash())
	mp.transactions[txHash] = tx
	log.Printf("Added transaction to mempool - Hash: %x, From: %x, To: %x, Value: %d, Nonce: %d", 
		tx.GetHash(), tx.From, tx.To, tx.Value, tx.Nonce)
}

// GetTransactions returns all transactions in the mempool
func (mp *Mempool) GetTransactions() []*types.Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	txs := make([]*types.Transaction, 0, len(mp.transactions))
	for _, tx := range mp.transactions {
		txs = append(txs, tx)
	}
	log.Printf("Retrieved %d transactions from mempool", len(txs))
	return txs
}

// RemoveTransactions removes transactions from the mempool
func (mp *Mempool) RemoveTransactions(txs []*types.Transaction) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	for _, tx := range txs {
		txHash := string(tx.GetHash())
		delete(mp.transactions, txHash)
		log.Printf("Removed transaction from mempool - Hash: %x, From: %x, To: %x, Value: %d, Nonce: %d", 
			tx.GetHash(), tx.From, tx.To, tx.Value, tx.Nonce)
	}
}

// BlockProcessor handles block processing
type BlockProcessor struct {
	config       *Config
	stateManager *StateManager
	dag          *DAG
	mempool      *Mempool
	mu           sync.RWMutex
}

// NewBlockProcessor creates a new block processor
func NewBlockProcessor(config *Config, stateManager *StateManager, dag *DAG) *BlockProcessor {
	bp := &BlockProcessor{
		config:       config,
		stateManager: stateManager,
		dag:          dag,
		mempool:      NewMempool(),
	}

	// Load existing blocks into DAG
	blocks, err := stateManager.storage.GetLatestBlocks(100) // Load fewer blocks
	if err != nil {
		log.Printf("Warning: Failed to load blocks from storage: %v", err)
	} else {
		// Filter out invalid blocks
		validBlocks := make([]*types.Block, 0)
		for _, block := range blocks {
			// Skip blocks with height 0 and no references
			if block.Header.Height == 0 && len(block.Header.References) == 0 {
				continue
			}
			validBlocks = append(validBlocks, block)
		}

		// // Add valid blocks to DAG
		// for _, block := range validBlocks {
		// 	if err := dag.AddBlock(block); err != nil {
		// 		log.Printf("Warning: Failed to add block %s to DAG: %v", block.ComputeHash(), err)
		// 	}
		// }
		log.Printf("Loaded %d valid blocks into DAG", len(validBlocks))
	}

	return bp
}

// CreateBlock creates a new block
func (bp *BlockProcessor) CreateBlock(round types.Round, currentWave types.Wave) (*types.Block, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Get transactions from mempool
	txs := bp.mempool.GetTransactions()
	log.Printf("Creating new block with %d transactions for round %d", len(txs), round)

	// Find all blocks from the previous wave (currentWave-1)
	prevWave := currentWave - 1
	prevWaveBlocks := make([]*types.Block, 0)
	allBlocks := bp.dag.GetRecentBlocks(10) // You may need to implement this if not present
	
	for _, block := range allBlocks {
		if block.Header.Wave == prevWave {
			prevWaveBlocks = append(prevWaveBlocks, block)
		}
	}
	// log.Printf("pppppppppppppppppppppppppppppppppppppppAll blocks: %d, prevWave: %d, prevWaveBlocks: %d", len(allBlocks), prevWave, len(prevWaveBlocks))


	references := make([]*types.Reference, 0, len(prevWaveBlocks))
	for _, block := range prevWaveBlocks {
		// Reference all blocks from the previous wave
		references = append(references, &types.Reference{
			BlockHash: block.ComputeHash(),
			Round:     block.Header.Round,
			Wave:      block.Header.Wave,
			Type:      types.ReferenceTypeStandard,
		})
	}

	// Create block header
	header := &types.BlockHeader{
		Version:    1,
		Timestamp:  time.Now(),
		Round:      round,
		Wave:       currentWave,
		Height:     types.BlockNumber(bp.getNextHeight()),
		ParentHash: bp.getParentHash(),
		References: references,
		StateRoot:  bp.calculateStateRoot(txs),
		Validator:  bp.config.NodeID,
	}

	// Create block body
	body := &types.BlockBody{
		Transactions: txs,
		Receipts:     make([]*types.Receipt, 0),
		Events:       make([]*types.Event, 0),
	}

	// Create block
	block := &types.Block{
		Header: header,
		Body:   body,
	}

	// Sign block
	if err := bp.signBlock(block); err != nil {
		return nil, err
	}

	// Save block to storage
	if err := bp.stateManager.storage.SaveBlock(block); err != nil {
		return nil, err
	}

	// Add block to DAG
	if err := bp.dag.AddBlock(block); err != nil {
		log.Printf("Warning: Failed to add block to DAG: %v", err)
	}

	// Remove processed transactions from mempool
	bp.mempool.RemoveTransactions(txs)
	log.Printf("Block created successfully with %d transactions and %d references", len(txs), len(references))
	log.Printf("Block details:")
	log.Printf("  Hash: %x", block.ComputeHash())
	log.Printf("  Height: %d", block.Header.Height)
	log.Printf("  Round: %d", block.Header.Round) 
	log.Printf("  Wave: %d", block.Header.Wave)
	log.Printf("  Validator: %s", block.Header.Validator)
	log.Printf("  Parent Hash: %x", block.Header.ParentHash)
	log.Printf("  References: %d", len(block.Header.References))
	log.Printf("  Transactions: %d", len(block.Body.Transactions))
	return block, nil
}

// getNextHeight returns the next block height
func (bp *BlockProcessor) getNextHeight() uint64 {
	state := bp.stateManager.GetState()
	if state.LatestBlock == nil {
		return 0
	}
	return uint64(state.LatestBlock.Header.Height + 1)
}

// getParentHash returns the parent block hash
func (bp *BlockProcessor) getParentHash() types.Hash {
	state := bp.stateManager.GetState()
	if state.LatestBlock == nil {
		return types.Hash{}
	}
	return state.LatestBlock.ComputeHash()
}

// calculateStateRoot calculates the state root based on transactions
func (bp *BlockProcessor) calculateStateRoot(txs []*types.Transaction) types.Hash {
	// Create a map to store state changes
	stateChanges := make(map[string]interface{})

	// Process each transaction
	for _, tx := range txs {
		// For each transaction, update the state
		// This is a simplified example - in a real implementation,
		// you would have more complex state transition logic
		stateChanges[string(tx.From)] = map[string]interface{}{
			"balance": tx.Value,
			"nonce":   tx.Nonce,
		}
	}

	// Serialize state changes to JSON
	stateBytes, err := json.Marshal(stateChanges)
	if err != nil {
		return types.Hash{}
	}

	// Calculate hash of state changes
	hash := sha256.Sum256(stateBytes)
	return types.Hash(hash[:])
}

// signBlock signs the block using ECDSA
func (bp *BlockProcessor) signBlock(block *types.Block) error {
	// Serialize block header for signing
	headerBytes, err := json.Marshal(block.Header)
	if err != nil {
		return err
	}

	// Create a new private key for signing
	// In a real implementation, you would use the node's private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	// Hash the header bytes
	hash := sha256.Sum256(headerBytes)

	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return err
	}

	// Convert signature to bytes
	signature := append(r.Bytes(), s.Bytes()...)

	// Set the signature in the block header
	block.Header.Signature = types.Signature{
		Validator:  bp.config.NodeID,
		Signature:  signature,
		Timestamp:  time.Now(),
	}

	return nil
}

// AddTransaction adds a transaction to the mempool
func (bp *BlockProcessor) AddTransaction(tx *types.Transaction) {
	bp.mempool.AddTransaction(tx)
}

// CreateCertificate creates a certificate for a block
func (bp *BlockProcessor) CreateCertificate(block *types.Block, signatures []*types.Signature) (*types.Certificate, error) {
	if len(signatures) == 0 {
		return nil, types.ErrInvalidCertificate
	}

	// Convert signatures to bytes
	sigBytes := make([][]byte, len(signatures))
	for i, sig := range signatures {
		sigBytes[i] = sig.Signature
	}

	certificate := &types.Certificate{
		BlockHash:  block.ComputeHash(),
		Signatures: sigBytes,
		Round:      block.Header.Round,
		Wave:       block.Header.Wave,
		Timestamp:  time.Now(),
	}

	return certificate, nil
}

// VerifyCertificate verifies a block certificate
func (bp *BlockProcessor) VerifyCertificate(cert *types.Certificate) error {
	if cert == nil || len(cert.BlockHash) == 0 {
		return types.ErrInvalidCertificate
	}

	// Check if block exists
	_, err := bp.dag.GetBlock(cert.BlockHash)
	if err != nil {
		return err
	}

	// TODO: Verify signatures
	// For now, just return nil
	return nil
}

// GetState returns the current state
func (bp *BlockProcessor) GetState() *types.State {
	return bp.stateManager.GetState()
} 