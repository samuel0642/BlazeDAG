package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/config"
	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// CLI represents the command line interface
type CLI struct {
	config          *config.Config
	consensusEngine *consensus.ConsensusEngine
	waveController  *consensus.WaveController
	blockProcessor  *core.BlockProcessor
	dag             *core.DAG
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
	wave := types.Wave(1)
	height := types.BlockNumber(0)
	lastWave := types.Wave(0)

	for {
		select {
		case <-c.stopChan:
			return
		default:
			// Create a block with current round number
			block, err := c.blockProcessor.CreateBlock(types.Round(round))
			if err != nil {
				log.Printf("Error creating block: %v", err)
				continue
			}

			// Set block properties
			block.Header.Wave = wave
			block.Header.Height = height

			// Show leader selection when wave changes
			if wave != lastWave {
				log.Printf("\n=== Wave %d Leader Selection ===", wave)
				log.Printf("Selected Leader: %s", c.config.NodeID)
				lastWave = wave
			}

			// Add block to DAG
			if err := c.dag.AddBlock(block); err != nil {
				log.Printf("Error adding block to DAG: %v", err)
				continue
			}

			// Process the block through consensus
			if err := c.consensusEngine.HandleBlock(block); err != nil {
				log.Printf("Error processing block: %v", err)
				continue
			}

			// Update current round
			c.currentRound = round

			// Increment round and wave together
			round++
			wave = types.Wave(round) // Wave number matches round number
			height++

			// Sleep to simulate block interval
			time.Sleep(1 * time.Second)
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
	// Initialize components
	stateManager := state.NewStateManager()
	c.dag = core.NewDAG()
	coreState := &core.State{
		CurrentWave: 0,
		LatestBlock: nil,
	}

	// Create block processor config
	blockConfig := &core.Config{
		BlockInterval:    1 * time.Second,
		ConsensusTimeout: 5 * time.Second,
		IsValidator:      true,
		NodeID:          c.config.NodeID,
	}
	
	// Create block processor
	c.blockProcessor = core.NewBlockProcessor(blockConfig, coreState, c.dag)
	
	// Create consensus config
	consensusConfig := &consensus.Config{
		TotalValidators: 3,
		WaveTimeout:     5 * time.Second,
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