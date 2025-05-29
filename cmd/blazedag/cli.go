package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/storage"
)

// CLI represents the command line interface with independent DAG and consensus
type CLI struct {
	config      *consensus.Config
	storage     *storage.Storage
	coordinator *core.BlazeDagCoordinator
	logger      *log.Logger
	scanner     *bufio.Scanner
	stopChan    chan struct{}
}

// NewCLI creates a new CLI instance
func NewCLI(config *consensus.Config) *CLI {
	return &CLI{
		config:   config,
		scanner:  bufio.NewScanner(os.Stdin),
		logger:   log.New(os.Stdout, "", log.LstdFlags),
		stopChan: make(chan struct{}),
	}
}

// Start starts the CLI with independent DAG and consensus
func (c *CLI) Start() error {
	// Initialize components
	if err := c.initialize(); err != nil {
		return fmt.Errorf("failed to initialize: %v", err)
	}

	// Start the independent BlazeDAG coordinator
	if err := c.coordinator.Start(); err != nil {
		return fmt.Errorf("failed to start BlazeDAG coordinator: %v", err)
	}

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		c.Stop()
		os.Exit(0)
	}()

	// Start CLI loop
	c.logger.Printf("=== BlazeDAG Independent Architecture Started ===")
	c.logger.Printf("DAG Rounds and Consensus Waves operating independently")
	c.logger.Printf("Type 'help' for available commands")
	return c.run()
}

// Stop stops the CLI
func (c *CLI) Stop() {
	c.logger.Println("Stopping BlazeDAG...")
	close(c.stopChan)
	if c.coordinator != nil {
		c.coordinator.Stop()
	}
	c.logger.Println("BlazeDAG stopped")
}

// initialize initializes the CLI components
func (c *CLI) initialize() error {
	// Initialize storage
	var err error
	c.storage, err = storage.NewStorage("./data")
	if err != nil {
		return fmt.Errorf("failed to create storage: %v", err)
	}

	// Create core config
	coreConfig := &core.Config{
		NodeID:           c.config.ListenAddr,
		BlockInterval:    1 * time.Second,
		ConsensusTimeout: 5 * time.Second,
		IsValidator:      true,
	}

	// Create the independent BlazeDAG coordinator
	c.coordinator = core.NewBlazeDagCoordinator(coreConfig, c.storage)

	c.logger.Printf("Initialized BlazeDAG with independent DAG transport and wave consensus")
	return nil
}

// run runs the CLI command loop
func (c *CLI) run() error {
	// Start status monitoring
	go c.runStatusMonitor()

	for {
		fmt.Print("blazedag> ")
		if !c.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(c.scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		switch command {
		case "help":
			c.printHelp()
		case "status":
			c.handleStatus()
		case "dag":
			c.handleDAGStatus()
		case "consensus":
			c.handleConsensusStatus()
		case "system":
			c.handleSystemStatus()
		case "committed":
			c.handleCommittedTransactions()
		case "exit", "quit":
			return nil
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", command)
		}
	}

	return nil
}

// runStatusMonitor runs a background status monitor
func (c *CLI) runStatusMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			status := c.coordinator.GetSystemStatus()
			dagStatus := status["dag_transport"].(map[string]interface{})
			consensusStatus := status["wave_consensus"].(map[string]interface{})
			
			c.logger.Printf("MONITOR - DAG Round: %v, Wave: %v, Blocks: %v, Committed: %v",
				dagStatus["current_dag_round"],
				consensusStatus["current_wave"],
				dagStatus["total_blocks"],
				consensusStatus["committed_blocks"])
		}
	}
}

// printHelp prints available commands
func (c *CLI) printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help      - Show this help message")
	fmt.Println("  status    - Show brief system status")
	fmt.Println("  dag       - Show DAG transport layer status")
	fmt.Println("  consensus - Show wave consensus layer status")
	fmt.Println("  system    - Show complete system status")
	fmt.Println("  committed - Show committed transactions")
	fmt.Println("  exit      - Exit the CLI")
	fmt.Println("")
	fmt.Println("BlazeDAG Independence Features:")
	fmt.Println("  - DAG rounds progress at network speed (independent)")
	fmt.Println("  - Consensus waves run separately (independent)")
	fmt.Println("  - No synchronization between rounds and waves")
	fmt.Println("  - High throughput through parallelization")
}

// handleStatus shows brief system status
func (c *CLI) handleStatus() {
	status := c.coordinator.GetSystemStatus()
	
	fmt.Println("=== BlazeDAG Brief Status ===")
	fmt.Printf("Validator ID: %s\n", status["validator_id"])
	fmt.Printf("Pending Transactions: %v\n", status["pending_transactions"])
	
	dagStatus := status["dag_transport"].(map[string]interface{})
	fmt.Printf("Current DAG Round: %v\n", dagStatus["current_dag_round"])
	
	consensusStatus := status["wave_consensus"].(map[string]interface{})
	fmt.Printf("Current Consensus Wave: %v\n", consensusStatus["current_wave"])
	
	independence := status["independence_demo"].(map[string]interface{})
	fmt.Printf("Independence: DAG=%v, Waves=%v\n", 
		independence["dag_rounds_independent"], 
		independence["waves_independent"])
}

// handleDAGStatus shows DAG transport layer status
func (c *CLI) handleDAGStatus() {
	dagStatus := c.coordinator.GetDAGStatus()
	
	fmt.Println("=== DAG Transport Layer Status ===")
	fmt.Printf("Current DAG Round: %v\n", dagStatus["current_dag_round"])
	fmt.Printf("Total Blocks in DAG: %v\n", dagStatus["total_blocks"])
	fmt.Printf("DAG Height: %v\n", dagStatus["dag_height"])
	fmt.Printf("Uncommitted Blocks: %v\n", dagStatus["uncommitted_blocks"])
	fmt.Println("Status: Operating independently at network speed")
}

// handleConsensusStatus shows wave consensus layer status
func (c *CLI) handleConsensusStatus() {
	consensusStatus := c.coordinator.GetConsensusStatus()
	
	fmt.Println("=== Wave Consensus Layer Status ===")
	fmt.Printf("Current Wave: %v\n", consensusStatus["current_wave"])
	fmt.Printf("Committed Blocks: %v\n", consensusStatus["committed_blocks"])
	fmt.Printf("Is Leader: %v\n", consensusStatus["is_leader"])
	fmt.Println("Status: Operating independently from DAG rounds")
}

// handleSystemStatus shows complete system status
func (c *CLI) handleSystemStatus() {
	status := c.coordinator.GetSystemStatus()
	
	fmt.Println("=== Complete BlazeDAG System Status ===")
	fmt.Printf("Validator ID: %s\n", status["validator_id"])
	fmt.Printf("Pending Transactions: %v\n", status["pending_transactions"])
	
	fmt.Println("\n--- DAG Transport Layer ---")
	dagStatus := status["dag_transport"].(map[string]interface{})
	for key, value := range dagStatus {
		fmt.Printf("  %s: %v\n", key, value)
	}
	
	fmt.Println("\n--- Wave Consensus Layer ---")
	consensusStatus := status["wave_consensus"].(map[string]interface{})
	for key, value := range consensusStatus {
		fmt.Printf("  %s: %v\n", key, value)
	}
	
	fmt.Println("\n--- Independence Demonstration ---")
	independence := status["independence_demo"].(map[string]interface{})
	for key, value := range independence {
		fmt.Printf("  %s: %v\n", key, value)
	}
}

// handleCommittedTransactions shows committed transactions
func (c *CLI) handleCommittedTransactions() {
	transactions := c.coordinator.GetCommittedTransactions()
	
	fmt.Printf("=== Committed Transactions (%d total) ===\n", len(transactions))
	if len(transactions) == 0 {
		fmt.Println("No transactions committed yet")
		return
	}
	
	// Show last 10 transactions
	start := len(transactions) - 10
	if start < 0 {
		start = 0
	}
	
	fmt.Printf("Showing last %d transactions:\n", len(transactions)-start)
	for i := start; i < len(transactions); i++ {
		tx := transactions[i]
		fmt.Printf("  TX %d: From=%s, To=%s, Value=%d, State=%d\n",
			i+1, tx.From, tx.To, tx.Value, tx.State)
	}
}
