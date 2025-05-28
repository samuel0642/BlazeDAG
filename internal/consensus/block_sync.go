package consensus

import (
	"bytes"
	"encoding/gob"
	"log"
	"net"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// BlockSynchronizer handles periodic synchronization of blocks between validators
type BlockSynchronizer struct {
	consensusEngine *ConsensusEngine
	dag             *core.DAG
	logger          *log.Logger
	stopChan        chan struct{}
	mu              sync.RWMutex
	running         bool
	syncInterval    time.Duration
}

// NewBlockSynchronizer creates a new block synchronizer
func NewBlockSynchronizer(engine *ConsensusEngine, syncInterval time.Duration) *BlockSynchronizer {
	return &BlockSynchronizer{
		consensusEngine: engine,
		dag:             core.GetDAG(), // Use singleton DAG
		logger:          log.New(log.Writer(), "[BlockSync] ", log.LstdFlags),
		stopChan:        make(chan struct{}),
		syncInterval:    syncInterval,
	}
}

// Start starts the block synchronizer
func (bs *BlockSynchronizer) Start() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.running {
		return
	}

	bs.running = true
	go bs.syncLoop()
	bs.logger.Printf("Block synchronizer started with interval %s", bs.syncInterval)
}

// Stop stops the block synchronizer
func (bs *BlockSynchronizer) Stop() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if !bs.running {
		return
	}

	close(bs.stopChan)
	bs.running = false
	bs.logger.Printf("Block synchronizer stopped")
}

// syncLoop performs periodic block synchronization
func (bs *BlockSynchronizer) syncLoop() {
	ticker := time.NewTicker(bs.syncInterval)
	defer ticker.Stop()

	// Initial sync after starting
	bs.syncWithPeers()

	for {
		select {
		case <-ticker.C:
			bs.syncWithPeers()
		case <-bs.stopChan:
			return
		}
	}
}

// syncWithPeers synchronizes blocks with all other validators
func (bs *BlockSynchronizer) syncWithPeers() {
	bs.logger.Printf("Starting block synchronization with peers")

	// Get all validators except ourselves
	ourID := bs.consensusEngine.nodeID
	validators := bs.consensusEngine.GetValidators()

	// Get our latest blocks
	ourBlocks := bs.dag.GetRecentBlocks(50)

	// Log our blocks for debugging
	bs.logger.Printf("Our local blocks (%d):", len(ourBlocks))
	for i, block := range ourBlocks {
		if i < 5 { // Only log a few blocks
			bs.logger.Printf("  Block[%d]: Hash=%x, Wave=%d, Validator=%s",
				i, block.ComputeHash(), block.Header.Wave, block.Header.Validator)
		}
	}

	// Create a map of our block hashes for quick lookup
	ourBlockHashes := make(map[string]bool)
	for _, block := range ourBlocks {
		ourBlockHashes[string(block.ComputeHash())] = true
	}

	// Sync with each validator
	var wg sync.WaitGroup
	for _, validator := range validators {
		if validator == ourID {
			continue // Skip ourselves
		}

		wg.Add(1)
		go func(validatorID types.Address) {
			defer wg.Done()
			bs.syncWithValidator(validatorID, ourBlockHashes)
		}(validator)
	}

	wg.Wait()
	bs.logger.Printf("Block synchronization completed")
}

// syncWithValidator synchronizes blocks with a specific validator
func (bs *BlockSynchronizer) syncWithValidator(validatorID types.Address, ourBlockHashes map[string]bool) {
	// Get validator address
	validatorAddr := bs.getValidatorSyncAddress(validatorID)
	if validatorAddr == "" {
		bs.logger.Printf("No sync address found for validator %s", validatorID)
		return
	}

	bs.logger.Printf("Syncing with validator %s at %s", validatorID, validatorAddr)

	// Connect to the validator
	conn, err := net.Dial("tcp", validatorAddr)
	if err != nil {
		bs.logger.Printf("Failed to connect to validator %s: %v", validatorID, err)
		return
	}
	defer conn.Close()

	// Send sync request
	syncReq := &SyncRequest{
		RequestingNode: bs.consensusEngine.nodeID,
		KnownBlocks:    make([][]byte, 0, len(ourBlockHashes)),
	}

	// Add our block hashes to the request
	for blockHash := range ourBlockHashes {
		syncReq.KnownBlocks = append(syncReq.KnownBlocks, []byte(blockHash))
	}

	// Send the request
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(syncReq); err != nil {
		bs.logger.Printf("Failed to encode sync request: %v", err)
		return
	}

	if _, err := conn.Write(buf.Bytes()); err != nil {
		bs.logger.Printf("Failed to send sync request: %v", err)
		return
	}

	// Read the response
	resp := make([]byte, 4096*10) // Allow for larger responses
	n, err := conn.Read(resp)
	if err != nil {
		bs.logger.Printf("Failed to read sync response: %v", err)
		return
	}

	// Decode the response
	var syncResp SyncResponse
	decoder := gob.NewDecoder(bytes.NewReader(resp[:n]))
	if err := decoder.Decode(&syncResp); err != nil {
		bs.logger.Printf("Failed to decode sync response: %v", err)
		return
	}

	// Process the missing blocks
	bs.logger.Printf("Received %d blocks from validator %s", len(syncResp.Blocks), validatorID)

	for i, block := range syncResp.Blocks {
		// Skip blocks we already have
		blockHash := block.ComputeHash()
		if ourBlockHashes[string(blockHash)] {
			bs.logger.Printf("Skipping block %d/%d - already have: Hash=%x, Wave=%d, Validator=%s",
				i+1, len(syncResp.Blocks), blockHash, block.Header.Wave, block.Header.Validator)
			continue
		}

		bs.logger.Printf("Processing new block %d/%d: Hash=%x, Wave=%d, Validator=%s",
			i+1, len(syncResp.Blocks), blockHash, block.Header.Wave, block.Header.Validator)

		// Add the block to our DAG
		if err := bs.dag.AddBlock(block); err != nil {
			if err.Error() == "block already exists" {
				bs.logger.Printf("Block %x already exists in DAG", blockHash)
			} else {
				bs.logger.Printf("Failed to add block %x to DAG: %v", blockHash, err)
			}
		} else {
			bs.logger.Printf("Successfully added block %d/%d to DAG: Hash=%x, Wave=%d, Validator=%s",
				i+1, len(syncResp.Blocks), blockHash, block.Header.Wave, block.Header.Validator)
		}
	}

	bs.logger.Printf("Sync with validator %s completed", validatorID)
}

// getValidatorSyncAddress returns the sync address for a validator
func (bs *BlockSynchronizer) getValidatorSyncAddress(validator types.Address) string {
	// Use the same IP addresses as the consensus engine but on a different port for sync
	switch string(validator) {
	case "validator1":
		return "54.183.204.244:4000" // Sync port for validator1
	case "validator2":
		return "52.53.192.236:4000" // Sync port for validator2
	default:
		return ""
	}
}

// HandleSyncRequest handles incoming block sync requests
func (bs *BlockSynchronizer) HandleSyncRequest(req *SyncRequest, conn net.Conn) {
	bs.logger.Printf("Received sync request from %s with %d known blocks",
		req.RequestingNode, len(req.KnownBlocks))

	// Create a set of known block hashes for quick lookup
	knownBlocks := make(map[string]bool)
	for _, blockHash := range req.KnownBlocks {
		knownBlocks[string(blockHash)] = true
	}

	// Get our blocks to send
	ourBlocks := bs.dag.GetRecentBlocks(50)

	// Create response with blocks the requester doesn't have
	resp := SyncResponse{
		RespondingNode: bs.consensusEngine.nodeID,
		Blocks:         make([]*types.Block, 0),
	}

	for _, block := range ourBlocks {
		blockHash := block.ComputeHash()
		if !knownBlocks[string(blockHash)] {
			resp.Blocks = append(resp.Blocks, block)
			bs.logger.Printf("Adding block to sync response: Hash=%x, Wave=%d, Validator=%s",
				blockHash, block.Header.Wave, block.Header.Validator)
		}
	}

	bs.logger.Printf("Sending %d blocks to %s", len(resp.Blocks), req.RequestingNode)

	// Send the response
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(resp); err != nil {
		bs.logger.Printf("Failed to encode sync response: %v", err)
		return
	}

	if _, err := conn.Write(buf.Bytes()); err != nil {
		bs.logger.Printf("Failed to send sync response: %v", err)
		return
	}

	bs.logger.Printf("Sync response sent to %s", req.RequestingNode)
}

// SyncRequest represents a request for block synchronization
type SyncRequest struct {
	RequestingNode types.Address
	KnownBlocks    [][]byte // Hashes of blocks the requesting node knows
}

// SyncResponse represents a response to a block synchronization request
type SyncResponse struct {
	RespondingNode types.Address
	Blocks         []*types.Block // Blocks the requester doesn't have
}

// RegisterTypes registers the types with gob for encoding/decoding
func RegisterSyncTypes() {
	gob.Register(&SyncRequest{})
	gob.Register(&SyncResponse{})
	gob.Register(&types.Block{})
	gob.Register(&types.BlockHeader{})
	gob.Register(&types.BlockBody{})
	gob.Register(&types.Transaction{})
}
