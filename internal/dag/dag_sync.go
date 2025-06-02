package dag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// DAGSync handles DAG synchronization between validators
type DAGSync struct {
	validatorID   types.Address
	dag           *DAG
	currentRound  types.Round
	roundDuration time.Duration
	
	// Network configuration
	listenAddr    string
	peers         []string
	
	// HTTP API configuration
	httpAddr      string
	httpServer    *http.Server
	
	// Synchronization
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	
	// Round management
	roundTicker   *time.Ticker
	roundBlocks   map[types.Round][]*types.Block // blocks per round
	
	// Network layer
	listener      net.Listener
	connections   map[string]net.Conn
	connMu        sync.RWMutex
}

// DAGMessage represents a message for DAG synchronization
type DAGMessage struct {
	Type      string          `json:"type"`
	Block     *types.Block    `json:"block,omitempty"`
	Round     types.Round     `json:"round,omitempty"`
	Validator types.Address   `json:"validator"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewDAGSync creates a new DAG sync instance
func NewDAGSync(validatorID types.Address, listenAddr string, peers []string, roundDuration time.Duration) *DAGSync {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Use fixed HTTP API port 8080 for frontend compatibility
	// Extract host from listenAddr or default to 0.0.0.0
	host := "0.0.0.0"
	if h, _, err := net.SplitHostPort(listenAddr); err == nil && h != "" {
		host = h
	}
	httpAddr := fmt.Sprintf("%s:8080", host) // Fixed port for Explorer frontend
	
	return &DAGSync{
		validatorID:   validatorID,
		dag:           NewDAG(),
		currentRound:  0,
		roundDuration: roundDuration,
		listenAddr:    listenAddr,
		peers:         peers,
		httpAddr:      httpAddr,
		ctx:           ctx,
		cancel:        cancel,
		roundBlocks:   make(map[types.Round][]*types.Block),
		connections:   make(map[string]net.Conn),
	}
}

// Start starts the DAG sync process
func (ds *DAGSync) Start() error {
	log.Printf("DAG Sync [%s]: Starting DAG synchronization", ds.validatorID)
	
	// Start network listener
	if err := ds.startListener(); err != nil {
		return fmt.Errorf("failed to start listener: %v", err)
	}
	
	// Start HTTP API server
	if err := ds.startHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %v", err)
	}
	
	// Connect to peers
	go ds.connectToPeers()
	
	// Wait for all peers to connect before starting rounds
	if len(ds.peers) > 0 {
		log.Printf("DAG Sync [%s]: Waiting for all %d peers to connect...", ds.validatorID, len(ds.peers))
		ds.waitForAllPeers()
		log.Printf("DAG Sync [%s]: ✅ All peers connected! Starting synchronized rounds...", ds.validatorID)
	} else {
		log.Printf("DAG Sync [%s]: No peers configured, starting immediately", ds.validatorID)
	}
	
	// Start round ticker AFTER all peers are connected
	ds.roundTicker = time.NewTicker(ds.roundDuration)
	go ds.runRounds()
	
	log.Printf("DAG Sync [%s]: Started at %s, connected to %d peers", 
		ds.validatorID, ds.listenAddr, len(ds.peers))
	log.Printf("DAG Sync [%s]: HTTP API available at http://%s", ds.validatorID, ds.httpAddr)
	
	return nil
}

// Stop stops the DAG sync
func (ds *DAGSync) Stop() {
	log.Printf("DAG Sync [%s]: Stopping", ds.validatorID)
	ds.cancel()
	
	if ds.roundTicker != nil {
		ds.roundTicker.Stop()
	}
	
	if ds.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ds.httpServer.Shutdown(ctx)
	}
	
	if ds.listener != nil {
		ds.listener.Close()
	}
	
	ds.connMu.Lock()
	for _, conn := range ds.connections {
		conn.Close()
	}
	ds.connMu.Unlock()
}

// startListener starts the network listener
func (ds *DAGSync) startListener() error {
	listener, err := net.Listen("tcp", ds.listenAddr)
	if err != nil {
		return err
	}
	
	ds.listener = listener
	
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ds.ctx.Done():
					return
				default:
					log.Printf("DAG Sync [%s]: Accept error: %v", ds.validatorID, err)
					continue
				}
			}
			
			go ds.handleConnection(conn)
		}
	}()
	
	return nil
}

// connectToPeers connects to peer validators
func (ds *DAGSync) connectToPeers() {
	for _, peer := range ds.peers {
		go func(peerAddr string) {
			for {
				select {
				case <-ds.ctx.Done():
					return
				default:
					if err := ds.connectToPeer(peerAddr); err != nil {
						log.Printf("DAG Sync [%s]: Failed to connect to %s: %v", ds.validatorID, peerAddr, err)
						time.Sleep(2 * time.Second) // Retry after 2 seconds
						continue
					}
					return // Successfully connected
				}
			}
		}(peer)
	}
}

// connectToPeer connects to a specific peer
func (ds *DAGSync) connectToPeer(peerAddr string) error {
	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		return err
	}
	
	ds.connMu.Lock()
	ds.connections[peerAddr] = conn
	ds.connMu.Unlock()
	
	log.Printf("DAG Sync [%s]: Connected to peer %s", ds.validatorID, peerAddr)
	
	go ds.handleConnection(conn)
	return nil
}

// handleConnection handles a network connection
func (ds *DAGSync) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	decoder := json.NewDecoder(conn)
	for {
		select {
		case <-ds.ctx.Done():
			return
		default:
			var msg DAGMessage
			if err := decoder.Decode(&msg); err != nil {
				return // Connection closed or error
			}
			
			ds.handleDAGMessage(&msg)
		}
	}
}

// runRounds runs the DAG round process
func (ds *DAGSync) runRounds() {
	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ds.roundTicker.C:
			ds.advanceRound()
		}
	}
}

// advanceRound advances to the next DAG round
func (ds *DAGSync) advanceRound() {
	ds.mu.Lock()
	ds.currentRound++
	currentRound := ds.currentRound
	ds.mu.Unlock()
	
	log.Printf("DAG Sync [%s]: Advanced to round %d", ds.validatorID, currentRound)
	
	// Create block for this round
	go ds.createRoundBlock(currentRound)
}

// createRoundBlock creates a block for the current round
func (ds *DAGSync) createRoundBlock(round types.Round) {
	// Get references to blocks from previous round
	references := ds.getPreviousRoundReferences(round - 1)
	
	// Create transactions (simulate) - Generate 40K transactions per block
	transactions := ds.generateTransactions(40000) // 40K transactions per block
	
	// Create the block
	block := &types.Block{
		Header: &types.BlockHeader{
			Version:    1,
			Timestamp:  time.Now(),
			Round:      round,
			Wave:       0, // DAG sync doesn't care about waves
			Height:     ds.dag.GetHeight() + 1,
			References: references,
			Validator:  ds.validatorID,
		},
		Body: &types.BlockBody{
			Transactions: transactions,
			Receipts:     make([]*types.Receipt, 0),
			Events:       make([]*types.Event, 0),
		},
	}
	
	// Calculate block hash before adding to DAG
	blockHash := block.ComputeHash()
	
	// Add to local DAG
	if err := ds.dag.AddBlock(block); err != nil {
		log.Printf("DAG Sync [%s]: Failed to add block to DAG: %v", ds.validatorID, err)
		return
	}
	
	// Track block for this round
	ds.mu.Lock()
	if ds.roundBlocks[round] == nil {
		ds.roundBlocks[round] = make([]*types.Block, 0)
	}
	ds.roundBlocks[round] = append(ds.roundBlocks[round], block)
	ds.mu.Unlock()
	
	// Detailed logging with block information and performance metrics
	blockSize := ds.estimateBlockSize(block)
	log.Printf("🔥 DAG Sync [%s]: CREATED BLOCK Details:", ds.validatorID)
	log.Printf("   📦 Hash: %x", blockHash)
	log.Printf("   🔄 Round: %d", block.Header.Round)
	log.Printf("   📏 Height: %d", block.Header.Height)
	log.Printf("   👤 Validator: %s", block.Header.Validator)
	log.Printf("   🔗 References: %d", len(references))
	log.Printf("   💼 Transactions: %d", len(transactions))
	log.Printf("   📊 Block Size: %.2f MB (%d bytes)", float64(blockSize)/(1024*1024), blockSize)
	log.Printf("   ⏰ Timestamp: %s", block.Header.Timestamp.Format("15:04:05"))
	
	log.Printf("DAG Sync [%s]: Created block for round %d with %d references and %d transactions (%.2f MB)", 
		ds.validatorID, round, len(references), len(transactions), float64(blockSize)/(1024*1024))
	
	// Broadcast to other validators
	ds.broadcastBlock(block)
}

// estimateBlockSize estimates the size of a block in bytes
func (ds *DAGSync) estimateBlockSize(block *types.Block) uint64 {
	// Rough estimation: header (500 bytes) + transactions (250 bytes each) + references (100 bytes each)
	headerSize := uint64(500)
	txSize := uint64(len(block.Body.Transactions)) * 250
	refSize := uint64(len(block.Header.References)) * 100
	return headerSize + txSize + refSize
}

// getPreviousRoundReferences gets references to blocks from previous round
func (ds *DAGSync) getPreviousRoundReferences(prevRound types.Round) []*types.Reference {
	if prevRound <= 0 {
		return []*types.Reference{} // Genesis round
	}
	
	// Wait a bit to ensure we have received blocks from other validators for the previous round
	time.Sleep(100 * time.Millisecond)
	
	ds.mu.RLock()
	prevBlocks := ds.roundBlocks[prevRound]
	ds.mu.RUnlock()
	
	// Also get ALL blocks from the DAG that belong to the previous round
	// This ensures we reference blocks from other validators too
	allDAGBlocks := ds.dag.GetRecentBlocks(100) // Get recent blocks
	
	// Collect all blocks from the previous round (from all validators)
	allPrevRoundBlocks := make([]*types.Block, 0)
	
	// Add blocks we tracked locally for this round
	allPrevRoundBlocks = append(allPrevRoundBlocks, prevBlocks...)
	
	// Add any blocks from DAG that belong to previous round but we might have missed
	for _, block := range allDAGBlocks {
		if block.Header.Round == prevRound {
			// Check if we already have this block in our list
			found := false
			for _, existing := range allPrevRoundBlocks {
				if string(existing.ComputeHash()) == string(block.ComputeHash()) {
					found = true
					break
				}
			}
			if !found {
				allPrevRoundBlocks = append(allPrevRoundBlocks, block)
			}
		}
	}
	
	log.Printf("DAG Sync [%s]: Found %d blocks from round %d (from all validators)", 
		ds.validatorID, len(allPrevRoundBlocks), prevRound)
	
	references := make([]*types.Reference, 0)
	
	// Reference ALL blocks from previous round (from all validators)
	for _, block := range allPrevRoundBlocks {
		ref := &types.Reference{
			BlockHash: block.ComputeHash(),
			Round:     block.Header.Round,
			Wave:      block.Header.Wave,
			Type:      types.ReferenceTypeStandard,
		}
		references = append(references, ref)
	}
	
	// Log which validators' blocks we're referencing
	validatorCounts := make(map[types.Address]int)
	for _, block := range allPrevRoundBlocks {
		validatorCounts[block.Header.Validator]++
	}
	
	log.Printf("DAG Sync [%s]: Referencing blocks from validators: %v", ds.validatorID, validatorCounts)
	
	return references
}

// generateTransactions generates sample transactions optimized for 40K+ transactions
func (ds *DAGSync) generateTransactions(count int) []*types.Transaction {
	// Pre-allocate slice with known capacity for better performance
	transactions := make([]*types.Transaction, count)
	
	// Use batch processing for large transaction volumes
	batchSize := 1000
	baseTime := time.Now().UnixNano()
	
	log.Printf("DAG Sync [%s]: Generating %d transactions in batches of %d...", ds.validatorID, count, batchSize)
	
	// Process transactions in batches for better performance
	for batchStart := 0; batchStart < count; batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > count {
			batchEnd = count
		}
		
		// Generate batch of transactions
		for i := batchStart; i < batchEnd; i++ {
			// Create more realistic transaction data
			recipientID := i % 1000  // Cycle through 1000 different recipients
			value := 100 + (i % 10000) // Values from 100 to 10099
			gasPrice := 1000000000 + uint64(i%1000000) // Varying gas prices for priority
			
			tx := &types.Transaction{
				Nonce:     types.Nonce(baseTime + int64(i)),
				From:      ds.validatorID,
				To:        types.Address(fmt.Sprintf("recipient_%d", recipientID)),
				Value:     types.Value(value),
				GasLimit:  21000,
				GasPrice:  gasPrice,
				Data:      []byte(fmt.Sprintf("tx_data_%d", i)),
				Timestamp: time.Now(),
				State:     types.TransactionStatePending,
			}
			transactions[i] = tx
		}
		
		// Log progress for large batches
		if count >= 10000 && (batchEnd%10000 == 0 || batchEnd == count) {
			log.Printf("DAG Sync [%s]: Generated %d/%d transactions...", ds.validatorID, batchEnd, count)
		}
	}
	
	log.Printf("DAG Sync [%s]: Successfully generated %d transactions", ds.validatorID, count)
	return transactions
}

// broadcastBlock broadcasts a block to all connected peers
func (ds *DAGSync) broadcastBlock(block *types.Block) {
	msg := &DAGMessage{
		Type:      "block",
		Block:     block,
		Round:     block.Header.Round,
		Validator: ds.validatorID,
		Timestamp: time.Now(),
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("DAG Sync [%s]: Failed to marshal block: %v", ds.validatorID, err)
		return
	}
	
	ds.connMu.RLock()
	defer ds.connMu.RUnlock()
	
	broadcastCount := 0
	for peerAddr, conn := range ds.connections {
		if _, err := conn.Write(append(data, '\n')); err != nil {
			log.Printf("DAG Sync [%s]: Failed to send to %s: %v", ds.validatorID, peerAddr, err)
			continue
		}
		broadcastCount++
	}
	
	log.Printf("DAG Sync [%s]: Broadcasted block (round %d) to %d peers", 
		ds.validatorID, block.Header.Round, broadcastCount)
}

// handleDAGMessage handles received DAG messages
func (ds *DAGSync) handleDAGMessage(msg *DAGMessage) {
	switch msg.Type {
	case "block":
		ds.handleReceivedBlock(msg.Block, msg.Validator)
	default:
		log.Printf("DAG Sync [%s]: Unknown message type: %s", ds.validatorID, msg.Type)
	}
}

// handleReceivedBlock handles a received block from another validator
func (ds *DAGSync) handleReceivedBlock(block *types.Block, fromValidator types.Address) {
	if block == nil {
		return
	}
	
	// Check if we already have this block
	if _, err := ds.dag.GetBlock(block.ComputeHash()); err == nil {
		return // Already have this block
	}
	
	// Add to local DAG
	if err := ds.dag.AddBlock(block); err != nil {
		log.Printf("DAG Sync [%s]: Failed to add received block: %v", ds.validatorID, err)
		return
	}
	
	// Track block for this round
	ds.mu.Lock()
	round := block.Header.Round
	if ds.roundBlocks[round] == nil {
		ds.roundBlocks[round] = make([]*types.Block, 0)
	}
	ds.roundBlocks[round] = append(ds.roundBlocks[round], block)
	ds.mu.Unlock()
	
	log.Printf("DAG Sync [%s]: Received and added block from %s (round %d)", 
		ds.validatorID, fromValidator, block.Header.Round)
}

// GetDAGStatus returns the current DAG status
func (ds *DAGSync) GetDAGStatus() map[string]interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	// Count blocks per round
	roundCounts := make(map[string]int)
	for round, blocks := range ds.roundBlocks {
		roundCounts[fmt.Sprintf("round_%d", round)] = len(blocks)
	}
	
	ds.connMu.RLock()
	connectedPeers := len(ds.connections)
	ds.connMu.RUnlock()
	
	return map[string]interface{}{
		"validator_id":     ds.validatorID,
		"current_round":    ds.currentRound,
		"total_blocks":     ds.dag.GetBlockCount(),
		"dag_height":       ds.dag.GetHeight(),
		"connected_peers":  connectedPeers,
		"blocks_per_round": roundCounts,
		"listen_addr":      ds.listenAddr,
	}
}

// GetCurrentRound returns the current round
func (ds *DAGSync) GetCurrentRound() types.Round {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.currentRound
}

// GetBlocksForRound returns all blocks for a specific round
func (ds *DAGSync) GetBlocksForRound(round types.Round) []*types.Block {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	blocks := make([]*types.Block, 0)
	if roundBlocks, exists := ds.roundBlocks[round]; exists {
		blocks = append(blocks, roundBlocks...)
	}
	return blocks
}

// GetRecentBlocks returns recent blocks from the DAG
func (ds *DAGSync) GetRecentBlocks(count int) []*types.Block {
	return ds.dag.GetRecentBlocks(count)
}

// GetDAG returns the underlying DAG for read access
func (ds *DAGSync) GetDAG() *DAG {
	return ds.dag
}

// GetValidatorID returns the validator ID
func (ds *DAGSync) GetValidatorID() types.Address {
	return ds.validatorID
}

// startHTTPServer starts the HTTP API server
func (ds *DAGSync) startHTTPServer() error {
	mux := http.NewServeMux()
	
	// Enable CORS for all endpoints
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			h.ServeHTTP(w, r)
		})
	}
	
	// Register API routes
	mux.HandleFunc("/", ds.handleRoot)
	mux.HandleFunc("/blocks", ds.handleGetBlocks)
	mux.HandleFunc("/status", ds.handleGetStatus)
	mux.HandleFunc("/dag/stats", ds.handleGetDAGStats)
	mux.HandleFunc("/transactions", ds.handleGetTransactions)
	mux.HandleFunc("/validators", ds.handleGetValidators)
	mux.HandleFunc("/consensus/wave", ds.handleGetConsensusWave)
	
	ds.httpServer = &http.Server{
		Addr:    ds.httpAddr,
		Handler: corsHandler(mux),
	}
	
	go func() {
		log.Printf("DAG Sync [%s]: Starting HTTP API server on %s", ds.validatorID, ds.httpAddr)
		if err := ds.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("DAG Sync [%s]: HTTP server error: %v", ds.validatorID, err)
		}
	}()
	
	return nil
}

// handleRoot handles the root endpoint
func (ds *DAGSync) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"message":    "BlazeDAG DAG Sync API",
		"validator":  string(ds.validatorID),
		"listen":     ds.listenAddr,
		"http":       ds.httpAddr,
	})
}

// handleGetBlocks handles the blocks endpoint
func (ds *DAGSync) handleGetBlocks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Get count parameter (default to 10)
	countStr := r.URL.Query().Get("count")
	count := 10
	if countStr != "" {
		var err error
		count, err = strconv.Atoi(countStr)
		if err != nil || count <= 0 {
			http.Error(w, "Invalid count parameter", http.StatusBadRequest)
			return
		}
	}
	
	// Get recent blocks from DAG
	blocks := ds.GetRecentBlocks(count)
	
	// Create response
	response := make([]map[string]interface{}, 0, len(blocks))
	for _, block := range blocks {
		blockHash := block.ComputeHash()
		blockMap := map[string]interface{}{
			"hash":      fmt.Sprintf("%x", blockHash),
			"validator": string(block.Header.Validator),
			"timestamp": block.Header.Timestamp,
			"round":     block.Header.Round,
			"wave":      block.Header.Wave,
			"height":    block.Header.Height,
			"txCount":   len(block.Body.Transactions),
		}
		response = append(response, blockMap)
	}
	
	json.NewEncoder(w).Encode(response)
}

// handleGetStatus handles the status endpoint
func (ds *DAGSync) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := ds.GetDAGStatus()
	json.NewEncoder(w).Encode(status)
}

// handleGetDAGStats handles the DAG stats endpoint
func (ds *DAGSync) handleGetDAGStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	ds.mu.RLock()
	roundCounts := make(map[string]int)
	for round, blocks := range ds.roundBlocks {
		roundCounts[fmt.Sprintf("round_%d", round)] = len(blocks)
	}
	ds.mu.RUnlock()
	
	ds.connMu.RLock()
	connectedPeers := len(ds.connections)
	ds.connMu.RUnlock()
	
	stats := map[string]interface{}{
		"total_blocks":     ds.dag.GetBlockCount(),
		"dag_height":       ds.dag.GetHeight(),
		"current_round":    ds.currentRound,
		"connected_peers":  connectedPeers,
		"blocks_per_round": roundCounts,
		"validator_id":     string(ds.validatorID),
		"listen_addr":      ds.listenAddr,
		"peers":           ds.peers,
	}
	
	json.NewEncoder(w).Encode(stats)
}

// handleGetTransactions handles the transactions endpoint
func (ds *DAGSync) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Get count parameter (default to 100)
	countStr := r.URL.Query().Get("count")
	count := 100
	if countStr != "" {
		var err error
		count, err = strconv.Atoi(countStr)
		if err != nil || count <= 0 {
			http.Error(w, "Invalid count parameter", http.StatusBadRequest)
			return
		}
	}
	
	// Get recent blocks and extract transactions
	blocks := ds.GetRecentBlocks(count / 5) // Assuming ~5 tx per block
	transactions := make([]map[string]interface{}, 0)
	
	for _, block := range blocks {
		for _, tx := range block.Body.Transactions {
			txMap := map[string]interface{}{
				"hash":      fmt.Sprintf("%x", tx.GetHash()),
				"from":      string(tx.From),
				"to":        string(tx.To),
				"value":     tx.Value,
				"nonce":     tx.Nonce,
				"gasLimit":  tx.GasLimit,
				"gasPrice":  tx.GasPrice,
				"timestamp": tx.Timestamp,
				"state":     tx.State,
				"blockHash": fmt.Sprintf("%x", block.ComputeHash()),
				"blockHeight": block.Header.Height,
			}
			transactions = append(transactions, txMap)
			
			if len(transactions) >= count {
				break
			}
		}
		if len(transactions) >= count {
			break
		}
	}
	
	json.NewEncoder(w).Encode(transactions)
}

// handleGetValidators handles the validators endpoint
func (ds *DAGSync) handleGetValidators(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Get all unique validators from recent blocks
	validatorMap := make(map[string]map[string]interface{})
	
	// Add ourselves
	validatorMap[string(ds.validatorID)] = map[string]interface{}{
		"id":            string(ds.validatorID),
		"status":        "active",
		"blocks_created": 0,
		"last_seen":     time.Now(),
		"is_connected":  true,
	}
	
	// Count blocks created by each validator
	blocks := ds.GetRecentBlocks(1000)
	for _, block := range blocks {
		validatorID := string(block.Header.Validator)
		if validator, exists := validatorMap[validatorID]; exists {
			validator["blocks_created"] = validator["blocks_created"].(int) + 1
			validator["last_seen"] = block.Header.Timestamp
		} else {
			validatorMap[validatorID] = map[string]interface{}{
				"id":             validatorID,
				"status":         "active",
				"blocks_created": 1,
				"last_seen":      block.Header.Timestamp,
				"is_connected":   false, // We don't know about other validators' connection status
			}
		}
	}
	
	// Check which validators are connected
	ds.connMu.RLock()
	for _ = range ds.connections {
		// Try to match peer addresses to validator IDs (simplified)
		for validatorID, validator := range validatorMap {
			if validatorID != string(ds.validatorID) {
				validator["is_connected"] = true
			}
		}
	}
	ds.connMu.RUnlock()
	
	// Convert map to slice
	validators := make([]map[string]interface{}, 0, len(validatorMap))
	for _, validator := range validatorMap {
		validators = append(validators, validator)
	}
	
	json.NewEncoder(w).Encode(validators)
}

// handleGetConsensusWave handles the consensus wave endpoint
func (ds *DAGSync) handleGetConsensusWave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// DAG sync doesn't handle waves directly, but we can provide basic info
	waveInfo := map[string]interface{}{
		"current_wave":    0, // DAG sync doesn't track waves
		"wave_status":     "dag_sync_only",
		"message":         "Wave consensus is handled by separate wave consensus service",
		"dag_round":       ds.currentRound,
		"suggestion":      "Connect to wave consensus service for wave information",
		"blocks_in_dag":   ds.dag.GetBlockCount(),
		"current_height":  ds.dag.GetHeight(),
	}
	
	json.NewEncoder(w).Encode(waveInfo)
}

// waitForAllPeers waits for all peers to connect before starting rounds
func (ds *DAGSync) waitForAllPeers() {
	for {
		ds.connMu.RLock()
		connectedPeers := len(ds.connections)
		ds.connMu.RUnlock()
		
		if connectedPeers >= len(ds.peers) {
			return
		}
		
		log.Printf("DAG Sync [%s]: Connected to %d/%d peers, waiting...", 
			ds.validatorID, connectedPeers, len(ds.peers))
		time.Sleep(1 * time.Second)
	}
} 