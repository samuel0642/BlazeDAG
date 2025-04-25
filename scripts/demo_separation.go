package main

import (
	"fmt"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

func main() {
	fmt.Println("Running BlazeDAG component separation demo...")

	// Initialize components
	stateManager := state.NewStateManager()
	dag := core.NewDAG()
	coreState := &core.State{
		CurrentWave: 0,
		LatestBlock: nil,
	}
	mempool := core.NewMempool()

	// Create block processor config
	blockConfig := &core.Config{
		BlockInterval:    1 * time.Second,
		ConsensusTimeout: 5 * time.Second,
		IsValidator:      true,
		NodeID:          types.Address("validator1"),
	}
	
	blockProcessor := core.NewBlockProcessor(blockConfig, coreState, dag)
	
	// Create consensus config
	consensusConfig := &consensus.Config{
		TotalValidators: 3,
		WaveTimeout:     5 * time.Second,
		QuorumSize:      2,
		ValidatorSet:    []types.Address{types.Address("validator1"), types.Address("validator2"), types.Address("validator3")},
	}

	// Initialize consensus engine
	consensusEngine := consensus.NewConsensusEngine(consensusConfig, stateManager, blockProcessor)

	// Create some sample transactions
	tx1 := &types.Transaction{
		From:  types.Address("0x123"),
		To:    types.Address("0x456"),
		Value: 100,
		Data:  []byte("test1"),
	}

	tx2 := &types.Transaction{
		From:  types.Address("0x789"),
		To:    types.Address("0xabc"),
		Value: 200,
		Data:  []byte("test2"),
	}

	// Add transactions to the mempool
	mempool.AddTransaction(tx1)
	mempool.AddTransaction(tx2)

	fmt.Println("\n=== Creating and Processing Blocks ===")
	
	// Create and process blocks continuously with separate round and wave timing
	round := 1
	wave := 1
	
	for {
		// Create a block with current round number
		block, err := blockProcessor.CreateBlock(types.Round(round))
		if err != nil {
			fmt.Printf("Error creating block: %v\n", err)
			return
		}

		// Set wave number (increment every 2 rounds)
		block.Header.Wave = types.Wave(wave)
		if round % 2 == 0 {
			wave++
		}

		fmt.Printf("\nBlock:\n")
		fmt.Printf("  Height: %d\n", block.Header.Height)
		fmt.Printf("  Hash: %x\n", block.ComputeHash())
		fmt.Printf("  Round: %d\n", block.Header.Round)
		fmt.Printf("  Wave: %d\n", block.Header.Wave)
		fmt.Printf("  Transaction Count: %d\n", len(block.Body.Transactions))

		// Add block to DAG
		if err := dag.AddBlock(block); err != nil {
			fmt.Printf("Error adding block to DAG: %v\n", err)
			return
		}

		// Process the block through consensus
		err = consensusEngine.HandleBlock(block)
		if err != nil {
			fmt.Printf("Error processing block: %v\n", err)
			return
		}

		// Verify state update
		stateRoot := stateManager.GetStateRoot()
		fmt.Printf("  State Root: %x\n", stateRoot)

		// Increment round counter
		round++

		// Sleep to simulate block interval
		time.Sleep(1 * time.Second)
	}

	// Print DAG statistics
	fmt.Println("\n=== DAG Statistics ===")
	fmt.Printf("Total Blocks: %d\n", dag.GetBlockCount())
	fmt.Printf("Latest Height: %d\n", dag.GetLatestHeight())
	fmt.Printf("Max Height: %d\n", dag.GetMaxHeight())

	// Get and print recent blocks
	fmt.Println("\n=== Recent Blocks ===")
	recentBlocks := dag.GetRecentBlocks(3)
	for i, block := range recentBlocks {
		fmt.Printf("Block %d:\n", i+1)
		fmt.Printf("  Height: %d\n", block.Header.Height)
		fmt.Printf("  Hash: %x\n", block.ComputeHash())
		fmt.Printf("  Round: %d\n", block.Header.Round)
		fmt.Printf("  Wave: %d\n", block.Header.Wave)
	}

	// Let the engine run for a while
	fmt.Println("\nWaiting for 5 seconds...")
	time.Sleep(5 * time.Second)
} 