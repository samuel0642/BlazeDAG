package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
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
	validators    []types.Address  // All validators in the network
	dagReader     DAGReader     // Interface to read blocks from DAG
	currentWave   types.Wave
	waveDuration  time.Duration
	
	// Network configuration  
	listenAddr    string
	peers         []string
	
	// HTTP API configuration
	httpAddr      string
	httpServer    *http.Server
	
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
func NewWaveConsensus(validatorID types.Address, validators []types.Address, dagReader DAGReader, listenAddr string, peers []string, waveDuration time.Duration) *WaveConsensus {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set up HTTP API on port 8081 for wave consensus
	host, _, _ := net.SplitHostPort(listenAddr)
	httpAddr := fmt.Sprintf("%s:8081", host) // Wave consensus HTTP API on port 8081
	
	log.Printf("🔧 Creating WaveConsensus for validator: %s", validatorID)
	log.Printf("🔧 Validators list: %v", validators)
	log.Printf("🔧 Total validators: %d", len(validators))
	
	return &WaveConsensus{
		validatorID:    validatorID,
		validators:     validators,
		dagReader:      dagReader,
		currentWave:    0,
		waveDuration:   waveDuration,
		listenAddr:     listenAddr,
		peers:          peers,
		httpAddr:       httpAddr,
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
	
	// Start HTTP API server
	if err := wc.startHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %v", err)
	}
	
	// Connect to peers
	go wc.connectToPeers()
	
	// Wait for all peers to connect before starting waves
	if len(wc.peers) > 0 {
		log.Printf("Wave Consensus [%s]: Waiting for all %d peers to connect...", wc.validatorID, len(wc.peers))
		wc.waitForAllPeers()
		log.Printf("Wave Consensus [%s]: ✅ All peers connected! Starting synchronized waves...", wc.validatorID)
	} else {
		log.Printf("Wave Consensus [%s]: No peers configured, starting immediately", wc.validatorID)
	}
	
	// Start wave ticker AFTER all peers are connected
	wc.waveTicker = time.NewTicker(wc.waveDuration)
	go wc.runWaves()
	
	log.Printf("Wave Consensus [%s]: Started at %s, connected to %d peers", 
		wc.validatorID, wc.listenAddr, len(wc.peers))
	log.Printf("Wave Consensus [%s]: HTTP API available at http://%s", wc.validatorID, wc.httpAddr)
	
	return nil
}

// Stop stops the wave consensus
func (wc *WaveConsensus) Stop() {
	log.Printf("Wave Consensus [%s]: Stopping", wc.validatorID)
	wc.cancel()
	
	if wc.waveTicker != nil {
		wc.waveTicker.Stop()
	}
	
	if wc.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		wc.httpServer.Shutdown(ctx)
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
	
	// Check if previous wave is finalized before advancing
	if wc.currentWave > 0 && !wc.finalizedWaves[wc.currentWave] {
		log.Printf("Wave Consensus [%s]: ⏳ Waiting for wave %d to finalize before advancing", wc.validatorID, wc.currentWave)
		wc.mu.Unlock()
		return
	}
	
	wc.currentWave++
	currentWave := wc.currentWave
	wc.mu.Unlock()
	
	leader := wc.getWaveLeader(currentWave)
	log.Printf("Wave Consensus [%s]: Advanced to wave %d, Leader: %s", wc.validatorID, currentWave, leader)
	
	// Only process wave if this validator is the leader
	if wc.isWaveLeader(currentWave) {
		log.Printf("Wave Consensus [%s]: 👑 I am the LEADER for wave %d!", wc.validatorID, currentWave)
		go wc.processWave(currentWave)
	} else {
		log.Printf("Wave Consensus [%s]: ⚪ Not leader for wave %d, waiting for leader's proposal...", wc.validatorID, currentWave)
	}
}

// isWaveLeader checks if this validator is the leader for the given wave
func (wc *WaveConsensus) isWaveLeader(wave types.Wave) bool {
	if len(wc.validators) == 0 {
		log.Printf("🔧 DEBUG: No validators configured, defaulting to leader (single validator mode)")
		return true // Single validator mode - always leader
	}
	leaderIndex := int(wave) % len(wc.validators)
	currentLeader := wc.validators[leaderIndex]
	isLeader := currentLeader == wc.validatorID
	
	log.Printf("🔧 DEBUG: Wave %d, leaderIndex: %d, currentLeader: %s, myID: %s, isLeader: %t", 
		wave, leaderIndex, currentLeader, wc.validatorID, isLeader)
	
	return isLeader
}

// getWaveLeader returns the leader for the given wave
func (wc *WaveConsensus) getWaveLeader(wave types.Wave) types.Address {
	if len(wc.validators) == 0 {
		log.Printf("🔧 DEBUG: No validators configured, returning own ID as leader")
		return wc.validatorID // Single validator mode
	}
	leaderIndex := int(wave) % len(wc.validators)
	leader := wc.validators[leaderIndex]
	
	log.Printf("🔧 DEBUG: Wave %d leader calculation: leaderIndex=%d, leader=%s, validators=%v", 
		wave, leaderIndex, leader, wc.validators)
	
	return leader
}

// isValidWaveLeader validates if the given validator is the correct leader for the wave
func (wc *WaveConsensus) isValidWaveLeader(validator types.Address, wave types.Wave) bool {
	return wc.getWaveLeader(wave) == validator
}

// processWave processes blocks for the current wave (only called by leader)
func (wc *WaveConsensus) processWave(wave types.Wave) {
	// Double-check that we are the leader
	if !wc.isWaveLeader(wave) {
		log.Printf("Wave Consensus [%s]: ❌ processWave called but not leader for wave %d", wc.validatorID, wave)
		return
	}

	// Get blocks from DAG sync
	recentBlocks := wc.dagReader.GetRecentBlocks(20)
	
	// Filter to only get blocks created by this validator (leader's own blocks)
	var ownBlocks []*types.Block
	for _, block := range recentBlocks {
		if block.Header.Validator == wc.validatorID {
			ownBlocks = append(ownBlocks, block)
		}
	}

	if len(ownBlocks) == 0 {
		log.Printf("Wave Consensus [%s]: 📭 No own blocks found for wave %d", wc.validatorID, wave)
		return
	}

	// Find the block with the highest round from own blocks
	var topRoundBlock *types.Block
	var highestRound types.Round = 0
	
	wc.mu.RLock()
	for _, block := range ownBlocks {
		// Check if this block was already processed in previous waves
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
		
		// Select the highest round block that hasn't been processed
		if !alreadyProcessed && block.Header.Round >= highestRound {
			highestRound = block.Header.Round
			topRoundBlock = block
		}
	}
	wc.mu.RUnlock()

	if topRoundBlock == nil {
		log.Printf("Wave Consensus [%s]: 📭 No unprocessed own blocks for wave %d - finalizing empty wave", wc.validatorID, wave)
		
		// For empty waves, immediately mark as finalized to maintain sequential progression
		wc.mu.Lock()
		wc.finalizedWaves[wave] = true
		wc.mu.Unlock()
		
		log.Printf("Wave Consensus [%s]: ✅ Wave %d FINALIZED (empty wave)", wc.validatorID, wave)
		return
	}

	// Update the wave field for the selected block
	topRoundBlock.Header.Wave = wave

	// Store the selected block for this wave
	wc.mu.Lock()
	wc.waveBlocks[wave] = []*types.Block{topRoundBlock} // Only one block per wave
	if wc.waveVotes[wave] == nil {
		wc.waveVotes[wave] = make(map[string]*WaveVote)
	}
	wc.mu.Unlock()

	log.Printf("Wave Consensus [%s]: 🚀 LEADER selected TOP ROUND block for wave %d", wc.validatorID, wave)
	log.Printf("Wave Consensus [%s]: 📦 Block Hash: %x", wc.validatorID, topRoundBlock.ComputeHash()[:8])
	log.Printf("Wave Consensus [%s]: 🔄 Round: %d", wc.validatorID, topRoundBlock.Header.Round)
	log.Printf("Wave Consensus [%s]: 📏 Height: %d", wc.validatorID, topRoundBlock.Header.Height)

	// Leader votes on their own block first
	wc.voteOnBlock(wave, topRoundBlock)

	// Broadcast the selected block to other validators
	wc.broadcastWaveBlocks(wave, []*types.Block{topRoundBlock})
}

// voteOnBlock votes on a block in the current wave
func (wc *WaveConsensus) voteOnBlock(wave types.Wave, block *types.Block) {
	// Simple voting logic - always vote commit for now
	// In real implementation, this would validate the block
	voteType := "commit"
	
	// Use original hash if available (from DAG sync), otherwise compute hash
	var blockHash []byte
	if len(block.OriginalHash) > 0 {
		blockHash = block.OriginalHash
		log.Printf("Wave Consensus [%s]: Using original hash for voting: %x", wc.validatorID, blockHash[:8])
	} else {
		blockHash = block.ComputeHash()
		log.Printf("Wave Consensus [%s]: Computing hash for voting: %x", wc.validatorID, blockHash[:8])
	}
	
	vote := &WaveVote{
		Wave:      wave,
		BlockHash: blockHash,
		Voter:     wc.validatorID,
		VoteType:  voteType,
		Timestamp: time.Now(),
	}
	
	// Store our vote
	voteKey := fmt.Sprintf("%x", blockHash)
	wc.mu.Lock()
	if wc.waveVotes[wave] == nil {
		wc.waveVotes[wave] = make(map[string]*WaveVote)
	}
	wc.waveVotes[wave][voteKey] = vote
	wc.mu.Unlock()
	
	log.Printf("Wave Consensus [%s]: Voted %s for block %x in wave %d", 
		wc.validatorID, voteType, blockHash[:8], wave)
	
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
	
	// Validate that the voter is in our validator set
	validVoter := false
	for _, validator := range wc.validators {
		if validator == vote.Voter {
			validVoter = true
			break
		}
	}
	
	if !validVoter {
		log.Printf("Wave Consensus [%s]: ❌ Rejected vote from invalid validator %s", wc.validatorID, vote.Voter)
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
	
	log.Printf("Wave Consensus [%s]: ✅ Received %s vote from validator %s for block %x in wave %d", 
		wc.validatorID, vote.VoteType, vote.Voter, vote.BlockHash[:8], vote.Wave)
	
	// Check if this enables finalization
	go wc.checkWaveFinalization(vote.Wave)
}

// handleReceivedWaveBlocks handles received wave blocks
func (wc *WaveConsensus) handleReceivedWaveBlocks(wave types.Wave, blocks []*types.Block, fromValidator types.Address) {
	if len(blocks) == 0 {
		return
	}
	
	// Validate that the sender is the correct leader for this wave
	if !wc.isValidWaveLeader(fromValidator, wave) {
		log.Printf("Wave Consensus [%s]: ❌ Rejected blocks from %s - not valid leader for wave %d", 
			wc.validatorID, fromValidator, wave)
		return
	}

	log.Printf("Wave Consensus [%s]: ✅ Received %d blocks from VALID LEADER %s for wave %d", 
		wc.validatorID, len(blocks), fromValidator, wave)
	
	// Validate that all blocks in the proposal were created by the leader
	for _, block := range blocks {
		if block.Header.Validator != fromValidator {
			log.Printf("Wave Consensus [%s]: ❌ Rejected block %x - not created by leader %s", 
				wc.validatorID, block.ComputeHash()[:8], fromValidator)
			return
		}
	}

	// Vote on received blocks from valid leader
	for _, block := range blocks {
		log.Printf("Wave Consensus [%s]: 🗳️ Voting on leader's block %x in wave %d", 
			wc.validatorID, block.ComputeHash()[:8], wave)
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
	totalValidators := len(wc.validators)
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
			log.Printf("Wave Consensus [%s]: 🎉 Block %s finalized in wave %d with %d/%d votes", 
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
	
	// Get current wave leader info
	currentLeader := wc.getWaveLeader(wc.currentWave)
	isCurrentLeader := wc.isWaveLeader(wc.currentWave)
	
	return map[string]interface{}{
		"validator_id":      wc.validatorID,
		"current_wave":      wc.currentWave,
		"current_leader":    currentLeader,
		"is_current_leader": isCurrentLeader,
		"validators":        wc.validators,
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

// startHTTPServer starts the HTTP API server
func (wc *WaveConsensus) startHTTPServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/wave_status", wc.handleWaveStatus)
	mux.HandleFunc("/current_wave", wc.handleCurrentWave)
	
	wc.httpServer = &http.Server{
		Addr:    wc.httpAddr,
		Handler: mux,
	}
	
	go func() {
		if err := wc.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Wave Consensus [%s]: HTTP server error: %v", wc.validatorID, err)
		}
	}()
	
	return nil
}

// handleWaveStatus handles the /wave_status API endpoint
func (wc *WaveConsensus) handleWaveStatus(w http.ResponseWriter, r *http.Request) {
	status := wc.GetWaveStatus()
	json.NewEncoder(w).Encode(status)
}

// handleCurrentWave handles the /current_wave API endpoint
func (wc *WaveConsensus) handleCurrentWave(w http.ResponseWriter, r *http.Request) {
	currentWave := wc.GetCurrentWave()
	json.NewEncoder(w).Encode(currentWave)
}

// waitForAllPeers waits for all peers to connect before starting waves
func (wc *WaveConsensus) waitForAllPeers() {
	for {
		select {
		case <-wc.ctx.Done():
			return
		default:
			wc.connMu.RLock()
			connectedPeers := len(wc.connections)
			wc.connMu.RUnlock()
			
			if connectedPeers >= len(wc.peers) {
				return
			}
			
			log.Printf("Wave Consensus [%s]: Connected to %d/%d peers, waiting...", 
				wc.validatorID, connectedPeers, len(wc.peers))
			time.Sleep(1 * time.Second)
		}
	}
} 