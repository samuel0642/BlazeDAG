package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/types"
	"github.com/CrossDAG/BlazeDAG/internal/storage"
)

// CLI represents the command line interface
type CLI struct {
	config          *consensus.Config
	stateManager    *state.StateManager
	blockProcessor  *core.BlockProcessor
	consensusEngine *consensus.ConsensusEngine
	logger          *log.Logger
	scanner         *bufio.Scanner
	stopChan        chan struct{}
	currentRound    int
	approvedBlocks  map[string]bool // Changed from types.Hash to string
}

// NewCLI creates a new CLI instance
func NewCLI(config *consensus.Config) *CLI {
	return &CLI{
		config:         config,
		scanner:        bufio.NewScanner(os.Stdin),
		logger:         log.New(os.Stdout, "", log.LstdFlags),
		stopChan:       make(chan struct{}),
		approvedBlocks: make(map[string]bool),
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

	// Start block creation and round forwarding
	go c.runChain()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		c.Stop()
		os.Exit(0)
	}()

	// Start CLI loop
	c.logger.Printf("BlazeDAG CLI - Type 'help' for available commands")
	return c.run()
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
				fmt.Printf("1111111111")
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
	

				// Process the block through consensus
				if err := c.consensusEngine.HandleBlock(block); err != nil {
					log.Printf("Error processing block: %v", err)
					continue
				}

				// Check block approval status
				blockHash := block.ComputeHash()
				isApproved := c.consensusEngine.IsBlockApproved(blockHash)
				c.approvedBlocks[string(blockHash)] = isApproved

				// Log block status
				log.Printf("Block Status:")
				log.Printf("  Hash: %s", blockHash)
				log.Printf("  Wave: %d", currentWave)
				log.Printf("  Round: %d", round)
				log.Printf("  Height: %d", height)
				log.Printf("  Approved: %v", isApproved)
				log.Printf("  Votes: %d/%d", c.consensusEngine.GetBlockVotes(blockHash), c.config.QuorumSize)

				// Update current round and mark block as created
				c.currentRound = round
				blockCreatedInWave = true

				// Increment round and height
				round++
				height++
			}

			// Sleep for the configured round duration
			time.Sleep(c.config.RoundDuration)
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
	// Initialize state manager
	c.stateManager = state.NewStateManager()

	// Create block processor config
	blockConfig := &core.Config{
		BlockInterval:    1 * time.Second,
		ConsensusTimeout: 5 * time.Second,
		IsValidator:      true,
		NodeID:          types.Address(c.config.NodeID),
	}

	// Create DAG
	dag := core.NewDAG()

	// Create initial state
	initialState := types.NewState()

	// Create storage
	storage, err := storage.NewStorage("data")
	if err != nil {
		return fmt.Errorf("failed to create storage: %v", err)
	}

	// Create state manager
	stateManager := core.NewStateManager(initialState, storage)

	// Initialize block processor
	c.blockProcessor = core.NewBlockProcessor(blockConfig, stateManager, dag)

	// Initialize consensus engine
	c.consensusEngine = consensus.NewConsensusEngine(c.config, c.stateManager, c.blockProcessor)

	return nil
}

// run runs the CLI loop
func (c *CLI) run() error {
	for {
		// Read command
		var cmd string
		fmt.Print("> ")
		if !c.scanner.Scan() {
			continue
		}

		cmd = c.scanner.Text()

		// Handle command
		switch cmd {
		case "help":
			c.printHelp()
		case "status":
			c.handleStatus()
		case "propose":
			c.handlePropose()
		case "vote":
			c.handleVote()
		case "blocks":
			c.handleBlocks()
		case "block":
			c.handleBlock()
		case "send":
			c.handleSend()
		case "showblocks":
			c.showBlocks()
		case "exit":
			c.Stop()
			return nil
		default:
			c.logger.Printf("Unknown command: %s", cmd)
		}
	}
}

// printHelp prints the help message
func (c *CLI) printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help    - Show this help message")
	fmt.Println("  status  - Show current status")
	fmt.Println("  propose - Propose a new block")
	fmt.Println("  vote    - Vote on a proposal")
	fmt.Println("  blocks  - List recent blocks")
	fmt.Println("  block   - Show block details")
	fmt.Println("  send    - Send a transaction (send <from> <to> <amount>)")
	fmt.Println("  showblocks - Show all blocks in the chain")
	fmt.Println("  exit    - Exit the CLI")
}

// handleStatus shows the current status
func (c *CLI) handleStatus() error {
	fmt.Printf("Current wave: %d\n", c.consensusEngine.GetCurrentWave())
	fmt.Printf("Current round: %d\n", c.currentRound)
	fmt.Printf("Is leader: %v\n", c.consensusEngine.IsLeader())
	return nil
}

// handlePropose handles the propose command
func (c *CLI) handlePropose() error {
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
func (c *CLI) handleVote() error {
	if !c.consensusEngine.IsLeader() {
		return fmt.Errorf("only the leader can vote")
	}

	proposalID := c.config.NodeID
	vote := &types.Vote{
		ProposalID: types.Hash(proposalID),
		Validator:  types.Address(c.config.NodeID),
		Timestamp:  time.Now(),
	}

	if err := c.consensusEngine.HandleVote(vote); err != nil {
		return fmt.Errorf("failed to handle vote: %v", err)
	}

	fmt.Println("Vote submitted successfully")
	return nil
}

// handleBlocks handles the blocks command
func (c *CLI) handleBlocks() error {
	count := 10 // Default to showing 10 most recent blocks
	blocks := c.consensusEngine.GetRecentBlocks(count)
	if len(blocks) == 0 {
		fmt.Println("No blocks found")
		return nil
	}

	fmt.Printf("Recent blocks (showing %d):\n", len(blocks))
	for _, block := range blocks {
		blockHash := block.ComputeHash()
		isApproved := c.approvedBlocks[string(blockHash)]
		votes := c.consensusEngine.GetBlockVotes(blockHash)
		
		fmt.Printf("Height: %d, Hash: %s, Validator: %s, Approved: %v, Votes: %d/%d\n",
			block.Header.Height,
			blockHash,
			block.Header.Validator,
			isApproved,
			votes,
			c.config.QuorumSize)
	}
	return nil
}

// handleBlock handles the block command
func (c *CLI) handleBlock() error {
	if !c.consensusEngine.IsLeader() {
		return fmt.Errorf("only the leader can show block details")
	}

	block, err := c.consensusEngine.GetBlock(types.Hash(string(c.config.NodeID)))
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
func (c *CLI) handleSend() error {
	if !c.consensusEngine.IsLeader() {
		return fmt.Errorf("only the leader can send transactions")
	}

	from := types.Address(c.config.NodeID)
	to := types.Address(c.config.NodeID)
	amount, err := strconv.ParseUint(c.config.NodeID, 10, 64)
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

// showBlocks shows all blocks in the chain
func (c *CLI) showBlocks() error {
	blocks, err := c.stateManager.GetAllBlocks()
	if err != nil {
		return fmt.Errorf("failed to get blocks: %v", err)
	}

	if len(blocks) == 0 {
		fmt.Println("No blocks found in the chain")
		return nil
	}

	fmt.Printf("Found %d blocks in the chain:\n\n", len(blocks))
	for _, block := range blocks {
		fmt.Printf("Block %s:\n", block.ComputeHash())
		fmt.Printf("  Height: %d\n", block.Header.Height)
		fmt.Printf("  Validator: %s\n", block.Header.Validator)
		fmt.Printf("  Timestamp: %s\n", block.Header.Timestamp)
		fmt.Printf("  Parent Hash: %s\n", block.Header.ParentHash)
		fmt.Printf("  Transactions: %d\n", len(block.Body.Transactions))
		fmt.Println()
	}

	return nil
} 