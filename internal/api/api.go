package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// Server represents the API server
type Server struct {
	consensusEngine *consensus.ConsensusEngine
	port            int
	router          *http.ServeMux
	logger          *log.Logger
	server          *http.Server
	mu              sync.RWMutex
	running         bool
}

// NewServer creates a new API server
func NewServer(engine *consensus.ConsensusEngine, port int) *Server {
	router := http.NewServeMux()
	logger := log.New(log.Writer(), "[API] ", log.LstdFlags)

	return &Server{
		consensusEngine: engine,
		port:            port,
		router:          router,
		logger:          logger,
		running:         false,
	}
}

// Start starts the API server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// Register API routes
	s.registerRoutes()

	// Start HTTP server
	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	go func() {
		s.logger.Printf("API server started on port %d", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Printf("API server error: %v", err)
		}
	}()

	s.running = true
	return nil
}

// Stop stops the API server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.server != nil {
		if err := s.server.Close(); err != nil {
			return err
		}
	}

	s.running = false
	s.logger.Printf("API server stopped")
	return nil
}

// registerRoutes registers the API routes
func (s *Server) registerRoutes() {
	// Root route
	s.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": "BlazeDAG API Server",
		})
	})

	// Status endpoint
	s.router.HandleFunc("/status", s.handleStatus)

	// Block endpoints
	s.router.HandleFunc("/blocks", s.handleGetBlocks)
	s.router.HandleFunc("/blocks/", s.handleGetBlock)

	// DAG endpoints
	s.router.HandleFunc("/dag/stats", s.handleDAGStats)

	// Wave/consensus endpoints
	s.router.HandleFunc("/consensus/wave", s.handleWaveInfo)

	// Transaction endpoints
	s.router.HandleFunc("/transactions", s.handleGetTransactions)

	// Validator endpoints
	s.router.HandleFunc("/validators", s.handleGetValidators)

	// Debug endpoints
	s.router.HandleFunc("/debug/blockhash/", s.handleDebugBlockHash)
}

// handleStatus handles the status endpoint
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	currentWave := s.consensusEngine.GetCurrentWave()

	status := map[string]interface{}{
		"nodeID":      s.consensusEngine.GetNodeID(),
		"currentWave": currentWave,
		"validators":  s.consensusEngine.GetValidators(),
		"status":      "running",
	}

	json.NewEncoder(w).Encode(status)
}

// handleGetBlocks handles the blocks endpoint
func (s *Server) handleGetBlocks(w http.ResponseWriter, r *http.Request) {
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

	// Get blocks
	blocks := s.consensusEngine.GetRecentBlocks(count)

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

// handleGetBlock handles the block endpoint
func (s *Server) handleGetBlock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get hash parameter
	hashStr := r.URL.Path[len("/blocks/"):]
	if hashStr == "" {
		http.Error(w, "Block hash is required", http.StatusBadRequest)
		return
	}

	// Convert hex string to byte array
	hashBytes, err := hex.DecodeString(hashStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid block hash format: %v", err), http.StatusBadRequest)
		return
	}

	// Get block using byte array hash
	block, err := s.consensusEngine.GetBlock(hashBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Block not found: %v", err), http.StatusNotFound)
		return
	}

	// Create response
	txs := make([]map[string]interface{}, 0, len(block.Body.Transactions))
	for _, tx := range block.Body.Transactions {
		txMap := map[string]interface{}{
			"hash":      fmt.Sprintf("%x", tx.ComputeHash()),
			"from":      string(tx.From),
			"to":        string(tx.To),
			"value":     tx.Value,
			"nonce":     tx.Nonce,
			"gasLimit":  tx.GasLimit,
			"gasPrice":  tx.GasPrice,
			"timestamp": tx.Timestamp,
		}
		txs = append(txs, txMap)
	}

	response := map[string]interface{}{
		"hash":         fmt.Sprintf("%x", block.ComputeHash()),
		"validator":    string(block.Header.Validator),
		"timestamp":    block.Header.Timestamp,
		"round":        block.Header.Round,
		"wave":         block.Header.Wave,
		"height":       block.Header.Height,
		"parentHash":   fmt.Sprintf("%x", block.Header.ParentHash),
		"stateRoot":    fmt.Sprintf("%x", block.Header.StateRoot),
		"transactions": txs,
	}

	json.NewEncoder(w).Encode(response)
}

// handleDAGStats handles the DAG stats endpoint
func (s *Server) handleDAGStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get DAG
	dag := core.GetDAG()

	// Get blocks
	blocks := dag.GetRecentBlocks(100)

	// Count blocks by validator
	validatorBlocks := make(map[string]int)
	for _, block := range blocks {
		validator := string(block.Header.Validator)
		validatorBlocks[validator]++
	}

	response := map[string]interface{}{
		"totalBlocks":     len(blocks),
		"validatorBlocks": validatorBlocks,
	}

	json.NewEncoder(w).Encode(response)
}

// handleWaveInfo handles the wave info endpoint
func (s *Server) handleWaveInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	currentWave := s.consensusEngine.GetCurrentWave()

	response := map[string]interface{}{
		"currentWave": currentWave,
		"waveStatus":  s.consensusEngine.GetWaveStatus(),
	}

	json.NewEncoder(w).Encode(response)
}

// handleGetTransactions handles the transactions endpoint
func (s *Server) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
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

	// Get blocks
	blocks := s.consensusEngine.GetRecentBlocks(count)

	// Collect transactions
	txs := make([]map[string]interface{}, 0)
	for _, block := range blocks {
		blockHash := block.ComputeHash()
		for _, tx := range block.Body.Transactions {
			txHash := tx.ComputeHash()
			txMap := map[string]interface{}{
				"hash":       fmt.Sprintf("%x", txHash),
				"from":       string(tx.From),
				"to":         string(tx.To),
				"value":      tx.Value,
				"nonce":      tx.Nonce,
				"gasLimit":   tx.GasLimit,
				"gasPrice":   tx.GasPrice,
				"timestamp":  tx.Timestamp,
				"blockHash":  fmt.Sprintf("%x", blockHash),
				"blockWave":  block.Header.Wave,
				"blockRound": block.Header.Round,
			}
			txs = append(txs, txMap)
		}
	}

	json.NewEncoder(w).Encode(txs)
}

// handleGetValidators handles the validators endpoint
func (s *Server) handleGetValidators(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	validators := s.consensusEngine.GetValidators()

	response := map[string]interface{}{
		"validators": validators,
		"count":      len(validators),
	}

	json.NewEncoder(w).Encode(response)
}

// handleDebugBlockHash handles the debug block hash endpoint
func (s *Server) handleDebugBlockHash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get hash parameter
	hashStr := r.URL.Path[len("/debug/blockhash/"):]
	if hashStr == "" {
		http.Error(w, "Block hash is required", http.StatusBadRequest)
		return
	}

	// Convert hex string to byte array
	hashBytes, err := hex.DecodeString(hashStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid block hash format: %v", err), http.StatusBadRequest)
		return
	}

	// Get recent blocks
	blocks := s.consensusEngine.GetRecentBlocks(100)

	// Check each block to see if it matches
	var matchedBlock *types.Block
	for _, block := range blocks {
		blockHash := block.ComputeHash()
		if bytes.Equal(blockHash, hashBytes) {
			matchedBlock = block
			break
		}
	}

	if matchedBlock == nil {
		http.Error(w, "Block not found in recent blocks", http.StatusNotFound)
		return
	}

	// Create detailed debug response
	response := map[string]interface{}{
		"hash": map[string]interface{}{
			"hex":   fmt.Sprintf("%x", hashBytes),
			"len":   len(hashBytes),
			"bytes": hashBytes,
			"raw":   string(hashBytes),
		},
		"block": map[string]interface{}{
			"validator": string(matchedBlock.Header.Validator),
			"timestamp": matchedBlock.Header.Timestamp,
			"wave":      matchedBlock.Header.Wave,
			"round":     matchedBlock.Header.Round,
			"height":    matchedBlock.Header.Height,
		},
		"lookupTest": map[string]interface{}{
			"hashType":     fmt.Sprintf("%T", hashBytes),
			"validHashLen": len(hashBytes) > 0,
		},
	}

	json.NewEncoder(w).Encode(response)
}
