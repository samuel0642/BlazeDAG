package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 3000, "Port to listen on")
	isValidator := flag.Bool("validator", false, "Whether this node is a validator")
	nodeIDStr := flag.String("nodeid", "", "Node ID (required for validators)")
	flag.Parse()

	// Validate flags
	if *isValidator && *nodeIDStr == "" {
		log.Fatal("Node ID is required for validators")
	}

	// Create node ID
	var nodeID types.Address
	if *nodeIDStr != "" {
		nodeID = types.Address(*nodeIDStr)
	}

	// Create and start CLI
	cli := NewCLI(*isValidator, nodeID)
	if err := cli.Start(*port); err != nil {
		log.Fatalf("Failed to start CLI: %v", err)
	}
	defer cli.Stop()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cli.Stop()
		os.Exit(0)
	}()

	// Run CLI
	cli.Run()
} 