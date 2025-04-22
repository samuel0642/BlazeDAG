package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samuel0642/BlazeDAG/internal/consensus"
	"github.com/samuel0642/BlazeDAG/internal/dag"
	"github.com/samuel0642/BlazeDAG/internal/evm"
	"github.com/samuel0642/BlazeDAG/internal/network"
	"github.com/samuel0642/BlazeDAG/internal/state"
)

var (
	port        = flag.Int("port", 9000, "P2P port")
	validator   = flag.Bool("validator", false, "Run as validator")
	genesisFile = flag.String("genesis", "genesis.json", "Genesis file path")
)

func main() {
	flag.Parse()

	// Create P2P node
	node, err := network.NewP2PNode(*port)
	if err != nil {
		log.Fatalf("Failed to create P2P node: %v", err)
	}

	// Create DAG
	dag := dag.NewDAG()

	// Create state
	state := state.NewState()

	// Create EVM executor
	executor := evm.NewEVMExecutor(state, big.NewInt(1))

	// Create consensus engine
	config := consensus.Config{
		TotalValidators: 10,
		FaultTolerance:  3,
		RoundDuration:   time.Second * 2,
		WaveTimeout:     time.Second * 10,
	}
	engine := consensus.NewEngine(config)

	// Create wave controller
	waveController := dag.NewWaveController(dag, time.Second*10)

	// Create message handler
	messageHandler := network.NewMessageHandler(node)

	// Register message handlers
	messageHandler.RegisterHandler(network.MessageTypeBlock, handleBlock)
	messageHandler.RegisterHandler(network.MessageTypeVote, handleVote)
	messageHandler.RegisterHandler(network.MessageTypeCertificate, handleCertificate)

	// Start message handler
	messageHandler.Start()

	// Start peer discovery
	go node.discoverPeers()

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("BlazeDAG node started on port %d\n", *port)
	fmt.Printf("Running as validator: %v\n", *validator)

	<-sigCh
	fmt.Println("Shutting down...")

	// Cleanup
	messageHandler.Stop()
	node.Close()
}

func handleBlock(msg *network.Message) error {
	// Handle block message
	return nil
}

func handleVote(msg *network.Message) error {
	// Handle vote message
	return nil
}

func handleCertificate(msg *network.Message) error {
	// Handle certificate message
	return nil
} 