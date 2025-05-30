package consensus

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

// DAGReader interface for reading blocks from DAG
type DAGReader interface {
	GetRecentBlocks(count int) []*types.Block
	GetValidatorID() types.Address
}

// WaveConsensus handles wave-based consensus on blocks from DAG sync
type WaveConsensus struct {
	validatorID   types.Address
	dagReader     DAGReader     // Interface to read blocks from DAG
	currentWave   types.Wave
	waveDuration  time.Duration
	
	// Network configuration  
	listenAddr    string
	peers         []string
	
	// Synchronization
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	
	// Wave management
	waveTicker    *time.Ticker
	waveBlocks    map[types.Wave][]*types.Block // blocks being processed in each wave
	waveVotes     map[types.Wave]map[string]*WaveVote // votes per wave
	finalizedWaves map[types.Wave]bool // finalized waves
	
	// Network layer
	listener      net.Listener
	connections   map[string]net.Conn
	connMu        sync.RWMutex
}

// WaveVote represents a vote in wave consensus
type WaveVote struct {
	Wave      types.Wave    `json:"wave"`
	BlockHash []byte        `json:"block_hash"`
	Voter     types.Address `json:"voter"`
	VoteType  string        `json:"vote_type"` // "commit", "abort"
	Timestamp time.Time     `json:"timestamp"`
}

// WaveMessage represents a message for wave consensus
type WaveMessage struct {
	Type      string           `json:"type"`
	Vote      *WaveVote        `json:"vote,omitempty"`
	Wave      types.Wave       `json:"wave,omitempty"`
	Blocks    []*types.Block   `json:"blocks,omitempty"`
	Validator types.Address    `json:"validator"`
	Timestamp time.Time        `json:"timestamp"`
}

// NewWaveConsensus creates a new wave consensus instance
func NewWaveConsensus(validatorID types.Address, dagReader DAGReader, listenAddr string, peers []string, waveDuration time.Duration) *WaveConsensus {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WaveConsensus{
		validatorID:    validatorID,
		dagReader:      dagReader,
		currentWave:    0,
		waveDuration:   waveDuration,
		listenAddr:     listenAddr,
		peers:          peers,
		ctx:            ctx,
		cancel:         cancel,
		waveBlocks:     make(map[types.Wave][]*types.Block),
		waveVotes:      make(map[types.Wave]map[string]*WaveVote),
		finalizedWaves: make(map[types.Wave]bool),
		connections:    make(map[string]net.Conn),
	}
}

// Start starts the wave consensus process
func (wc *WaveConsensus) Start() error {
	log.Printf("Wave Consensus [%s]: Starting wave consensus", wc.validatorID)
	
	// Start network listener
	if err := wc.startListener(); err != nil {
		return fmt.Errorf("failed to start listener: %v", err)
	}
	
	// Connect to peers
	go wc.connectToPeers()
	
	// Start wave ticker
	wc.waveTicker = time.NewTicker(wc.waveDuration)
	go wc.runWaves()
	
	log.Printf("Wave Consensus [%s]: Started at %s, connecting to %d peers", 
		wc.validatorID, wc.listenAddr, len(wc.peers))
	
	return nil
}

// Stop stops the wave consensus
func (wc *WaveConsensus) Stop() {
	log.Printf("Wave Consensus [%s]: Stopping", wc.validatorID)
	wc.cancel()
	
	if wc.waveTicker != nil {
		wc.waveTicker.Stop()
	}
	
	if wc.listener != nil {
		wc.listener.Close()
	}
	
	wc.connMu.Lock()
	for _, conn := range wc.connections {
		conn.Close()
	}
	wc.connMu.Unlock()
}

// startListener starts the network listener
func (wc *WaveConsensus) startListener() error {
	listener, err := net.Listen("tcp", wc.listenAddr)
	if err != nil {
		return err
	}
	
	wc.listener = listener
	
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-wc.ctx.Done():
					return
				default:
					log.Printf("Wave Consensus [%s]: Accept error: %v", wc.validatorID, err)
					continue
				}
			}
			
			go wc.handleConnection(conn)
		}
	}()
	
	return nil
}

// connectToPeers connects to peer validators
func (wc *WaveConsensus) connectToPeers() {
	for _, peer := range wc.peers {
		go func(peerAddr string) {
			for {
				select {
				case <-wc.ctx.Done():
					return
				default:
					if err := wc.connectToPeer(peerAddr); err != nil {
						log.Printf("Wave Consensus [%s]: Failed to connect to %s: %v", wc.validatorID, peerAddr, err)
						time.Sleep(2 * time.Second)
						continue
					}
					return
				}
			}
		}(peer)
	}
}

// connectToPeer connects to a specific peer
func (wc *WaveConsensus) connectToPeer(peerAddr string) error {
	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		return err
	}
	
	wc.connMu.Lock()
	wc.connections[peerAddr] = conn
	wc.connMu.Unlock()
	
	log.Printf("Wave Consensus [%s]: Connected to peer %s", wc.validatorID, peerAddr)
	
	go wc.handleConnection(conn)
	return nil
}

// handleConnection handles a network connection
func (wc *WaveConsensus) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	decoder := json.NewDecoder(conn)
	for {
		select {
		case <-wc.ctx.Done():
			return
		default:
			var msg WaveMessage
			if err := decoder.Decode(&msg); err != nil {
				return
			}
			
			wc.handleWaveMessage(&msg)
		}
	}
}

// runWaves runs the wave consensus process
func (wc *WaveConsensus) runWaves() {
	for {
		select {
		case <-wc.ctx.Done():
			return
		case <-wc.waveTicker.C:
			wc.advanceWave()
		}
	}
}

// advanceWave advances to the next wave
func (wc *WaveConsensus) advanceWave() {
	wc.mu.Lock()
	wc.currentWave++
	currentWave := wc.currentWave
	wc.mu.Unlock()
	
	log.Printf("Wave Consensus [%s]: Advanced to wave %d", wc.validatorID, currentWave)
	
	// Get blocks from DAG sync for this wave
	go wc.processWave(currentWave)
}

// processWave processes blocks for the current wave
func (wc *WaveConsensus) processWave(wave types.Wave) {
	// Get blocks from DAG sync - use recent blocks
	recentBlocks := wc.dagReader.GetRecentBlocks(20)
	
	// Filter blocks that haven't been processed in previous waves
	newBlocks := make([]*types.Block, 0)
	
	wc.mu.RLock()
	for _, block := range recentBlocks {
		// Check if this block was already processed in a finalized wave
		alreadyProcessed := false
		for finalizedWave := range wc.finalizedWaves {
			if waveBlocks, exists := wc.waveBlocks[finalizedWave]; exists {
				for _, processedBlock := range waveBlocks {
					if string(processedBlock.ComputeHash()) == string(block.ComputeHash()) {
						alreadyProcessed = true
						break
					}
				}
			}
			if alreadyProcessed {
				break
			}
		}
		
		if !alreadyProcessed {
			// Update the block's wave field for consensus
			block.Header.Wave = wave
			newBlocks = append(newBlocks, block)
		}
	}
	wc.mu.RUnlock()
	
	if len(newBlocks) == 0 {
		log.Printf("Wave Consensus [%s]: No new blocks for wave %d", wc.validatorID, wave)
		return
	}
	
	// Store blocks for this wave
	wc.mu.Lock()
	wc.waveBlocks[wave] = newBlocks
	if wc.waveVotes[wave] == nil {
		wc.waveVotes[wave] = make(map[string]*WaveVote)
	}
	wc.mu.Unlock()
	
	log.Printf("Wave Consensus [%s]: Processing %d blocks in wave %d", 
		wc.validatorID, len(newBlocks), wave)
	
	// Vote on these blocks
	for _, block := range newBlocks {
		wc.voteOnBlock(wave, block)
	}
	
	// Broadcast blocks to other validators
	wc.broadcastWaveBlocks(wave, newBlocks)
}

// voteOnBlock votes on a block in the current wave
func (wc *WaveConsensus) voteOnBlock(wave types.Wave, block *types.Block) {
	// Simple voting logic - always vote commit for now
	// In real implementation, this would validate the block
	voteType := "commit"
	
	vote := &WaveVote{
		Wave:      wave,
		BlockHash: block.ComputeHash(),
		Voter:     wc.validatorID,
		VoteType:  voteType,
		Timestamp: time.Now(),
	}
	
	// Store our vote
	voteKey := fmt.Sprintf("%x", block.ComputeHash())
	wc.mu.Lock()
	if wc.waveVotes[wave] == nil {
		wc.waveVotes[wave] = make(map[string]*WaveVote)
	}
	wc.waveVotes[wave][voteKey] = vote
	wc.mu.Unlock()
	
	log.Printf("Wave Consensus [%s]: Voted %s for block %x in wave %d", 
		wc.validatorID, voteType, block.ComputeHash()[:8], wave)
	
	// Broadcast vote
	wc.broadcastVote(vote)
	
	// Check if we can finalize this wave
	go func() {
		time.Sleep(1 * time.Second) // Wait for other votes
		wc.checkWaveFinalization(wave)
	}()
}

// broadcastWaveBlocks broadcasts blocks for a wave
func (wc *WaveConsensus) broadcastWaveBlocks(wave types.Wave, blocks []*types.Block) {
	msg := &WaveMessage{
		Type:      "wave_blocks",
		Wave:      wave,
		Blocks:    blocks,
		Validator: wc.validatorID,
		Timestamp: time.Now(),
	}
	
	wc.broadcastMessage(msg)
}

// broadcastVote broadcasts a vote
func (wc *WaveConsensus) broadcastVote(vote *WaveVote) {
	msg := &WaveMessage{
		Type:      "vote",
		Vote:      vote,
		Wave:      vote.Wave,
		Validator: wc.validatorID,
		Timestamp: time.Now(),
	}
	
	wc.broadcastMessage(msg)
}

// broadcastMessage broadcasts a message to all peers
func (wc *WaveConsensus) broadcastMessage(msg *WaveMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Wave Consensus [%s]: Failed to marshal message: %v", wc.validatorID, err)
		return
	}
	
	wc.connMu.RLock()
	defer wc.connMu.RUnlock()
	
	broadcastCount := 0
	for peerAddr, conn := range wc.connections {
		if _, err := conn.Write(append(data, '\n')); err != nil {
			log.Printf("Wave Consensus [%s]: Failed to send to %s: %v", wc.validatorID, peerAddr, err)
			continue
		}
		broadcastCount++
	}
	
	log.Printf("Wave Consensus [%s]: Broadcasted %s to %d peers", 
		wc.validatorID, msg.Type, broadcastCount)
}

// handleWaveMessage handles received wave messages
func (wc *WaveConsensus) handleWaveMessage(msg *WaveMessage) {
	switch msg.Type {
	case "vote":
		wc.handleReceivedVote(msg.Vote, msg.Validator)
	case "wave_blocks":
		wc.handleReceivedWaveBlocks(msg.Wave, msg.Blocks, msg.Validator)
	default:
		log.Printf("Wave Consensus [%s]: Unknown message type: %s", wc.validatorID, msg.Type)
	}
}

// handleReceivedVote handles a received vote
func (wc *WaveConsensus) handleReceivedVote(vote *WaveVote, fromValidator types.Address) {
	if vote == nil {
		return
	}
	
	wc.mu.Lock()
	if wc.waveVotes[vote.Wave] == nil {
		wc.waveVotes[vote.Wave] = make(map[string]*WaveVote)
	}
	
	// Store the vote (latest vote from each validator wins)
	voteKey := fmt.Sprintf("%x", vote.BlockHash)
	voterKey := fmt.Sprintf("%s_%s", voteKey, vote.Voter)
	wc.waveVotes[vote.Wave][voterKey] = vote
	wc.mu.Unlock()
	
	log.Printf("Wave Consensus [%s]: Received %s vote from %s for block %x in wave %d", 
		wc.validatorID, vote.VoteType, fromValidator, vote.BlockHash[:8], vote.Wave)
	
	// Check if this enables finalization
	go wc.checkWaveFinalization(vote.Wave)
}

// handleReceivedWaveBlocks handles received wave blocks
func (wc *WaveConsensus) handleReceivedWaveBlocks(wave types.Wave, blocks []*types.Block, fromValidator types.Address) {
	if len(blocks) == 0 {
		return
	}
	
	log.Printf("Wave Consensus [%s]: Received %d blocks from %s for wave %d", 
		wc.validatorID, len(blocks), fromValidator, wave)
	
	// Vote on received blocks
	for _, block := range blocks {
		wc.voteOnBlock(wave, block)
	}
}

// checkWaveFinalization checks if a wave can be finalized
func (wc *WaveConsensus) checkWaveFinalization(wave types.Wave) {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	
	// Check if already finalized
	if wc.finalizedWaves[wave] {
		return
	}
	
	// Count votes for each block
	waveVotes := wc.waveVotes[wave]
	if waveVotes == nil {
		return
	}
	
	// Group votes by block
	blockVotes := make(map[string]map[string]*WaveVote)
	for _, vote := range waveVotes {
		blockKey := fmt.Sprintf("%x", vote.BlockHash)
		if blockVotes[blockKey] == nil {
			blockVotes[blockKey] = make(map[string]*WaveVote)
		}
		blockVotes[blockKey][string(vote.Voter)] = vote
	}
	
	// Check if any block has enough votes to finalize (simple majority)
	totalValidators := len(wc.peers) + 1 // peers + self
	requiredVotes := (totalValidators / 2) + 1
	
	finalizedBlocks := 0
	for blockKey, votes := range blockVotes {
		commitVotes := 0
		for _, vote := range votes {
			if vote.VoteType == "commit" {
				commitVotes++
			}
		}
		
		if commitVotes >= requiredVotes {
			finalizedBlocks++
			log.Printf("Wave Consensus [%s]: Block %s finalized in wave %d with %d/%d votes", 
				wc.validatorID, blockKey[:16], wave, commitVotes, totalValidators)
		}
	}
	
	// Finalize wave if we have processed enough blocks
	if finalizedBlocks > 0 {
		wc.finalizedWaves[wave] = true
		log.Printf("Wave Consensus [%s]: ✅ Wave %d FINALIZED with %d blocks", 
			wc.validatorID, wave, finalizedBlocks)
	}
}

// GetWaveStatus returns the current wave consensus status
func (wc *WaveConsensus) GetWaveStatus() map[string]interface{} {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	
	// Count finalized waves
	finalizedCount := len(wc.finalizedWaves)
	
	// Count votes per wave
	waveVoteCounts := make(map[string]int)
	for wave, votes := range wc.waveVotes {
		waveVoteCounts[fmt.Sprintf("wave_%d", wave)] = len(votes)
	}
	
	wc.connMu.RLock()
	connectedPeers := len(wc.connections)
	wc.connMu.RUnlock()
	
	return map[string]interface{}{
		"validator_id":      wc.validatorID,
		"current_wave":      wc.currentWave,
		"finalized_waves":   finalizedCount,
		"connected_peers":   connectedPeers,
		"votes_per_wave":    waveVoteCounts,
		"listen_addr":       wc.listenAddr,
	}
}

// GetCurrentWave returns the current wave
func (wc *WaveConsensus) GetCurrentWave() types.Wave {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	return wc.currentWave
} 