package core

import (
	// "context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/storage"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// Engine represents the core BlazeDAG engine
type Engine struct {
	// Configuration
	config *Config

	// State management
	stateManager *StateManager
	stateLock    sync.RWMutex

	// Components
	blockProcessor  *BlockProcessor
	consensusEngine *ConsensusEngine
	networkManager  *NetworkManager

	// Channels
	blockChan     chan *types.Block
	consensusChan chan *types.ConsensusMessage
	stopChan      chan struct{}
}

// Config holds the engine configuration
type Config struct {
	BlockInterval    time.Duration
	ConsensusTimeout time.Duration
	IsValidator      bool
	NodeID           types.Address
	
	// Block configuration for 40K transactions support
	MaxTransactionsPerBlock uint64
	MaxBlockSize            uint64  // in bytes
	TransactionTimeoutMs    uint64  // timeout for transaction processing
	MemPoolSize             uint64  // maximum transactions in mempool
	BatchSize               uint64  // batch size for parallel processing
	WorkerCount             int     // number of worker threads
	EnableOptimizations     bool    // enable performance optimizations
}

// NewDefaultConfig creates a new config with default values optimized for 40K transactions
func NewDefaultConfig() *Config {
	return &Config{
		BlockInterval:           2 * time.Second,
		ConsensusTimeout:        30 * time.Second,
		IsValidator:             false,
		NodeID:                  types.Address(""),
		MaxTransactionsPerBlock: 40000,
		MaxBlockSize:            100 * 1024 * 1024, // 100MB
		TransactionTimeoutMs:    5000,               // 5 seconds
		MemPoolSize:             200000,             // 200K transactions
		BatchSize:               1000,               // process 1000 transactions per batch
		WorkerCount:             16,                 // 16 worker threads
		EnableOptimizations:     true,
	}
}

// NewEngine creates a new BlazeDAG engine
func NewEngine(config *Config, storage *storage.Storage) *Engine {
	state := types.NewState()
	stateManager := NewStateManager(state, storage)

	// Use singleton DAG instance
	dag := GetDAG()

	blockProcessor := NewBlockProcessor(config, stateManager, dag)
	consensusEngine := NewConsensusEngine(config, stateManager)

	return &Engine{
		config:          config,
		stateManager:    stateManager,
		blockProcessor:  blockProcessor,
		consensusEngine: consensusEngine,
		blockChan:       make(chan *types.Block, 100),
		consensusChan:   make(chan *types.ConsensusMessage, 100),
		stopChan:        make(chan struct{}),
	}
}

// Stop gracefully shuts down the engine
func (e *Engine) Stop() {
	close(e.stopChan)
	e.networkManager.Stop()
}

// blockCreationLoop handles automatic block creation
func (e *Engine) blockCreationLoop() {
	ticker := time.NewTicker(e.config.BlockInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if e.config.IsValidator {
				fmt.Printf("22222")
				state := e.stateManager.GetState()
				
				state.CleanupOldBlocks()
				
				block, err := e.blockProcessor.CreateBlock(types.Round(state.CurrentRound), types.Wave(state.CurrentWave))
				if err != nil {
					log.Printf("Failed to create block: %v", err)
					continue
				}
				e.blockChan <- block
			}
		case <-e.stopChan:
			return
		}
	}
}

// State represents the current state of the system
type State struct {
	// Block management
	LatestBlock     *types.Block
	PendingBlocks   map[string]*types.Block
	FinalizedBlocks map[string]*types.Block
	
	// Consensus state
	CurrentRound   types.Round
	CurrentWave    types.Wave
	CurrentLeader  types.Address
	
	// Network state
	ActiveProposals map[string]*types.Proposal
	Votes          map[string][]*types.Vote
	ConnectedPeers map[types.Address]*types.Peer
	
	// Memory management
	maxPendingBlocks   int
	maxFinalizedBlocks int
	
	mu sync.RWMutex
}

// NewState creates a new state
func NewState() *State {
	return &State{
		PendingBlocks:   make(map[string]*types.Block),
		FinalizedBlocks: make(map[string]*types.Block),
		ActiveProposals: make(map[string]*types.Proposal),
		Votes:           make(map[string][]*types.Vote),
		ConnectedPeers:  make(map[types.Address]*types.Peer),
		maxPendingBlocks:   50,
		maxFinalizedBlocks: 100,
	}
}

// CleanupOldBlocks removes old blocks to prevent memory overflow
func (s *State) CleanupOldBlocks() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Clean up pending blocks
	if len(s.PendingBlocks) > 10 {
		// Convert to slice for easier management
		pendingHashes := make([]string, 0, len(s.PendingBlocks))
		for hash := range s.PendingBlocks {
			pendingHashes = append(pendingHashes, hash)
		}
		
		keepCount := 5
		
		blocksToRemove := len(pendingHashes) - keepCount
		if blocksToRemove > 0 {
			for i := 0; i < blocksToRemove; i++ {
				delete(s.PendingBlocks, pendingHashes[i])
			}
			log.Printf("🧹 State Cleanup: Removed %d old pending blocks, keeping %d in memory", 
				blocksToRemove, len(s.PendingBlocks))
		}
	}
	
	// Clean up finalized blocks
	if len(s.FinalizedBlocks) > 20 {
		// Convert to slice for easier management
		finalizedHashes := make([]string, 0, len(s.FinalizedBlocks))
		for hash := range s.FinalizedBlocks {
			finalizedHashes = append(finalizedHashes, hash)
		}
		
		keepCount := 10
		
		blocksToRemove := len(finalizedHashes) - keepCount
		if blocksToRemove > 0 {
			for i := 0; i < blocksToRemove; i++ {
				delete(s.FinalizedBlocks, finalizedHashes[i])
			}
			log.Printf("🧹 State Cleanup: Removed %d old finalized blocks, keeping %d in memory", 
				blocksToRemove, len(s.FinalizedBlocks))
		}
	}
}

// SetMemoryLimits allows configuring memory limits
func (s *State) SetMemoryLimits(maxPending, maxFinalized int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxPendingBlocks = maxPending
	s.maxFinalizedBlocks = maxFinalized
	log.Printf("State memory limits set: %d pending blocks, %d finalized blocks", maxPending, maxFinalized)
}
