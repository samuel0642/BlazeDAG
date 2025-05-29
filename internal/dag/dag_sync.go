package dag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// DAGSync handles DAG synchronization between validators
type DAGSync struct {
	validatorID   types.Address
	dag           *DAG
	currentRound  types.Round
	roundDuration time.Duration
	
	// Network configuration
	listenAddr    string
	peers         []string
	
	// Synchronization
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	
	// Round management
	roundTicker   *time.Ticker
	roundBlocks   map[types.Round][]*types.Block // blocks per round
	
	// Network layer
	listener      net.Listener
	connections   map[string]net.Conn
	connMu        sync.RWMutex
}

// DAGMessage represents a message for DAG synchronization
type DAGMessage struct {
	Type      string          `json:"type"`
	Block     *types.Block    `json:"block,omitempty"`
	Round     types.Round     `json:"round,omitempty"`
	Validator types.Address   `json:"validator"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewDAGSync creates a new DAG sync instance
func NewDAGSync(validatorID types.Address, listenAddr string, peers []string, roundDuration time.Duration) *DAGSync {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DAGSync{
		validatorID:   validatorID,
		dag:           NewDAG(),
		currentRound:  0,
		roundDuration: roundDuration,
		listenAddr:    listenAddr,
		peers:         peers,
		ctx:           ctx,
		cancel:        cancel,
		roundBlocks:   make(map[types.Round][]*types.Block),
		connections:   make(map[string]net.Conn),
	}
}

// Start starts the DAG sync process
func (ds *DAGSync) Start() error {
	log.Printf("DAG Sync [%s]: Starting DAG synchronization", ds.validatorID)
	
	// Start network listener
	if err := ds.startListener(); err != nil {
		return fmt.Errorf("failed to start listener: %v", err)
	}
	
	// Connect to peers
	go ds.connectToPeers()
	
	// Start round ticker
	ds.roundTicker = time.NewTicker(ds.roundDuration)
	go ds.runRounds()
	
	log.Printf("DAG Sync [%s]: Started at %s, connecting to %d peers", 
		ds.validatorID, ds.listenAddr, len(ds.peers))
	
	return nil
}

// Stop stops the DAG sync
func (ds *DAGSync) Stop() {
	log.Printf("DAG Sync [%s]: Stopping", ds.validatorID)
	ds.cancel()
	
	if ds.roundTicker != nil {
		ds.roundTicker.Stop()
	}
	
	if ds.listener != nil {
		ds.listener.Close()
	}
	
	ds.connMu.Lock()
	for _, conn := range ds.connections {
		conn.Close()
	}
	ds.connMu.Unlock()
}

// startListener starts the network listener
func (ds *DAGSync) startListener() error {
	listener, err := net.Listen("tcp", ds.listenAddr)
	if err != nil {
		return err
	}
	
	ds.listener = listener
	
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ds.ctx.Done():
					return
				default:
					log.Printf("DAG Sync [%s]: Accept error: %v", ds.validatorID, err)
					continue
				}
			}
			
			go ds.handleConnection(conn)
		}
	}()
	
	return nil
}

// connectToPeers connects to peer validators
func (ds *DAGSync) connectToPeers() {
	for _, peer := range ds.peers {
		go func(peerAddr string) {
			for {
				select {
				case <-ds.ctx.Done():
					return
				default:
					if err := ds.connectToPeer(peerAddr); err != nil {
						log.Printf("DAG Sync [%s]: Failed to connect to %s: %v", ds.validatorID, peerAddr, err)
						time.Sleep(2 * time.Second) // Retry after 2 seconds
						continue
					}
					return // Successfully connected
				}
			}
		}(peer)
	}
}

// connectToPeer connects to a specific peer
func (ds *DAGSync) connectToPeer(peerAddr string) error {
	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		return err
	}
	
	ds.connMu.Lock()
	ds.connections[peerAddr] = conn
	ds.connMu.Unlock()
	
	log.Printf("DAG Sync [%s]: Connected to peer %s", ds.validatorID, peerAddr)
	
	go ds.handleConnection(conn)
	return nil
}

// handleConnection handles a network connection
func (ds *DAGSync) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	decoder := json.NewDecoder(conn)
	for {
		select {
		case <-ds.ctx.Done():
			return
		default:
			var msg DAGMessage
			if err := decoder.Decode(&msg); err != nil {
				return // Connection closed or error
			}
			
			ds.handleDAGMessage(&msg)
		}
	}
}

// runRounds runs the DAG round process
func (ds *DAGSync) runRounds() {
	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ds.roundTicker.C:
			ds.advanceRound()
		}
	}
}

// advanceRound advances to the next DAG round
func (ds *DAGSync) advanceRound() {
	ds.mu.Lock()
	ds.currentRound++
	currentRound := ds.currentRound
	ds.mu.Unlock()
	
	log.Printf("DAG Sync [%s]: Advanced to round %d", ds.validatorID, currentRound)
	
	// Create block for this round
	go ds.createRoundBlock(currentRound)
}

// createRoundBlock creates a block for the current round
func (ds *DAGSync) createRoundBlock(round types.Round) {
	// Get references to blocks from previous round
	references := ds.getPreviousRoundReferences(round - 1)
	
	// Create transactions (simulate)
	transactions := ds.generateTransactions(5) // 5 transactions per block
	
	// Create the block
	block := &types.Block{
		Header: &types.BlockHeader{
			Version:    1,
			Timestamp:  time.Now(),
			Round:      round,
			Wave:       0, // DAG sync doesn't care about waves
			Height:     ds.dag.GetHeight() + 1,
			References: references,
			Validator:  ds.validatorID,
		},
		Body: &types.BlockBody{
			Transactions: transactions,
			Receipts:     make([]*types.Receipt, 0),
			Events:       make([]*types.Event, 0),
		},
	}
	
	// Add to local DAG
	if err := ds.dag.AddBlock(block); err != nil {
		log.Printf("DAG Sync [%s]: Failed to add block to DAG: %v", ds.validatorID, err)
		return
	}
	
	// Track block for this round
	ds.mu.Lock()
	if ds.roundBlocks[round] == nil {
		ds.roundBlocks[round] = make([]*types.Block, 0)
	}
	ds.roundBlocks[round] = append(ds.roundBlocks[round], block)
	ds.mu.Unlock()
	
	log.Printf("DAG Sync [%s]: Created block for round %d with %d references", 
		ds.validatorID, round, len(references))
	
	// Broadcast to other validators
	ds.broadcastBlock(block)
}

// getPreviousRoundReferences gets references to blocks from previous round
func (ds *DAGSync) getPreviousRoundReferences(prevRound types.Round) []*types.Reference {
	if prevRound <= 0 {
		return []*types.Reference{} // Genesis round
	}
	
	// Wait a bit to ensure we have received blocks from other validators for the previous round
	time.Sleep(100 * time.Millisecond)
	
	ds.mu.RLock()
	prevBlocks := ds.roundBlocks[prevRound]
	ds.mu.RUnlock()
	
	// Also get ALL blocks from the DAG that belong to the previous round
	// This ensures we reference blocks from other validators too
	allDAGBlocks := ds.dag.GetRecentBlocks(100) // Get recent blocks
	
	// Collect all blocks from the previous round (from all validators)
	allPrevRoundBlocks := make([]*types.Block, 0)
	
	// Add blocks we tracked locally for this round
	allPrevRoundBlocks = append(allPrevRoundBlocks, prevBlocks...)
	
	// Add any blocks from DAG that belong to previous round but we might have missed
	for _, block := range allDAGBlocks {
		if block.Header.Round == prevRound {
			// Check if we already have this block in our list
			found := false
			for _, existing := range allPrevRoundBlocks {
				if string(existing.ComputeHash()) == string(block.ComputeHash()) {
					found = true
					break
				}
			}
			if !found {
				allPrevRoundBlocks = append(allPrevRoundBlocks, block)
			}
		}
	}
	
	log.Printf("DAG Sync [%s]: Found %d blocks from round %d (from all validators)", 
		ds.validatorID, len(allPrevRoundBlocks), prevRound)
	
	references := make([]*types.Reference, 0)
	
	// Reference ALL blocks from previous round (from all validators)
	for _, block := range allPrevRoundBlocks {
		ref := &types.Reference{
			BlockHash: block.ComputeHash(),
			Round:     block.Header.Round,
			Wave:      block.Header.Wave,
			Type:      types.ReferenceTypeStandard,
		}
		references = append(references, ref)
	}
	
	// Log which validators' blocks we're referencing
	validatorCounts := make(map[types.Address]int)
	for _, block := range allPrevRoundBlocks {
		validatorCounts[block.Header.Validator]++
	}
	
	log.Printf("DAG Sync [%s]: Referencing blocks from validators: %v", ds.validatorID, validatorCounts)
	
	return references
}

// generateTransactions generates sample transactions
func (ds *DAGSync) generateTransactions(count int) []*types.Transaction {
	transactions := make([]*types.Transaction, count)
	
	for i := 0; i < count; i++ {
		tx := &types.Transaction{
			Nonce:     types.Nonce(time.Now().UnixNano() + int64(i)),
			From:      ds.validatorID,
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

// broadcastBlock broadcasts a block to all connected peers
func (ds *DAGSync) broadcastBlock(block *types.Block) {
	msg := &DAGMessage{
		Type:      "block",
		Block:     block,
		Round:     block.Header.Round,
		Validator: ds.validatorID,
		Timestamp: time.Now(),
	}
	
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("DAG Sync [%s]: Failed to marshal block: %v", ds.validatorID, err)
		return
	}
	
	ds.connMu.RLock()
	defer ds.connMu.RUnlock()
	
	broadcastCount := 0
	for peerAddr, conn := range ds.connections {
		if _, err := conn.Write(append(data, '\n')); err != nil {
			log.Printf("DAG Sync [%s]: Failed to send to %s: %v", ds.validatorID, peerAddr, err)
			continue
		}
		broadcastCount++
	}
	
	log.Printf("DAG Sync [%s]: Broadcasted block (round %d) to %d peers", 
		ds.validatorID, block.Header.Round, broadcastCount)
}

// handleDAGMessage handles received DAG messages
func (ds *DAGSync) handleDAGMessage(msg *DAGMessage) {
	switch msg.Type {
	case "block":
		ds.handleReceivedBlock(msg.Block, msg.Validator)
	default:
		log.Printf("DAG Sync [%s]: Unknown message type: %s", ds.validatorID, msg.Type)
	}
}

// handleReceivedBlock handles a received block from another validator
func (ds *DAGSync) handleReceivedBlock(block *types.Block, fromValidator types.Address) {
	if block == nil {
		return
	}
	
	// Check if we already have this block
	if _, err := ds.dag.GetBlock(block.ComputeHash()); err == nil {
		return // Already have this block
	}
	
	// Add to local DAG
	if err := ds.dag.AddBlock(block); err != nil {
		log.Printf("DAG Sync [%s]: Failed to add received block: %v", ds.validatorID, err)
		return
	}
	
	// Track block for this round
	ds.mu.Lock()
	round := block.Header.Round
	if ds.roundBlocks[round] == nil {
		ds.roundBlocks[round] = make([]*types.Block, 0)
	}
	ds.roundBlocks[round] = append(ds.roundBlocks[round], block)
	ds.mu.Unlock()
	
	log.Printf("DAG Sync [%s]: Received and added block from %s (round %d)", 
		ds.validatorID, fromValidator, block.Header.Round)
}

// GetDAGStatus returns the current DAG status
func (ds *DAGSync) GetDAGStatus() map[string]interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	// Count blocks per round
	roundCounts := make(map[string]int)
	for round, blocks := range ds.roundBlocks {
		roundCounts[fmt.Sprintf("round_%d", round)] = len(blocks)
	}
	
	ds.connMu.RLock()
	connectedPeers := len(ds.connections)
	ds.connMu.RUnlock()
	
	return map[string]interface{}{
		"validator_id":     ds.validatorID,
		"current_round":    ds.currentRound,
		"total_blocks":     ds.dag.GetBlockCount(),
		"dag_height":       ds.dag.GetHeight(),
		"connected_peers":  connectedPeers,
		"blocks_per_round": roundCounts,
		"listen_addr":      ds.listenAddr,
	}
}

// GetCurrentRound returns the current round
func (ds *DAGSync) GetCurrentRound() types.Round {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.currentRound
} 