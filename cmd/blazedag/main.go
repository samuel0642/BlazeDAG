package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/network"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 3000, "Port to listen on")
	isValidator := flag.Bool("validator", false, "Whether this node is a validator")
	flag.Parse()

	// Create P2P node
	nodeID := network.PeerID(fmt.Sprintf("node-%d", *port))
	node := network.NewP2PNode(nodeID)
	if err := node.Start(); err != nil {
		log.Fatalf("Failed to start P2P node: %v", err)
	}

	// Create message handler
	msgHandler := network.NewMessageHandler(node)

	// Create DAG
	dag := core.NewDAG()
	
	// Create state manager
	stateManager := state.NewStateManager()
	
	// Create EVM executor
	executor := core.NewEVMExecutor(stateManager)

	// Create consensus engine
	consensusEngine := consensus.NewConsensusEngine(dag, stateManager, executor, *isValidator)

	// Create wave controller
	// waveController := consensus.NewWaveController(consensusEngine)

	// Register message handlers
	msgHandler.RegisterHandler(network.MessageTypeBlock, func(msg *network.Message) error {
		block := &types.Block{}
		if err := msg.UnmarshalPayload(block); err != nil {
			return fmt.Errorf("failed to unmarshal block: %v", err)
		}
		return consensusEngine.HandleBlock(block)
	})

	msgHandler.RegisterHandler(network.MessageTypeVote, func(msg *network.Message) error {
		vote := &consensus.Vote{}
		if err := msg.UnmarshalPayload(vote); err != nil {
			return fmt.Errorf("failed to unmarshal vote: %v", err)
		}
		return consensusEngine.HandleVote(vote)
	})

	msgHandler.RegisterHandler(network.MessageTypeCertificate, func(msg *network.Message) error {
		cert := &types.Certificate{}
		if err := msg.UnmarshalPayload(cert); err != nil {
			return fmt.Errorf("failed to unmarshal certificate: %v", err)
		}
		return consensusEngine.HandleCertificate(cert)
	})

	// Start message handler
	if err := msgHandler.Start(); err != nil {
		log.Fatalf("Failed to start message handler: %v", err)
	}

	// Start consensus engine
	if err := consensusEngine.Start(); err != nil {
		log.Fatalf("Failed to start consensus engine: %v", err)
	}

	log.Printf("BlazeDAG node started on port %d (Validator: %v)", *port, *isValidator)
	log.Printf("Node ID: %s", nodeID)

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Cleanup
	log.Println("Shutting down...")
	consensusEngine.Stop()
	msgHandler.Stop()
	node.Stop()
	log.Println("Node stopped")
} 