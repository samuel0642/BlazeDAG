package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/dag"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

func main() {
	validatorID := flag.String("id", "validator1", "Validator ID")
	listenAddr := flag.String("listen", "localhost:3001", "Listen address")
	peersStr := flag.String("peers", "", "Comma-separated peer addresses")
	flag.Parse()

	// Parse peers
	var peers []string
	if *peersStr != "" {
		peers = strings.Split(*peersStr, ",")
		for i, peer := range peers {
			peers[i] = strings.TrimSpace(peer)
		}
	}

	dagSync := dag.NewDAGSync(
		types.Address(*validatorID),
		*listenAddr,
		peers,
		1*time.Second,
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		dagSync.Stop()
		os.Exit(0)
	}()

	if err := dagSync.Start(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}

	fmt.Printf("=== DAG Sync Started ===\n")
	fmt.Printf("Validator: %s\n", *validatorID)
	fmt.Printf("Listen: %s\n", *listenAddr)
	if len(peers) > 0 {
		fmt.Printf("Peers: %v\n", peers)
	} else {
		fmt.Printf("Peers: none (single validator mode)\n")
	}
	fmt.Printf("========================\n")
	
	select {} // Block forever
} 