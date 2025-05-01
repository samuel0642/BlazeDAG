package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/CrossDAG/BlazeDAG/internal/consensus"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := consensus.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Start CLI
	cli := NewCLI(cfg)
	if err := cli.Start(); err != nil {
		fmt.Printf("Error starting CLI: %v\n", err)
		os.Exit(1)
	}
} 