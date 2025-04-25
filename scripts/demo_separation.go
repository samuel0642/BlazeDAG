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
	wave := types.Wave(1)
	height := types.BlockNumber(0)
	lastWave := types.Wave(0)
	
	for {
		// Get transactions from mempool
		txs := mempool.GetTransactions()
		
		// Create a block with current round number and transactions
		block, err := blockProcessor.CreateBlock(types.Round(round))
		if err != nil {
			fmt.Printf("Error creating block: %v\n", err)
			return
		}

		// Select N-f references from previous blocks
		N := consensusConfig.TotalValidators
		f := (N - 1) / 3 // Byzantine fault tolerance
		requiredRefs := N - f

		// Get recent blocks to use as references
		recentBlocks := dag.GetRecentBlocks(int(requiredRefs))
		references := make([]*types.Reference, 0)
		for _, refBlock := range recentBlocks {
			ref := &types.Reference{
				BlockHash: refBlock.ComputeHash(),
				Round:     refBlock.Header.Round,
				Wave:      refBlock.Header.Wave,
				Type:      types.ReferenceTypeStandard,
			}
			references = append(references, ref)
		}
		block.Header.References = references

		// Set block properties
		block.Header.Wave = wave
		block.Header.Height = height
		block.Body.Transactions = txs

		// Show leader selection when wave changes
		if wave != lastWave {
			fmt.Printf("\n=== Wave %d Leader Selection ===\n", wave)
			fmt.Printf("Selected Leader: %s\n", blockConfig.NodeID)
			fmt.Printf("Validator Set: %v\n", consensusConfig.ValidatorSet)
			lastWave = wave
		}

		// Set wave number (increment every 2 rounds)
		if round % 2 == 0 {
			wave++
		}

		fmt.Printf("\nBlock from Validator %s:\n", blockConfig.NodeID)
		fmt.Printf("  Height: %d\n", block.Header.Height)
		fmt.Printf("  Hash: %x\n", block.ComputeHash())
		fmt.Printf("  Round: %d\n", block.Header.Round)
		fmt.Printf("  Wave: %d\n", block.Header.Wave)
		fmt.Printf("  Transaction Count: %d\n", len(block.Body.Transactions))
		if len(block.Body.Transactions) > 0 {
			fmt.Println("  Transactions:")
			for i, tx := range block.Body.Transactions {
				fmt.Printf("    %d: From %s to %s, Value: %d\n", i+1, tx.From, tx.To, tx.Value)
			}
		}

		// Display block references
		if len(block.Header.References) > 0 {
			fmt.Println("  References:")
			for i, ref := range block.Header.References {
				fmt.Printf("    %d: Block Hash: %x\n", i+1, ref.BlockHash)
				fmt.Printf("      Round: %d, Wave: %d\n", ref.Round, ref.Wave)
			}
		}

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

		// Increment counters
		round++
		height++

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