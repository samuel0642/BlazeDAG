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

// Mempool stores pending transactions with parallel processing optimized for 40K+ transactions
type Mempool struct {
	transactions map[string]*types.Transaction
	mu           sync.RWMutex
	workers      int
	workChan     chan *types.Transaction
	doneChan     chan bool
	batchSize    int
	maxSize      uint64
	priorityHeap *TransactionPriorityHeap
}

// TransactionPriorityHeap implements a priority queue for transactions
type TransactionPriorityHeap []*types.Transaction

func (h TransactionPriorityHeap) Len() int { return len(h) }
func (h TransactionPriorityHeap) Less(i, j int) bool {
	// Higher gas price = higher priority
	return h[i].GasPrice > h[j].GasPrice
}
func (h TransactionPriorityHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *TransactionPriorityHeap) Push(x interface{}) {
	*h = append(*h, x.(*types.Transaction))
}
func (h *TransactionPriorityHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// NewMempoolWithConfig creates a new mempool with configuration for 40K+ transactions
func NewMempoolWithConfig(config *Config) *Mempool {
	workers := config.WorkerCount
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	
	batchSize := int(config.BatchSize)
	if batchSize <= 0 {
		batchSize = 1000
	}
	
	maxSize := config.MemPoolSize
	if maxSize == 0 {
		maxSize = 200000 // Default to 200K transactions
	}
	
	mp := &Mempool{
		transactions: make(map[string]*types.Transaction, maxSize),
		workers:      workers,
		workChan:     make(chan *types.Transaction, int(maxSize/10)), // 10% of max size as buffer
		doneChan:     make(chan bool),
		batchSize:    batchSize,
		maxSize:      maxSize,
		priorityHeap: &TransactionPriorityHeap{},
	}

	// Start worker pool
	for i := 0; i < workers; i++ {
		go mp.worker()
	}

	return mp
}

// NewMempool creates a new mempool with default configuration
func NewMempool() *Mempool {
	defaultConfig := NewDefaultConfig()
	return NewMempoolWithConfig(defaultConfig)
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

// GetTransactions returns transactions from the mempool with optimized batching for large volumes
func (mp *Mempool) GetTransactions() []*types.Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	// Pre-allocate slice with known capacity for better performance
	txs := make([]*types.Transaction, 0, len(mp.transactions))
	
	// Use batching for large transaction volumes
	if len(mp.transactions) > mp.batchSize {
		return mp.getBatchedTransactions()
	}

	// For smaller volumes, use simple approach
	for _, tx := range mp.transactions {
		txs = append(txs, tx)
	}

	log.Printf("Retrieved %d transactions from mempool", len(txs))
	return txs
}

// getBatchedTransactions processes large transaction volumes in batches
func (mp *Mempool) getBatchedTransactions() []*types.Transaction {
	txs := make([]*types.Transaction, 0, len(mp.transactions))
	batch := make([]*types.Transaction, 0, mp.batchSize)
	
	batchCount := 0
	for _, tx := range mp.transactions {
		batch = append(batch, tx)
		if len(batch) >= mp.batchSize {
			// Process batch
			txs = append(txs, batch...)
			batch = batch[:0] // Reset batch slice
			batchCount++
		}
	}
	
	// Process remaining transactions
	if len(batch) > 0 {
		txs = append(txs, batch...)
		batchCount++
	}
	
	log.Printf("Retrieved %d transactions from mempool using %d batches", len(txs), batchCount)
	return txs
}

// GetTopTransactions returns the top N transactions by priority for block creation
func (mp *Mempool) GetTopTransactions(maxCount uint64) []*types.Transaction {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if maxCount == 0 {
		return mp.GetTransactions()
	}

	// Convert to slice for sorting
	allTxs := make([]*types.Transaction, 0, len(mp.transactions))
	for _, tx := range mp.transactions {
		if tx.State == types.TransactionStatePending {
			allTxs = append(allTxs, tx)
		}
	}

	// Sort by gas price (descending)
	for i := 0; i < len(allTxs)-1; i++ {
		for j := i + 1; j < len(allTxs); j++ {
			if allTxs[i].GasPrice < allTxs[j].GasPrice {
				allTxs[i], allTxs[j] = allTxs[j], allTxs[i]
			}
		}
	}

	// Return top transactions up to maxCount
	if uint64(len(allTxs)) > maxCount {
		return allTxs[:maxCount]
	}
	
	return allTxs
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
		mempool:      NewMempoolWithConfig(config), // Use configured mempool
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

		log.Printf("Loaded %d valid blocks into DAG", len(validBlocks))
	}

	log.Printf("BlockProcessor initialized with config: MaxTxPerBlock=%d, MemPoolSize=%d, Workers=%d, BatchSize=%d",
		config.MaxTransactionsPerBlock, config.MemPoolSize, config.WorkerCount, config.BatchSize)

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
	
	// Filter only pending transactions
	pendingTxs := make([]*types.Transaction, 0)
	for _, tx := range allTxs {
		if tx.State == types.TransactionStatePending {
			pendingTxs = append(pendingTxs, tx)
		}
	}

	// Enforce transaction limit per block (40K transactions)
	maxTxPerBlock := bp.config.MaxTransactionsPerBlock
	if maxTxPerBlock == 0 {
		maxTxPerBlock = 40000 // Default to 40K
	}

	// Select top transactions by priority if we have more than the limit
	selectedTxs := pendingTxs
	if uint64(len(pendingTxs)) > maxTxPerBlock {
		log.Printf("Selecting top %d transactions from %d pending transactions", maxTxPerBlock, len(pendingTxs))
		selectedTxs = bp.mempool.GetTopTransactions(maxTxPerBlock)
	}

	log.Printf("Creating new block with %d transactions (max: %d) for round %d, wave %d", 
		len(selectedTxs), maxTxPerBlock, round, currentWave)

	// Set initial state for transactions
	for _, tx := range selectedTxs {
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
		StateRoot:  bp.calculateStateRoot(selectedTxs),
		Validator:  bp.config.NodeID,
	}

	// Create block body with selected transactions
	body := &types.BlockBody{
		Transactions: selectedTxs,
		Receipts:     make([]*types.Receipt, 0),
		Events:       make([]*types.Event, 0),
	}

	// Create block
	block := &types.Block{
		Header: header,
		Body:   body,
	}

	// Validate block size
	blockSize := bp.estimateBlockSize(block)
	if blockSize > bp.config.MaxBlockSize {
		log.Printf("Warning: Block size (%d bytes) exceeds maximum (%d bytes)", blockSize, bp.config.MaxBlockSize)
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
	for _, tx := range selectedTxs {
		txHash := string(tx.GetHash())
		if existingTx, exists := bp.mempool.transactions[txHash]; exists {
			existingTx.State = types.TransactionStateIncluded
		}
	}

	log.Printf("Block created successfully with %d transactions (size: %d bytes) and %d references", 
		len(selectedTxs), blockSize, len(references))
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

// estimateBlockSize estimates the size of a block in bytes
func (bp *BlockProcessor) estimateBlockSize(block *types.Block) uint64 {
	// Rough estimation: header (500 bytes) + transactions (200 bytes each) + references (100 bytes each)
	headerSize := uint64(500)
	txSize := uint64(len(block.Body.Transactions)) * 200
	refSize := uint64(len(block.Header.References)) * 100
	return headerSize + txSize + refSize
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
