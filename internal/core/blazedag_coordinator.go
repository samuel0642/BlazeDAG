package core

import (
	"log"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/dag"
	"github.com/CrossDAG/BlazeDAG/internal/storage"
	"github.com/CrossDAG/BlazeDAG/internal/transaction"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// BlazeDagCoordinator coordinates independent DAG transport and wave consensus
type BlazeDagCoordinator struct {
	config        *Config
	storage       *storage.Storage
	dag           *dag.DAG
	dagTransport  *dag.DAGTransport
	waveConsensus *WaveConsensus
	
	// Transaction handling
	transactionPool *transaction.SimplePool
	
	// Timing configuration
	dagRoundDuration  time.Duration
	waveTimeout       time.Duration
	
	// Validator information
	validatorID types.Address
	validators  []types.Address
}

// NewBlazeDagCoordinator creates a new BlazeDAG coordinator
func NewBlazeDagCoordinator(config *Config, storage *storage.Storage) *BlazeDagCoordinator {
	// Create DAG
	dagInstance := dag.NewDAG()
	
	// Configure timing - these are INDEPENDENT
	dagRoundDuration := 500 * time.Millisecond  // DAG rounds every 500ms
	waveTimeout := 3 * time.Second              // Wave timeout after 3 seconds
	
	// Create independent DAG transport
	dagTransport := dag.NewDAGTransport(dagInstance, dagRoundDuration)
	
	// Set up validators
	validators := []types.Address{
		types.Address("validator1"),
		types.Address("validator2"),
		types.Address("validator3"),
	}
	
	// Create independent wave consensus
	waveConsensus := NewWaveConsensus(
		dagTransport,
		config.NodeID,
		validators,
		waveTimeout,
	)
	
	// Create transaction pool
	transactionPool := transaction.NewSimplePool()
	
	return &BlazeDagCoordinator{
		config:            config,
		storage:           storage,
		dag:               dagInstance,
		dagTransport:      dagTransport,
		waveConsensus:     waveConsensus,
		transactionPool:   transactionPool,
		dagRoundDuration:  dagRoundDuration,
		waveTimeout:       waveTimeout,
		validatorID:       config.NodeID,
		validators:        validators,
	}
}

// Start starts both DAG transport and wave consensus independently
func (bdc *BlazeDagCoordinator) Start() error {
	log.Printf("BlazeDAG Coordinator: Starting independent DAG transport and wave consensus")
	
	// Start DAG transport (runs at network speed)
	bdc.dagTransport.Start()
	
	// Start wave consensus (runs independently)
	bdc.waveConsensus.Start()
	
	// Start transaction generation (simulate network activity)
	go bdc.generateTransactions()
	
	// Start block creation (independent of waves)
	go bdc.createDAGBlocks()
	
	log.Printf("BlazeDAG Coordinator: Successfully started - DAG rounds and waves operating independently")
	return nil
}

// Stop stops both layers
func (bdc *BlazeDagCoordinator) Stop() {
	log.Printf("BlazeDAG Coordinator: Stopping independent layers")
	bdc.dagTransport.Stop()
	bdc.waveConsensus.Stop()
}

// generateTransactions simulates incoming transactions
func (bdc *BlazeDagCoordinator) generateTransactions() {
	ticker := time.NewTicker(200 * time.Millisecond) // Generate transactions every 200ms
	defer ticker.Stop()
	
	txCounter := 0
	for range ticker.C {
		tx := &types.Transaction{
			Nonce:     types.Nonce(txCounter),
			From:      bdc.validatorID,
			To:        types.Address("recipient"),
			Value:     types.Value(100),
			GasLimit:  21000,
			GasPrice:  1000000000,
			Data:      []byte("transaction_data"),
			Timestamp: time.Now(),
			State:     types.TransactionStatePending,
		}
		
		bdc.transactionPool.Add(tx)
		txCounter++
		
		if txCounter%10 == 0 {
			log.Printf("BlazeDAG Coordinator: Generated %d transactions", txCounter)
		}
	}
}

// createDAGBlocks creates DAG blocks independently of consensus waves
func (bdc *BlazeDagCoordinator) createDAGBlocks() {
	ticker := time.NewTicker(800 * time.Millisecond) // Create blocks every 800ms
	defer ticker.Stop()
	
	blockCounter := 0
	for range ticker.C {
		// Get pending transactions
		transactions := bdc.transactionPool.GetPending(10) // Get up to 10 transactions
		
		if len(transactions) == 0 {
			continue
		}
		
		// Create DAG block (note: wave is set to 0, will be updated by consensus)
		block := bdc.dagTransport.CreateDAGBlock(bdc.validatorID, transactions)
		
		// Broadcast to DAG transport
		bdc.dagTransport.BroadcastBlock(block)
		
		// Mark transactions as included
		for _, tx := range transactions {
			tx.State = types.TransactionStateIncluded
		}
		
		blockCounter++
		log.Printf("BlazeDAG Coordinator: Created DAG block %d with %d transactions at round %d", 
			blockCounter, len(transactions), block.Header.Round)
	}
}

// GetDAGStatus returns status of the DAG transport layer
func (bdc *BlazeDagCoordinator) GetDAGStatus() map[string]interface{} {
	return map[string]interface{}{
		"current_dag_round": bdc.dagTransport.GetCurrentDAGRound(),
		"total_blocks":      bdc.dag.GetBlockCount(),
		"dag_height":        bdc.dag.GetHeight(),
		"uncommitted_blocks": len(bdc.dagTransport.GetUncommittedBlocks()),
	}
}

// GetConsensusStatus returns status of the wave consensus layer
func (bdc *BlazeDagCoordinator) GetConsensusStatus() map[string]interface{} {
	committedBlocks := bdc.waveConsensus.GetCommittedBlocks()
	return map[string]interface{}{
		"current_wave":     bdc.waveConsensus.GetCurrentWave(),
		"committed_blocks": len(committedBlocks),
		"is_leader":        bdc.isCurrentWaveLeader(),
	}
}

// isCurrentWaveLeader checks if this validator is the leader for the current wave
func (bdc *BlazeDagCoordinator) isCurrentWaveLeader() bool {
	currentWave := bdc.waveConsensus.GetCurrentWave()
	leaderIndex := int(currentWave) % len(bdc.validators)
	return bdc.validators[leaderIndex] == bdc.validatorID
}

// GetSystemStatus returns the overall system status showing independence
func (bdc *BlazeDagCoordinator) GetSystemStatus() map[string]interface{} {
	dagStatus := bdc.GetDAGStatus()
	consensusStatus := bdc.GetConsensusStatus()
	
	return map[string]interface{}{
		"validator_id":        bdc.validatorID,
		"dag_transport":       dagStatus,
		"wave_consensus":      consensusStatus,
		"pending_transactions": bdc.transactionPool.GetPendingCount(),
		"independence_demo": map[string]interface{}{
			"dag_rounds_independent":    true,
			"waves_independent":         true,
			"no_synchronization":        true,
			"network_speed_operation":   true,
		},
	}
}

// GetCommittedTransactions returns transactions that have been committed by consensus
func (bdc *BlazeDagCoordinator) GetCommittedTransactions() []*types.Transaction {
	committedBlocks := bdc.waveConsensus.GetCommittedBlocks()
	var committedTxs []*types.Transaction
	
	for _, block := range committedBlocks {
		for _, tx := range block.Body.Transactions {
			tx.State = types.TransactionStateCommitted
			committedTxs = append(committedTxs, tx)
		}
	}
	
	return committedTxs
} 