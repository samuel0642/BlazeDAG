package main

import (
	"encoding/json"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	// "strconv"
	"strings"
	"syscall"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/dag"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// RemoteDAGSync implements a client to read blocks from a remote DAG sync via HTTP
type RemoteDAGSync struct {
	validatorID types.Address
	dagAddr     string
	dag         *dag.DAG
	httpClient  *http.Client
}

func NewRemoteDAGSync(validatorID types.Address, dagAddr string) *RemoteDAGSync {
	return &RemoteDAGSync{
		validatorID: validatorID,
		dagAddr:     dagAddr,
		dag:         dag.NewDAG(),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// BlockResponse represents the JSON response from DAG sync API
type BlockResponse struct {
	Hash      string                 `json:"hash"`
	Validator string                 `json:"validator"`
	Timestamp time.Time              `json:"timestamp"`
	Round     types.Round            `json:"round"`
	Wave      types.Wave             `json:"wave"`
	Height    types.BlockNumber      `json:"height"`
	TxCount   int                    `json:"txCount"`
}

// GetRecentBlocks returns recent blocks from DAG sync
func (rds *RemoteDAGSync) GetRecentBlocks(count int) []*types.Block {
	// Try to connect to DAG sync HTTP API
	url := fmt.Sprintf("http://%s:8080/blocks?count=%d", 
		strings.Split(rds.dagAddr, ":")[0], count)
	
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Wave Consensus: Failed to connect to DAG sync API: %v", err)
		return rds.getFallbackBlocks()
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		log.Printf("Wave Consensus: DAG sync API returned status %d", resp.StatusCode)
		return rds.getFallbackBlocks()
	}
	
	var blockResponses []BlockResponse
	if err := json.NewDecoder(resp.Body).Decode(&blockResponses); err != nil {
		log.Printf("Wave Consensus: Failed to decode blocks response: %v", err)
		return rds.getFallbackBlocks()
	}
	
	// log.Printf("Wave Consensus: Received %d blocks from DAG sync", len(blockResponses))
	
	// // Log details of received blocks from HTTP API
	// log.Printf("🌊 Wave Consensus: RECEIVED BLOCKS from DAG API:")
	// for i, blockResp := range blockResponses {
	// 	log.Printf("   [%d] 📦 Hash: %s", i+1, blockResp.Hash)
	// 	log.Printf("   [%d] 🔄 Round: %d", i+1, blockResp.Round)
	// 	log.Printf("   [%d] 📏 Height: %d", i+1, blockResp.Height)
	// 	log.Printf("   [%d] 👤 Validator: %s", i+1, blockResp.Validator)
	// 	log.Printf("   [%d] 💼 TxCount: %d", i+1, blockResp.TxCount)
	// 	log.Printf("   [%d] ⏰ Timestamp: %s", i+1, blockResp.Timestamp.Format("15:04:05"))
	// }
	
	// Convert HTTP response to lightweight block format that preserves original hash
	blocks := make([]*types.Block, 0, len(blockResponses))
	for _, blockResp := range blockResponses {
		// Parse the original hash from hex string
		originalHashBytes, err := hex.DecodeString(blockResp.Hash)
		if err != nil {
			log.Printf("Wave Consensus: Failed to decode hash %s: %v", blockResp.Hash, err)
			continue
		}
		
		// Create a lightweight block structure that preserves the original hash
		block := &types.Block{
			Header: &types.BlockHeader{
				Version:   1,
				Timestamp: blockResp.Timestamp,
				Round:     blockResp.Round,
				Wave:      blockResp.Wave,
				Height:    blockResp.Height,
				Validator: types.Address(blockResp.Validator),
			},
			Body: &types.BlockBody{
				// Keep empty - we don't need transaction data for consensus voting
				Transactions: make([]*types.Transaction, 0),
				Receipts:     make([]*types.Receipt, 0),
				Events:       make([]*types.Event, 0),
			},
			OriginalHash: originalHashBytes, // Store the original hash
		}
		
		// log.Printf("✅ Wave Consensus: PRESERVED ORIGINAL HASH [%d]:", i+1)
		// log.Printf("   📦 Original Hash: %s", blockResp.Hash)
		// log.Printf("   📦 Preserved Hash: %x", originalHashBytes)
		// log.Printf("   ✅ Hash Match: %t", true) // Always true now since we preserve the original


		blocks = append(blocks, block)
	}
	
	return blocks
}

// getFallbackBlocks returns mock blocks when DAG sync is not available
func (rds *RemoteDAGSync) getFallbackBlocks() []*types.Block {
	log.Printf("Wave Consensus: Using fallback mock blocks")
	blocks := make([]*types.Block, 0)
	
	// Create a few mock blocks to demonstrate wave consensus
	for i := 0; i < 3; i++ {
		block := &types.Block{
			Header: &types.BlockHeader{
				Version:   1,
				Timestamp: time.Now().Add(-time.Duration(i)*time.Second),
				Round:     types.Round(time.Now().Unix() % 100), // Mock round
				Wave:      0,
				Height:    types.BlockNumber(i + 1),
				Validator: rds.validatorID,
			},
			Body: &types.BlockBody{
				Transactions: rds.generateSampleTransactions(1),
				Receipts:     make([]*types.Receipt, 0),
				Events:       make([]*types.Event, 0),
			},
		}
		blocks = append(blocks, block)
	}
	
	return blocks
}

// generateSampleTransactions creates sample transactions for blocks
func (rds *RemoteDAGSync) generateSampleTransactions(count int) []*types.Transaction {
	transactions := make([]*types.Transaction, count)
	
	for i := 0; i < count; i++ {
		tx := &types.Transaction{
			Nonce:     types.Nonce(time.Now().UnixNano() + int64(i)),
			From:      rds.validatorID,
			To:        types.Address(fmt.Sprintf("recipient_%d", i)),
			Value:     types.Value(100 + i),
			GasLimit:  21000,
			GasPrice:  1000000000,
			Data:      []byte(fmt.Sprintf("tx_data_%d", i)),
			Timestamp: time.Now(),
			State:     types.TransactionStatePending,
		}
		transactions[i] = tx
	}
	
	return transactions
}

func (rds *RemoteDAGSync) GetValidatorID() types.Address {
	return rds.validatorID
}

func main() {
	validatorID := flag.String("id", "wave-validator1", "Validator ID")
	dagAddr := flag.String("dag-addr", "localhost:4001", "DAG sync address to connect to")
	waveListenAddr := flag.String("wave-listen", "localhost:6001", "Wave consensus listen address")
	wavePeersStr := flag.String("wave-peers", "", "Comma-separated wave consensus peer addresses")
	// validatorsStr := flag.String("validators", "", "Comma-separated list of all validators for leader selection")
	waveDuration := flag.Duration("wave-duration", 1*time.Second, "Wave duration for consensus")
	flag.Parse()

	// Parse wave peers
	var wavePeers []string
	if *wavePeersStr != "" {
		wavePeers = strings.Split(*wavePeersStr, ",")
		for i, peer := range wavePeers {
			wavePeers[i] = strings.TrimSpace(peer)
		}
	}

	// Parse validators list - HARDCODED VALIDATOR SET
	// Always use the complete validator set regardless of CLI input
	validators := []types.Address{
		types.Address("validator1"),
		types.Address("validator2"),
		// types.Address("validator3"),  // Uncomment to add more validators
		// types.Address("validator4"),  // Uncomment to add more validators
	}
	
	log.Printf("🔧 HARDCODED: Using fixed validator set: %v", validators)
	log.Printf("🔧 HARDCODED: Total validators in network: %d", len(validators))
	
	// Original CLI parsing (commented out for reference)
	/*
	var validators []types.Address
	if *validatorsStr != "" {
		validatorStrs := strings.Split(*validatorsStr, ",")
		for _, validatorStr := range validatorStrs {
			validators = append(validators, types.Address(strings.TrimSpace(validatorStr)))
		}
	} else {
		// Default: single validator mode
		validators = []types.Address{types.Address(*validatorID)}
	}
	*/

	// Check if DAG sync is reachable
	conn, err := net.DialTimeout("tcp", *dagAddr, 2*time.Second)
	if err != nil {
		log.Printf("Warning: Cannot connect to DAG sync at %s: %v", *dagAddr, err)
		log.Printf("Running with mock DAG data...")
	} else {
		conn.Close()
		log.Printf("Connected to DAG sync at %s", *dagAddr)
	}

	// Create remote DAG sync interface
	remoteDAG := NewRemoteDAGSync(types.Address(*validatorID), *dagAddr)

	// Create wave consensus
	waveConsensus := consensus.NewWaveConsensus(
		types.Address(*validatorID),
		validators,
		remoteDAG, // Use remote DAG interface
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
		os.Exit(0)
	}()

	// Start wave consensus
	if err := waveConsensus.Start(); err != nil {
		log.Fatalf("Failed to start wave consensus: %v", err)
	}

	fmt.Printf("=== Wave Consensus Only Started ===\n")
	fmt.Printf("Validator: %s\n", *validatorID)
	fmt.Printf("Validators: %v\n", validators)
	fmt.Printf("DAG Source: %s\n", *dagAddr)
	fmt.Printf("Wave Listen: %s\n", *waveListenAddr)
	if len(wavePeers) > 0 {
		fmt.Printf("Wave Peers: %v\n", wavePeers)
	} else {
		fmt.Printf("Wave Peers: none (single validator mode)\n")
	}
	fmt.Printf("Wave Duration: %v\n", *waveDuration)
	fmt.Printf("===================================\n")

	// Status reporting
	go func() {
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-sigChan:
				return
			case <-ticker.C:
				waveStatus := waveConsensus.GetWaveStatus()
				
				leaderSymbol := "⚪"
				if waveStatus["is_current_leader"].(bool) {
					leaderSymbol = "👑"
				}
				
				fmt.Printf("\n=== Wave Status [%s] ===\n", time.Now().Format("15:04:05"))
				fmt.Printf("Wave: %d, Leader: %s %s, Finalized: %d, Peers: %d\n", 
					waveStatus["current_wave"], 
					waveStatus["current_leader"], 
					leaderSymbol,
					waveStatus["finalized_waves"], 
					waveStatus["connected_peers"])
				fmt.Printf("========================\n")
			}
		}
	}()
	
	select {} // Block forever
} 