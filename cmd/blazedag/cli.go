package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/CrossDAG/BlazeDAG/internal/config"
	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/state"
)

// CLI represents the command line interface
type CLI struct {
	config          *config.Config
	consensusEngine *consensus.ConsensusEngine
	waveController  *consensus.WaveController
	scanner         *bufio.Scanner
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

// initialize initializes the CLI components
func (c *CLI) initialize() error {
	// Create DAG
	dag := core.NewDAG()

	// Create state manager
	stateManager := state.NewStateManager()

	// Create consensus engine
	consensusConfig := &consensus.Config{
		WaveTimeout:   c.config.Consensus.WaveTimeout,
		RoundDuration: c.config.Consensus.RoundDuration,
		ValidatorSet:  c.config.Consensus.ValidatorSet,
		QuorumSize:    c.config.Consensus.QuorumSize,
		ListenAddr:    c.config.Consensus.ListenAddr,
		Seeds:         c.config.Consensus.Seeds,
	}

	c.consensusEngine = consensus.NewConsensusEngine(dag, stateManager, c.config.NodeID, consensusConfig)

	// Create wave controller
	c.waveController = consensus.NewWaveController(c.consensusEngine, c.config.Consensus.WaveTimeout)

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
	fmt.Println("  exit    - Exit the CLI")
	return nil
}

// handleStatus handles the status command
func (c *CLI) handleStatus() error {
	fmt.Printf("Current wave: %d\n", c.consensusEngine.GetCurrentWave())
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
	if err := c.consensusEngine.HandleVote(c.config.NodeID, proposalID); err != nil {
		return fmt.Errorf("failed to handle vote: %v", err)
	}

	fmt.Println("Vote submitted successfully")
	return nil
} 