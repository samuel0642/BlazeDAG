package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// Mempool stores pending transactions with parallel processing
type Mempool struct {
	transactions map[string]*types.Transaction
	mu           sync.RWMutex
	workers      int
	workChan     chan *types.Transaction
	doneChan     chan bool
}

// NewMempool creates a new mempool with parallel processing
func NewMempool() *Mempool {
	workers := runtime.NumCPU()
	mp := &Mempool{
		transactions: make(map[string]*types.Transaction),
		workers:      workers,
		workChan:     make(chan *types.Transaction, 1000), // Buffer size of 1000
		doneChan:     make(chan bool),
	}

	// Start worker pool
	for i := 0; i < workers; i++ {
		go mp.worker()
	}

	return mp
}

// worker processes transactions in parallel
func (mp *Mempool) worker() {
	for {
		select {
		case tx := <-mp.workChan:
			mp.mu.Lock()
			// Set timestamp if not set
			if tx.Timestamp.IsZero() {
				tx.Timestamp = time.Now()
			}

			txHash := string(tx.GetHash())
			mp.transactions[txHash] = tx
			mp.mu.Unlock()

			log.Printf("Added transaction to mempool - Hash: %x, From: %x, To: %x, Value: %d, Nonce: %d",
				tx.GetHash(), tx.From, tx.To, tx.Value, tx.Nonce)
		case <-mp.doneChan:
			return
		}
	}
}

// AddTransaction adds a transaction to the mempool using parallel processing
func (mp *Mempool) AddTransaction(tx *types.Transaction) {
	mp.workChan <- tx
}

// GetTransactions returns all transactions in the mempool
func (mp *Mempool) GetTransactions() []*types.Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	// Create a slice to store transactions
	txs := make([]*types.Transaction, 0, len(mp.transactions))

	// Use a wait group to process transactions in parallel
	var wg sync.WaitGroup
	txChan := make(chan *types.Transaction, len(mp.transactions))

	// Start workers to process transactions
	for i := 0; i < mp.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for tx := range txChan {
				txs = append(txs, tx)
			}
		}()
	}

	// Send transactions to workers
	for _, tx := range mp.transactions {
		txChan <- tx
	}
	close(txChan)

	// Wait for all workers to finish
	wg.Wait()

	log.Printf("Retrieved %d transactions from mempool", len(txs))
	return txs
}

// RemoveTransactions removes transactions from the mempool in parallel
func (mp *Mempool) RemoveTransactions(txs []*types.Transaction) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Create a wait group for parallel processing
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, mp.workers)

	for _, tx := range txs {
		wg.Add(1)
		go func(tx *types.Transaction) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			txHash := string(tx.GetHash())
			delete(mp.transactions, txHash)
			log.Printf("Removed transaction from mempool - Hash: %x, From: %x, To: %x, Value: %d, Nonce: %d",
				tx.GetHash(), tx.From, tx.To, tx.Value, tx.Nonce)
		}(tx)
	}

	wg.Wait()
}

// Stop stops the mempool workers
func (mp *Mempool) Stop() {
	for i := 0; i < mp.workers; i++ {
		mp.doneChan <- true
	}
	close(mp.workChan)
	close(mp.doneChan)
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

	// Print all transactions in mempool with their states
	log.Printf("\n=== Current Mempool State ===")
	allTxs := bp.mempool.GetTransactions()
	log.Printf("Total transactions in mempool: %d", len(allTxs))
	for _, tx := range allTxs {
		stateStr := "Unknown"
		switch tx.State {
		case types.TransactionStatePending:
			stateStr = "Pending"
		case types.TransactionStateIncluded:
			stateStr = "Included"
		case types.TransactionStateCommitted:
			stateStr = "Committed"
		}
		log.Printf("Transaction: %x", tx.GetHash())
		log.Printf("  From: %s", tx.From)
		log.Printf("  To: %s", tx.To)
		log.Printf("  Value: %d", tx.Value)
		log.Printf("  Nonce: %d", tx.Nonce)
		log.Printf("  State: %s", stateStr)
		log.Printf("  Timestamp: %s", tx.Timestamp.Format(time.RFC3339))
		log.Printf("---")
	}
	log.Printf("=== End Mempool State ===\n")

	// Filter only pending transactions
	pendingTxs := make([]*types.Transaction, 0)
	for _, tx := range allTxs {
		if tx.State == types.TransactionStatePending {
			pendingTxs = append(pendingTxs, tx)
		}
	}

	log.Printf("Creating new block with %d pending transactions for round %d", len(pendingTxs), round)

	// Set initial state for transactions
	for _, tx := range pendingTxs {
		tx.State = types.TransactionStateIncluded
	}

	// Find all blocks from the previous wave (currentWave-1)
	prevWave := currentWave - 1
	prevWaveBlocks := make([]*types.Block, 0)
	allBlocks := bp.dag.GetRecentBlocks(10)

	// Check existing blocks in the current wave to avoid duplicates
	for _, block := range allBlocks {
		if block.Header.Wave == currentWave && block.Header.Validator == bp.config.NodeID {
			log.Printf("IMPORTANT: Validator %s already has a block in wave %d, skipping creation",
				bp.config.NodeID, currentWave)
			return nil, fmt.Errorf("validator already has block in current wave")
		}
	}

	// Debug: Log all blocks in DAG to see what's being received from other validators
	log.Printf("\n=== Current DAG State ===")
	log.Printf("Total blocks in DAG: %d", bp.dag.GetBlockCount())
	for i, block := range allBlocks {
		if i >= 10 {
			log.Printf("... and %d more blocks", len(allBlocks)-10)
			break
		}
		log.Printf("Block #%d: Hash=%x, Height=%d, Wave=%d, Round=%d, Validator=%s",
			i+1, block.ComputeHash(), block.Header.Height, block.Header.Wave,
			block.Header.Round, block.Header.Validator)
	}
	log.Printf("=== End DAG State ===\n")

	for _, block := range allBlocks {
		if block.Header.Wave == prevWave {
			prevWaveBlocks = append(prevWaveBlocks, block)
		}
	}

	log.Printf("Found %d blocks from previous wave %d", len(prevWaveBlocks), prevWave)

	references := make([]*types.Reference, 0, len(prevWaveBlocks))
	for _, block := range prevWaveBlocks {
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
		StateRoot:  bp.calculateStateRoot(pendingTxs),
		Validator:  bp.config.NodeID,
	}

	// Create block body
	body := &types.BlockBody{
		Transactions: pendingTxs,
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

	// Add block to DAG, but don't error out if it's already in the DAG
	blockHash := block.ComputeHash()
	if err := bp.dag.AddBlock(block); err != nil {
		if err.Error() == "block already exists" {
			log.Printf("Block %x already exists in DAG, continuing", blockHash)
		} else {
			log.Printf("Warning: Failed to add block to DAG: %v", err)
			return nil, fmt.Errorf("failed to add block to DAG: %v", err)
		}
	} else {
		// Log successful addition to DAG
		log.Printf("Added block to DAG - Hash: %x, Height: %d, Validator: %s, References: %d",
			blockHash, block.Header.Height, block.Header.Validator, len(block.Header.References))
	}

	// Update transaction states in mempool instead of removing them
	for _, tx := range pendingTxs {
		txHash := string(tx.GetHash())
		if existingTx, exists := bp.mempool.transactions[txHash]; exists {
			existingTx.State = types.TransactionStateIncluded
		}
	}

	log.Printf("Block created successfully with %d transactions and %d references", len(pendingTxs), len(references))
	log.Printf("Block details:")
	log.Printf("  Hash: %x", blockHash)
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
		Validator: bp.config.NodeID,
		Signature: signature,
		Timestamp: time.Now(),
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

// UpdateTransactionStates updates the state of transactions based on vote counts
func (bp *BlockProcessor) UpdateTransactionStates(block *types.Block, voteCount int, quorumSize int) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// If we have enough votes, mark transactions as committed
	if voteCount >= quorumSize {
		for _, tx := range block.Body.Transactions {
			tx.State = types.TransactionStateCommitted
		}
	}
}

// GetMempoolTransactions returns all transactions in the mempool with their states
func (bp *BlockProcessor) GetMempoolTransactions() []*types.Transaction {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	// Simple, fast approach - just get the transactions without complex parallel processing
	bp.mempool.mu.RLock()
	defer bp.mempool.mu.RUnlock()

	txs := make([]*types.Transaction, 0, len(bp.mempool.transactions))
	for _, tx := range bp.mempool.transactions {
		txs = append(txs, tx)
	}

	log.Printf("Retrieved %d transactions from mempool (simplified)", len(txs))
	return txs
}
