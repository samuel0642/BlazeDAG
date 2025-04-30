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
	mp.transactions[string(tx.From)] = tx
	log.Printf("Added transaction to mempool - From: %x, To: %x, Value: %d, Nonce: %d", 
		tx.From, tx.To, tx.Value, tx.Nonce)
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
		delete(mp.transactions, string(tx.From))
		log.Printf("Removed transaction from mempool - From: %x, To: %x, Value: %d, Nonce: %d", 
			tx.From, tx.To, tx.Value, tx.Nonce)
	}
}

// BlockProcessor handles block creation and certification
type BlockProcessor struct {
	config  *Config
	state   *State
	dag     *DAG
	mempool *Mempool
	mu      sync.RWMutex
}

// NewBlockProcessor creates a new block processor
func NewBlockProcessor(config *Config, state *State, dag *DAG) *BlockProcessor {
	return &BlockProcessor{
		config:  config,
		state:   state,
		dag:     dag,
		mempool: NewMempool(),
	}
}

// CreateBlock creates a new block
func (bp *BlockProcessor) CreateBlock(round types.Round) (*types.Block, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Get transactions from mempool
	txs := bp.mempool.GetTransactions()
	log.Printf("Creating new block with %d transactions for round %d", len(txs), round)

	// Create block header
	header := &types.BlockHeader{
		Version:    1,
		Timestamp:  time.Now(),
		Round:      round,
		Wave:       types.Wave(bp.state.CurrentWave),
		Height:     types.BlockNumber(bp.getNextHeight()),
		ParentHash: bp.getParentHash(),
		References: bp.selectReferences(),
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

	// Remove processed transactions from mempool
	bp.mempool.RemoveTransactions(txs)
	log.Printf("Block created successfully with %d transactions", len(txs))

	return block, nil
}

// getNextHeight returns the next block height
func (bp *BlockProcessor) getNextHeight() uint64 {
	if bp.state.LatestBlock == nil {
		return 0
	}
	return uint64(bp.state.LatestBlock.Header.Height + 1)
}

// getParentHash returns the parent block hash
func (bp *BlockProcessor) getParentHash() types.Hash {
	if bp.state.LatestBlock == nil {
		return types.Hash{}
	}
	return bp.state.LatestBlock.ComputeHash()
}

// selectReferences selects references for the new block
func (bp *BlockProcessor) selectReferences() []*types.Reference {
	// Get latest blocks from DAG
	latestBlocks := bp.dag.GetRecentBlocks(10) // Get 10 most recent blocks
	references := make([]*types.Reference, 0, len(latestBlocks))

	for _, block := range latestBlocks {
		references = append(references, &types.Reference{
			BlockHash: block.ComputeHash(),
			Round:     block.Header.Round,
			Wave:      block.Header.Wave,
			Type:      types.ReferenceTypeStandard,
		})
	}

	return references
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