package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/api"
	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/dag"
	"github.com/CrossDAG/BlazeDAG/internal/evm"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// IntegratedNode combines DAG sync, wave consensus, and EVM compatibility
type IntegratedNode struct {
	validatorID     types.Address
	dagSync         *dag.DAGSync
	waveConsensus   *consensus.WaveConsensus
	evmExecutor     *evm.EVMExecutor
	evmKeystore     *evm.Keystore
	evmRPCServer    *api.EVMRPCServer
	evmHTTPServer   *http.Server
	
	// Configuration
	chainID         *big.Int
	evmRPCAddr      string
	
	// Synchronization
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	
	// EVM transaction integration
	evmTxChannel    chan *evm.Transaction
	blockUpdateChan chan *types.Block
}

func main() {
	// Command line flags
	validatorID := flag.String("id", "validator1", "Validator ID")
	dagListenAddr := flag.String("dag-listen", "0.0.0.0:4001", "DAG sync listen address")
	waveListenAddr := flag.String("wave-listen", "0.0.0.0:6001", "Wave consensus listen address")
	dagPeersStr := flag.String("dag-peers", "", "Comma-separated DAG sync peer addresses")
	wavePeersStr := flag.String("wave-peers", "", "Comma-separated wave consensus peer addresses")
	roundDuration := flag.Duration("round-duration", 2*time.Second, "Round duration for DAG sync")
	waveDuration := flag.Duration("wave-duration", 3*time.Second, "Wave duration for consensus")
	
	// EVM configuration
	evmRPCAddr := flag.String("evm-rpc", "0.0.0.0:8545", "EVM JSON-RPC server address")
	chainID := flag.Int64("chain-id", 1337, "Chain ID for EVM")
	createAccounts := flag.Int("create-accounts", 5, "Number of test accounts to create")
	initialBalance := flag.String("initial-balance", "1000000000000000000000", "Initial balance for test accounts (in wei)")
	
	flag.Parse()

	fmt.Printf("🚀 Starting BlazeDAG Integrated Node\n")
	fmt.Printf("Validator ID: %s\n", *validatorID)
	fmt.Printf("DAG Listen: %s\n", *dagListenAddr)
	fmt.Printf("Wave Listen: %s\n", *waveListenAddr)
	fmt.Printf("EVM RPC: %s\n", *evmRPCAddr)
	fmt.Printf("Chain ID: %d\n", *chainID)

	// Parse peers
	var dagPeers []string
	if *dagPeersStr != "" {
		dagPeers = strings.Split(*dagPeersStr, ",")
		for i, peer := range dagPeers {
			dagPeers[i] = strings.TrimSpace(peer)
		}
	}

	var wavePeers []string
	if *wavePeersStr != "" {
		wavePeers = strings.Split(*wavePeersStr, ",")
		for i, peer := range wavePeers {
			wavePeers[i] = strings.TrimSpace(peer)
		}
	}

	// Create integrated node
	node, err := NewIntegratedNode(
		types.Address(*validatorID),
		*dagListenAddr,
		*waveListenAddr,
		*evmRPCAddr,
		dagPeers,
		wavePeers,
		*roundDuration,
		*waveDuration,
		*chainID,
		*createAccounts,
		*initialBalance,
	)
	if err != nil {
		log.Fatalf("Failed to create integrated node: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n=== Shutting Down Integrated Node ===")
		node.Stop()
		os.Exit(0)
	}()

	// Start the integrated node
	if err := node.Start(); err != nil {
		log.Fatalf("Failed to start integrated node: %v", err)
	}

	fmt.Printf("=== BlazeDAG Integrated Node Started ===\n")
	fmt.Printf("🔥 All services running:\n")
	fmt.Printf("  📡 DAG Sync: %s\n", *dagListenAddr)
	fmt.Printf("  🌊 Wave Consensus: %s\n", *waveListenAddr)
	fmt.Printf("  ⚡ EVM RPC: %s\n", *evmRPCAddr)
	fmt.Printf("  🔗 Chain ID: %d\n", *chainID)
	fmt.Printf("=======================================\n")

	// Block forever
	select {}
}

// NewIntegratedNode creates a new integrated node
func NewIntegratedNode(
	validatorID types.Address,
	dagListenAddr,
	waveListenAddr,
	evmRPCAddr string,
	dagPeers,
	wavePeers []string,
	roundDuration,
	waveDuration time.Duration,
	chainID int64,
	createAccounts int,
	initialBalance string,
) (*IntegratedNode, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create EVM components
	evmState := evm.NewState()
	keystore := evm.NewKeystore()
	
	// Parse initial balance
	balance, ok := new(big.Int).SetString(initialBalance, 10)
	if !ok {
		return nil, fmt.Errorf("invalid initial balance: %s", initialBalance)
	}
	
	// Create test accounts
	testAccounts, err := evm.CreateTestAccounts(evmState, createAccounts, balance)
	if err != nil {
		return nil, fmt.Errorf("failed to create test accounts: %v", err)
	}
	
	// Add accounts to keystore
	for _, account := range testAccounts {
		keystore.AddKey(account)
	}
	
	fmt.Printf("\n📝 Created EVM Test Accounts:\n")
	for i, account := range testAccounts {
		fmt.Printf("Account %d:\n", i+1)
		fmt.Printf("  Address: %s\n", account.GetAddressHex())
		fmt.Printf("  Private Key: %s\n", account.GetPrivateKeyHex())
		fmt.Printf("  Balance: %s ETH\n", weiToEth(balance).String())
	}
	
	// Create EVM executor
	gasPrice := big.NewInt(1000000000) // 1 gwei
	evmExecutor := evm.NewEVMExecutor(evmState, gasPrice)
	
	// Create EVM RPC server
	chainIDBig := big.NewInt(chainID)
	evmRPCServer := api.NewEVMRPCServer(evmExecutor, keystore, chainIDBig)
	
	// Create integrated DAG sync (with EVM integration)
	dagSync := NewIntegratedDAGSync(
		validatorID,
		dagListenAddr,
		dagPeers,
		roundDuration,
		evmExecutor, // Pass EVM executor for transaction processing
	)
	
	// Create list of all validators for wave consensus
	allValidators := []types.Address{validatorID}
	for _, peer := range wavePeers {
		// Extract validator ID from peer address (simplified)
		peerID := types.Address(fmt.Sprintf("validator_%s", strings.Split(peer, ":")[0]))
		allValidators = append(allValidators, peerID)
	}
	
	// Create wave consensus
	waveConsensus := consensus.NewWaveConsensus(
		validatorID,
		allValidators,
		dagSync, // Pass DAG sync as DAGReader
		waveListenAddr,
		wavePeers,
		waveDuration,
	)
	
	node := &IntegratedNode{
		validatorID:     validatorID,
		dagSync:         dagSync,
		waveConsensus:   waveConsensus,
		evmExecutor:     evmExecutor,
		evmKeystore:     keystore,
		evmRPCServer:    evmRPCServer,
		chainID:         chainIDBig,
		evmRPCAddr:      evmRPCAddr,
		ctx:             ctx,
		cancel:          cancel,
		evmTxChannel:    make(chan *evm.Transaction, 1000),
		blockUpdateChan: make(chan *types.Block, 100),
	}
	
	return node, nil
}

// Start starts all components of the integrated node
func (n *IntegratedNode) Start() error {
	log.Printf("🚀 Starting integrated node for validator: %s", n.validatorID)
	
	// Start DAG sync first
	if err := n.dagSync.Start(); err != nil {
		return fmt.Errorf("failed to start DAG sync: %v", err)
	}
	
	// Wait a bit for DAG sync to initialize
	time.Sleep(1 * time.Second)
	
	// Start wave consensus
	if err := n.waveConsensus.Start(); err != nil {
		return fmt.Errorf("failed to start wave consensus: %v", err)
	}
	
	// Start EVM HTTP server
	if err := n.startEVMRPCServer(); err != nil {
		return fmt.Errorf("failed to start EVM RPC server: %v", err)
	}
	
	// Start EVM transaction processor
	n.evmRPCServer.StartTransactionProcessor()
	
	// Start integration services
	go n.processEVMTransactions()
	go n.processBlockUpdates()
	go n.connectEVMToDAG()
	go n.statusReporter()
	
	return nil
}

// Stop stops all components
func (n *IntegratedNode) Stop() {
	log.Printf("🛑 Stopping integrated node: %s", n.validatorID)
	n.cancel()
	
	if n.evmHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		n.evmHTTPServer.Shutdown(ctx)
	}
	
	if n.waveConsensus != nil {
		n.waveConsensus.Stop()
	}
	
	if n.dagSync != nil {
		n.dagSync.Stop()
	}
}

// startEVMRPCServer starts the EVM JSON-RPC server
func (n *IntegratedNode) startEVMRPCServer() error {
	n.evmHTTPServer = &http.Server{
		Addr:    n.evmRPCAddr,
		Handler: n.evmRPCServer,
	}
	
	go func() {
		log.Printf("⚡ Starting EVM JSON-RPC server on %s", n.evmRPCAddr)
		if err := n.evmHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("EVM RPC server error: %v", err)
		}
	}()
	
	return nil
}

// processEVMTransactions processes EVM transactions and includes them in DAG blocks
func (n *IntegratedNode) processEVMTransactions() {
	for {
		select {
		case <-n.ctx.Done():
			return
		case evmTx := <-n.evmTxChannel:
			// Convert EVM transaction to BlazeDAG transaction and add to pending pool
			blazeTx := n.convertEVMToBlazeTransaction(evmTx)
			n.dagSync.AddPendingTransaction(blazeTx)
			
			log.Printf("⚡ EVM transaction integrated into DAG: %s", blazeTx.ComputeHash())
		}
	}
}

// processBlockUpdates processes new blocks and updates EVM state
func (n *IntegratedNode) processBlockUpdates() {
	for {
		select {
		case <-n.ctx.Done():
			return
		case block := <-n.blockUpdateChan:
			// Update EVM RPC server with new block number
			n.evmRPCServer.UpdateBlockNumber(big.NewInt(int64(block.Header.Height)))
			
			// Process EVM transactions in the block
			for _, tx := range block.Body.Transactions {
				if n.isEVMTransaction(tx) {
					n.processEVMTransactionInBlock(tx)
				}
			}
			
			log.Printf("🔄 Block %d processed, EVM state updated", block.Header.Height)
		}
	}
}

// statusReporter reports the status of all components
func (n *IntegratedNode) statusReporter() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			dagStatus := n.dagSync.GetDAGStatus()
			waveStatus := n.waveConsensus.GetWaveStatus()
			
			fmt.Printf("\n=== Integrated Node Status [%s] ===\n", time.Now().Format("15:04:05"))
			fmt.Printf("🔥 Validator: %s\n", n.validatorID)
			fmt.Printf("📡 DAG: Round %d, %d blocks, %d peers\n", 
				dagStatus["current_round"], dagStatus["total_blocks"], dagStatus["connected_peers"])
			fmt.Printf("🌊 Wave: Wave %d, %d finalized, %d peers\n", 
				waveStatus["current_wave"], waveStatus["finalized_waves"], waveStatus["connected_peers"])
			fmt.Printf("⚡ EVM: Chain ID %s, RPC on %s\n", n.chainID.String(), n.evmRPCAddr)
			fmt.Printf("================================\n")
		}
	}
}

// connectEVMToDAG connects EVM RPC server pending transactions to DAG
func (n *IntegratedNode) connectEVMToDAG() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			// Get pending EVM transactions
			pendingTxs := n.evmRPCServer.GetPendingTransactions()
			
			// Add each EVM transaction to DAG pending pool
			for _, evmTx := range pendingTxs {
				blazeTx := n.convertEVMToBlazeTransaction(evmTx)
				n.dagSync.AddPendingTransaction(blazeTx)
				
				log.Printf("⚡ EVM transaction added to DAG pending pool: %s", blazeTx.ComputeHash())
			}
		}
	}
}

// Helper functions
func (n *IntegratedNode) convertEVMToBlazeTransaction(evmTx *evm.Transaction) *types.Transaction {
	// Handle contract creation (nil To)
	var toAddr types.Address
	if evmTx.To != nil {
		toAddr = types.Address(evmTx.To.Hex())
	} else {
		// Contract creation - generate contract address
		contractAddr := evm.CreateContractAddress(evmTx.From, evmTx.Nonce)
		toAddr = types.Address(contractAddr.Hex())
	}
	
	return &types.Transaction{
		Nonce:     types.Nonce(evmTx.Nonce),
		From:      types.Address(evmTx.From.Hex()),
		To:        toAddr,
		Value:     types.Value(evmTx.Value.Uint64()),
		GasLimit:  evmTx.GasLimit,
		GasPrice:  evmTx.GasPrice.Uint64(),
		Data:      evmTx.Data,
		Timestamp: evmTx.Timestamp,
		State:     types.TransactionStatePending,
	}
}

func (n *IntegratedNode) isEVMTransaction(tx *types.Transaction) bool {
	// Check if transaction has EVM-specific data or is a contract interaction
	return len(tx.Data) > 0 || strings.HasPrefix(string(tx.To), "0x")
}

func (n *IntegratedNode) processEVMTransactionInBlock(tx *types.Transaction) {
	// This would process the transaction through the EVM when it's included in a finalized block
	log.Printf("⚡ Processing EVM transaction in finalized block: %s", tx.ComputeHash())
}

// weiToEth converts wei to ETH
func weiToEth(wei *big.Int) *big.Float {
	eth := new(big.Float).SetInt(wei)
	eth.Quo(eth, big.NewFloat(1e18))
	return eth
}

// NewIntegratedDAGSync creates a DAG sync with EVM integration
func NewIntegratedDAGSync(validatorID types.Address, listenAddr string, peers []string, roundDuration time.Duration, evmExecutor *evm.EVMExecutor) *dag.DAGSync {
	// For now, use the existing DAG sync - we'll enhance it to include EVM transactions
	return dag.NewDAGSync(validatorID, listenAddr, peers, roundDuration)
} 