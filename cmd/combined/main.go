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

	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/dag"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

func main() {
	validatorID := flag.String("id", "validator1", "Validator ID")
	dagListenAddr := flag.String("dag-listen", "localhost:4001", "DAG sync listen address")
	waveListenAddr := flag.String("wave-listen", "localhost:5001", "Wave consensus listen address")
	dagPeersStr := flag.String("dag-peers", "", "Comma-separated DAG sync peer addresses")
	wavePeersStr := flag.String("wave-peers", "", "Comma-separated wave consensus peer addresses")
	roundDuration := flag.Duration("round-duration", 2*time.Second, "Round duration for DAG sync")
	waveDuration := flag.Duration("wave-duration", 3*time.Second, "Wave duration for consensus")
	flag.Parse()

	// Parse DAG peers
	var dagPeers []string
	if *dagPeersStr != "" {
		dagPeers = strings.Split(*dagPeersStr, ",")
		for i, peer := range dagPeers {
			dagPeers[i] = strings.TrimSpace(peer)
		}
	}

	// Parse wave peers
	var wavePeers []string
	if *wavePeersStr != "" {
		wavePeers = strings.Split(*wavePeersStr, ",")
		for i, peer := range wavePeers {
			wavePeers[i] = strings.TrimSpace(peer)
		}
	}

	// Create DAG sync
	dagSync := dag.NewDAGSync(
		types.Address(*validatorID),
		*dagListenAddr,
		dagPeers,
		*roundDuration,
	)

	// Create wave consensus that reads from DAG sync
	waveConsensus := consensus.NewWaveConsensus(
		types.Address(*validatorID),
		dagSync, // Pass DAG sync reference
		*waveListenAddr,
		wavePeers,
		*waveDuration,
	)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n=== Shutting Down ===")
		waveConsensus.Stop()
		dagSync.Stop()
		os.Exit(0)
	}()

	// Start DAG sync first
	if err := dagSync.Start(); err != nil {
		log.Fatalf("Failed to start DAG sync: %v", err)
	}

	// Wait a bit for DAG sync to initialize
	time.Sleep(1 * time.Second)

	// Start wave consensus
	if err := waveConsensus.Start(); err != nil {
		log.Fatalf("Failed to start wave consensus: %v", err)
	}

	fmt.Printf("=== BlazeDAG Combined Started ===\n")
	fmt.Printf("Validator: %s\n", *validatorID)
	fmt.Printf("DAG Sync: %s\n", *dagListenAddr)
	fmt.Printf("Wave Consensus: %s\n", *waveListenAddr)
	if len(dagPeers) > 0 {
		fmt.Printf("DAG Peers: %v\n", dagPeers)
	}
	if len(wavePeers) > 0 {
		fmt.Printf("Wave Peers: %v\n", wavePeers)
	}
	fmt.Printf("Round Duration: %v\n", *roundDuration)
	fmt.Printf("Wave Duration: %v\n", *waveDuration)
	fmt.Printf("================================\n")

	// Status reporting
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-sigChan:
				return
			case <-ticker.C:
				dagStatus := dagSync.GetDAGStatus()
				waveStatus := waveConsensus.GetWaveStatus()
				
				fmt.Printf("\n=== Status Report [%s] ===\n", time.Now().Format("15:04:05"))
				fmt.Printf("DAG: Round %d, %d blocks, %d peers\n", 
					dagStatus["current_round"], dagStatus["total_blocks"], dagStatus["connected_peers"])
				fmt.Printf("Wave: Wave %d, %d finalized, %d peers\n", 
					waveStatus["current_wave"], waveStatus["finalized_waves"], waveStatus["connected_peers"])
				fmt.Printf("========================\n")
			}
		}
	}()
	
	select {} // Block forever
} 