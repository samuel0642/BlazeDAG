package core

import (
	// "context"
	"sync"
	"time"	
	"log"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// Engine represents the core BlazeDAG engine
type Engine struct {
	// Configuration
	config *Config

	// State
	state     *State
	stateLock sync.RWMutex

	// Components
	blockCreator    *BlockCreator
	consensusEngine *ConsensusEngine
	networkManager  *NetworkManager

	// Channels
	blockChan    chan *types.Block
	consensusChan chan *types.ConsensusMessage
	stopChan     chan struct{}
}

// Config holds the engine configuration
type Config struct {
	BlockInterval    time.Duration
	ConsensusTimeout time.Duration
	IsValidator      bool
	NodeID          types.Address
}

// NewEngine creates a new BlazeDAG engine
func NewEngine(config *Config) *Engine {
	return &Engine{
		config: config,
		state:  NewState(),
		blockChan: make(chan *types.Block, 100),
		consensusChan: make(chan *types.ConsensusMessage, 100),
		stopChan: make(chan struct{}),
	}
}

// Start initializes and starts the engine
func (e *Engine) Start() error {
	// Initialize components
	e.blockCreator = NewBlockCreator(e.config, e.state)
	e.consensusEngine = NewConsensusEngine(e.config, e.state)
	e.networkManager = NewNetworkManager(e.config, e.state)

	// Start components
	if err := e.networkManager.Start(); err != nil {
		return err
	}

	// Start block creation loop
	go e.blockCreationLoop()

	// Start consensus loop
	go e.consensusLoop()

	return nil
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
				block, err := e.blockCreator.CreateBlock()
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

// consensusLoop handles the consensus process
func (e *Engine) consensusLoop() {
	for {
		select {
		case block := <-e.blockChan:
			if e.config.IsValidator {
				// Start consensus process for the block
				go e.consensusEngine.ProcessBlock(block)
			}
		case msg := <-e.consensusChan:
			// Handle consensus messages
			e.consensusEngine.HandleMessage(msg)
		case <-e.stopChan:
			return
		}
	}
}

// State represents the current state of the engine
type State struct {
	// Block state
	LatestBlock     *types.Block
	PendingBlocks   map[string]*types.Block
	FinalizedBlocks map[string]*types.Block

	// Consensus state
	CurrentWave     uint64
	CurrentRound    uint64
	ActiveProposals map[string]*types.Proposal
	Votes           map[string][]*types.Vote

	// Network state
	ConnectedPeers map[types.Address]*types.Peer
}

// NewState creates a new engine state
func NewState() *State {
	return &State{
		PendingBlocks:   make(map[string]*types.Block),
		FinalizedBlocks: make(map[string]*types.Block),
		ActiveProposals: make(map[string]*types.Proposal),
		Votes:           make(map[string][]*types.Vote),
		ConnectedPeers:  make(map[types.Address]*types.Peer),
	}
} 