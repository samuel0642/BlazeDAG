package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/config"
	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/storage"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// CLI represents the command line interface
type CLI struct {
	config          *config.Config
	consensusEngine *consensus.ConsensusEngine
	waveController  *consensus.WaveController
	blockProcessor  *core.BlockProcessor
	dag             *core.DAG
	storage         *storage.Storage
	scanner         *bufio.Scanner
	stopChan        chan struct{}
	currentRound    int
}

// NewCLI creates a new CLI instance
func NewCLI(cfg *config.Config) *CLI {
	return &CLI{
		config:  cfg,
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// Start starts the CLI
func (c *CLI) Start() error {
	// Initialize components
	if err := c.initialize(); err != nil {
		return fmt.Errorf("failed to initialize: %v", err)
	}

	// Start consensus engine
	if err := c.consensusEngine.Start(); err != nil {
		return fmt.Errorf("failed to start consensus engine: %v", err)
	}

	// Start wave controller
	c.waveController.Start()

	// Start block creation and round forwarding
	go c.runChain()

	// Print welcome message
	fmt.Println("BlazeDAG CLI - Type 'help' for available commands")

	// Main loop
	for {
		fmt.Print("> ")
		if !c.scanner.Scan() {
			break
		}

		line := c.scanner.Text()
		if err := c.handleCommand(line); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	return nil
}

// runChain runs the chain with round and wave forwarding
func (c *CLI) runChain() {
	round := 1
	height := types.BlockNumber(0)
	lastWave := types.Wave(1) // Start from wave 1
	blockCreatedInWave := false

	for {
		select {
		case <-c.stopChan:
			return
		default:
			// Get current wave from consensus engine
			currentWave := c.consensusEngine.GetCurrentWave()
			
			// Only create block if we're in a new wave and haven't created a block yet
			if currentWave != lastWave {
				blockCreatedInWave = false
				lastWave = currentWave
			}

			if !blockCreatedInWave {
				// Create a block with current round number
				block, err := c.blockProcessor.CreateBlock(types.Round(round))
				if err != nil {
					log.Printf("Error creating block: %v", err)
					continue
				}

				// Set block properties
				block.Header.Wave = currentWave
				block.Header.Height = height

				// Show leader selection when wave changes
				log.Printf("\n=== Wave %d Leader Selection ===", currentWave)
				log.Printf("Selected Leader: %s", c.config.NodeID)

				// Add block to DAG
				if err := c.dag.AddBlock(block); err != nil {
					log.Printf("Error adding block to DAG: %v", err)
					continue
				}

				// Save block to storage
				if err := c.storage.SaveBlock(block); err != nil {
					log.Printf("Error saving block: %v", err)
				}

				// Process the block through consensus
				if err := c.consensusEngine.HandleBlock(block); err != nil {
					log.Printf("Error processing block: %v", err)
					continue
				}

				// Update current round and mark block as created
				c.currentRound = round
				blockCreatedInWave = true

				// Increment round and height
				round++
				height++

				// Save state
				engineState := &core.State{
					CurrentWave: uint64(currentWave),
					LatestBlock: block,
					PendingBlocks: make(map[string]*types.Block),
					FinalizedBlocks: make(map[string]*types.Block),
					ActiveProposals: make(map[string]*types.Proposal),
					Votes: make(map[string][]*types.Vote),
					ConnectedPeers: make(map[types.Address]*types.Peer),
				}
				if err := c.storage.SaveState(engineState); err != nil {
					log.Printf("Error saving state: %v", err)
				}
			}

			// Sleep for the configured round duration
			time.Sleep(c.config.Consensus.RoundDuration)
		}
	}
}

// Stop stops the CLI
func (c *CLI) Stop() {
	close(c.stopChan)
	// No need to stop consensus engine as it doesn't have a Stop method
}

// initialize initializes the CLI components
func (c *CLI) initialize() error {
	// Initialize storage
	baseDir := filepath.Join("data", string(c.config.NodeID))
	storage, err := storage.NewStorage(baseDir)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %v", err)
	}
	c.storage = storage

	// Load state from storage
	engineState, err := storage.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %v", err)
	}

	// Initialize components
	// accountState := state.NewState()
	stateManager := state.NewStateManager()
	c.dag = core.NewDAG()

	// Create block processor config
	blockConfig := &core.Config{
		BlockInterval:    1 * time.Second,
		ConsensusTimeout: 5 * time.Second,
		IsValidator:      true,
		NodeID:          c.config.NodeID,
	}
	
	// Create block processor
	c.blockProcessor = core.NewBlockProcessor(blockConfig, engineState, c.dag)
	
	// Load mempool transactions
	txs, err := storage.LoadMempool()
	if err != nil {
		return fmt.Errorf("failed to load mempool: %v", err)
	}
	for _, tx := range txs {
		c.blockProcessor.AddTransaction(tx)
	}
	
	// Create consensus config
	consensusConfig := &consensus.Config{
		TotalValidators: 3,
		WaveTimeout:     c.config.Consensus.WaveTimeout,
		QuorumSize:      2,
		ValidatorSet:    []types.Address{c.config.NodeID, types.Address("validator2"), types.Address("validator3")},
	}

	// Initialize consensus engine
	c.consensusEngine = consensus.NewConsensusEngine(consensusConfig, stateManager, c.blockProcessor)

	// Create wave controller
	c.waveController = consensus.NewWaveController(c.consensusEngine, c.config.Consensus.WaveTimeout)

	// Initialize stop channel
	c.stopChan = make(chan struct{})

	return nil
}

// handleCommand handles a command
func (c *CLI) handleCommand(line string) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "help":
		return c.handleHelp()
	case "status":
		return c.handleStatus()
	case "propose":
		return c.handlePropose(args)
	case "vote":
		return c.handleVote(args)
	case "blocks":
		return c.handleBlocks(args)
	case "block":
		return c.handleBlock(args)
	case "send":
		return c.handleSend(args)
	case "exit":
		os.Exit(0)
		return nil
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// handleHelp handles the help command
func (c *CLI) handleHelp() error {
	fmt.Println("Available commands:")
	fmt.Println("  help    - Show this help message")
	fmt.Println("  status  - Show current status")
	fmt.Println("  propose - Propose a new block")
	fmt.Println("  vote    - Vote on a proposal")
	fmt.Println("  blocks  - List recent blocks")
	fmt.Println("  block   - Show block details")
	fmt.Println("  send    - Send a transaction (send <from> <to> <amount>)")
	fmt.Println("  exit    - Exit the CLI")
	return nil
}

// handleStatus shows the current status
func (c *CLI) handleStatus() error {
	fmt.Printf("Current wave: %d\n", c.consensusEngine.GetCurrentWave())
	fmt.Printf("Current round: %d\n", c.currentRound)
	fmt.Printf("Is leader: %v\n", c.consensusEngine.IsLeader())
	return nil
}

// handlePropose handles the propose command
func (c *CLI) handlePropose(args []string) error {
	if !c.consensusEngine.IsLeader() {
		return fmt.Errorf("only the leader can propose blocks")
	}

	block, err := c.consensusEngine.CreateBlock()
	if err != nil {
		return fmt.Errorf("failed to create block: %v", err)
	}

	if err := c.consensusEngine.BroadcastBlock(block); err != nil {
		return fmt.Errorf("failed to broadcast block: %v", err)
	}

	fmt.Println("Block proposed successfully")
	return nil
}

// handleVote handles the vote command
func (c *CLI) handleVote(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: vote <proposal_id>")
	}

	proposalID := args[0]
	vote := &types.Vote{
		ProposalID: types.Hash(proposalID),
		Validator:  c.config.NodeID,
		Timestamp:  time.Now(),
	}

	if err := c.consensusEngine.HandleVote(vote); err != nil {
		return fmt.Errorf("failed to handle vote: %v", err)
	}

	fmt.Println("Vote submitted successfully")
	return nil
}

// handleBlocks handles the blocks command
func (c *CLI) handleBlocks(args []string) error {
	count := 10 // Default to showing 10 most recent blocks
	if len(args) > 0 {
		if _, err := fmt.Sscanf(args[0], "%d", &count); err != nil {
			return fmt.Errorf("invalid count: %v", err)
		}
	}

	blocks := c.consensusEngine.GetRecentBlocks(count)
	if len(blocks) == 0 {
		fmt.Println("No blocks found")
		return nil
	}

	fmt.Printf("Recent blocks (showing %d):\n", len(blocks))
	for _, block := range blocks {
		fmt.Printf("Height: %d, Hash: %s, Validator: %s\n",
			block.Header.Height,
			block.ComputeHash(),
			block.Header.Validator)
	}
	return nil
}

// handleBlock handles the block command
func (c *CLI) handleBlock(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: block <block_hash>")
	}

	block, err := c.consensusEngine.GetBlock(types.Hash(args[0]))
	if err != nil {
		return fmt.Errorf("failed to get block: %v", err)
	}

	fmt.Printf("Block Details:\n")
	fmt.Printf("  Height: %d\n", block.Header.Height)
	fmt.Printf("  Hash: %s\n", block.ComputeHash())
	fmt.Printf("  Validator: %s\n", block.Header.Validator)
	fmt.Printf("  Timestamp: %s\n", block.Header.Timestamp)
	fmt.Printf("  Parent Hash: %s\n", block.Header.ParentHash)
	fmt.Printf("  Transactions: %d\n", len(block.Body.Transactions))
	fmt.Printf("  References: %d\n", len(block.Header.References))
	return nil
}

// handleSend handles the send command
func (c *CLI) handleSend(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: send <from> <to> <amount>")
	}

	from := types.Address(args[0])
	to := types.Address(args[1])
	amount, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %v", err)
	}

	tx := &types.Transaction{
		From:      from,
		To:        to,
		Value:     types.Value(amount),
		Nonce:     0, // TODO: Get actual nonce from state
		GasLimit:  21000,
		GasPrice:  1,
		Data:      nil,
	}

	c.blockProcessor.AddTransaction(tx)
	fmt.Println("Transaction added to mempool")
	return nil
} 