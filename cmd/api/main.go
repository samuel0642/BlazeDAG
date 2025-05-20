package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/api"
	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/storage"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	apiPort := flag.Int("port", 8080, "API server port")
	flag.Parse()

	// Load configuration
	cfg, err := consensus.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize components
	stateManager := state.NewStateManager()

	// Create block processor config
	blockConfig := &core.Config{
		BlockInterval:    1 * time.Second,
		ConsensusTimeout: 5 * time.Second,
		IsValidator:      true,
		NodeID:           types.Address(cfg.NodeID),
	}

	// Use singleton DAG
	dag := core.GetDAG()

	// Create initial state
	initialState := types.NewState()

	// Create storage
	storage, err := storage.NewStorage("data")
	if err != nil {
		fmt.Printf("Error creating storage: %v\n", err)
		os.Exit(1)
	}

	// Create state manager (using core package)
	coreStateManager := core.NewStateManager(initialState, storage)

	// Initialize block processor
	blockProcessor := core.NewBlockProcessor(blockConfig, coreStateManager, dag)

	// Initialize consensus engine
	engine := consensus.NewConsensusEngine(cfg, stateManager, blockProcessor)

	// Start consensus engine
	if err := engine.Start(); err != nil {
		fmt.Printf("Error starting consensus engine: %v\n", err)
		os.Exit(1)
	}

	// Create API server
	server := api.NewServer(engine, *apiPort)

	// Start API server
	if err := server.Start(); err != nil {
		fmt.Printf("Error starting API server: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("API server started on port %d\n", *apiPort)

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Stop API server
	if err := server.Stop(); err != nil {
		fmt.Printf("Error stopping API server: %v\n", err)
	}

	// Stop consensus engine
	engine.Stop()

	fmt.Println("Shutdown complete")
}
