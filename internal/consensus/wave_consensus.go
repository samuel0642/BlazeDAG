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
	confirmedBlocks map[string]bool // confirmed blocks by hash (including transitively confirmed)
	
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
		confirmedBlocks: make(map[string]bool),
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
	
	// Process ALL blocks from ALL validators (not just leader's own blocks)
	// This ensures cross-validator block confirmation
	if len(recentBlocks) == 0 {
		log.Printf("Wave Consensus [%s]: 📭 No blocks found for wave %d", wc.validatorID, wave)
		return
	}

	// Find the block with the highest round from ALL validators
	var topRoundBlock *types.Block
	var highestRound types.Round = 0
	var candidateBlocks []*types.Block // Store all blocks with highest round
	
	wc.mu.RLock()
	for _, block := range recentBlocks {
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
		
		// Collect blocks with highest round that haven't been processed
		if !alreadyProcessed {
			if block.Header.Round > highestRound {
				// Found higher round - reset candidates
				highestRound = block.Header.Round
				candidateBlocks = []*types.Block{block}
			} else if block.Header.Round == highestRound {
				// Same round - add to candidates
				candidateBlocks = append(candidateBlocks, block)
			}
		}
	}
	wc.mu.RUnlock()

	if len(candidateBlocks) == 0 {
		log.Printf("Wave Consensus [%s]: 📭 No unprocessed blocks for wave %d - finalizing empty wave", wc.validatorID, wave)
		
		// For empty waves, immediately mark as finalized to maintain sequential progression
		wc.mu.Lock()
		wc.finalizedWaves[wave] = true
		wc.mu.Unlock()
		
		log.Printf("Wave Consensus [%s]: ✅ Wave %d FINALIZED (empty wave)", wc.validatorID, wave)
		return
	}

	// Fair selection: rotate based on wave number to ensure all validators get blocks selected
	selectedIndex := int(wave) % len(candidateBlocks)
	topRoundBlock = candidateBlocks[selectedIndex]
	
	log.Printf("Wave Consensus [%s]: 🎯 FAIR SELECTION - Found %d candidate blocks with round %d", 
		wc.validatorID, len(candidateBlocks), highestRound)
	
	// Log all candidates
	for i, candidate := range candidateBlocks {
		marker := "⚪"
		if i == selectedIndex {
			marker = "🎯"
		}
		log.Printf("Wave Consensus [%s]: %s Candidate %d: validator %s, hash %x", 
			wc.validatorID, marker, i, candidate.Header.Validator, candidate.ComputeHash()[:8])
	}
	
	log.Printf("Wave Consensus [%s]: ✅ SELECTED block from validator %s (index %d of %d)", 
		wc.validatorID, topRoundBlock.Header.Validator, selectedIndex, len(candidateBlocks))

	// Update the wave field for the selected block
	topRoundBlock.Header.Wave = wave

	// Store the selected block for this wave
	wc.mu.Lock()
	wc.waveBlocks[wave] = []*types.Block{topRoundBlock} // Only one block per wave
	if wc.waveVotes[wave] == nil {
		wc.waveVotes[wave] = make(map[string]*WaveVote)
	}
	wc.mu.Unlock()

	log.Printf("Wave Consensus [%s]: 🚀 LEADER selected TOP ROUND block for wave %d from validator %s", 
		wc.validatorID, wave, topRoundBlock.Header.Validator)
	log.Printf("Wave Consensus [%s]: 📦 Block Hash: %x", wc.validatorID, topRoundBlock.ComputeHash()[:8])
	log.Printf("Wave Consensus [%s]: 🔄 Round: %d", wc.validatorID, topRoundBlock.Header.Round)
	log.Printf("Wave Consensus [%s]: 📏 Height: %d", wc.validatorID, topRoundBlock.Header.Height)
	log.Printf("Wave Consensus [%s]: 👤 Block Creator: %s", wc.validatorID, topRoundBlock.Header.Validator)

	// Leader votes on the selected block (regardless of which validator created it)
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
	
	// Leaders can now propose blocks from ANY validator (cross-validator confirmation)
	// No need to validate block creator matches leader
	for _, block := range blocks {
		log.Printf("Wave Consensus [%s]: 📋 Processing block %x from validator %s (proposed by leader %s)", 
			wc.validatorID, block.ComputeHash()[:8], block.Header.Validator, fromValidator)
	}

	// Vote on received blocks from valid leader
	for _, block := range blocks {
		log.Printf("Wave Consensus [%s]: 🗳️ Voting on leader's proposed block %x in wave %d", 
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
	var blocksToConfirm []*types.Block
	
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
			
			// Find the corresponding block to confirm it and its references
			if waveBlocks, exists := wc.waveBlocks[wave]; exists {
				for _, block := range waveBlocks {
					var currentHash string
					if len(block.OriginalHash) > 0 {
						currentHash = fmt.Sprintf("%x", block.OriginalHash)
					} else {
						currentHash = fmt.Sprintf("%x", block.ComputeHash())
					}
					
					if currentHash == blockKey {
						blocksToConfirm = append(blocksToConfirm, block)
						break
					}
				}
			}
		}
	}
	
	// Finalize wave if we have processed enough blocks
	if finalizedBlocks > 0 {
		wc.finalizedWaves[wave] = true
		log.Printf("Wave Consensus [%s]: ✅ Wave %d FINALIZED with %d blocks", 
			wc.validatorID, wave, finalizedBlocks)
		
		// Confirm all finalized blocks and their references transitively
		for _, block := range blocksToConfirm {
			log.Printf("Wave Consensus [%s]: 🔗 Starting transitive confirmation for block in wave %d", 
				wc.validatorID, wave)
			wc.confirmBlockAndReferences(block)
		}
		
		log.Printf("Wave Consensus [%s]: 📊 Total confirmed blocks: %d", 
			wc.validatorID, len(wc.confirmedBlocks))
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
	mux.HandleFunc("/block_confirmation", wc.handleBlockConfirmation)
	mux.HandleFunc("/all_blocks_confirmation", wc.handleAllBlocksConfirmation)
	
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

// GetBlockConfirmationStatus returns detailed block confirmation status
func (wc *WaveConsensus) GetBlockConfirmationStatus(blockHash string) map[string]interface{} {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	
	// Check if block is transitively confirmed first
	isTransitivelyConfirmed := wc.confirmedBlocks[blockHash]
	
	// Find which wave this block belongs to
	var blockWave types.Wave = 0
	var found bool
	var isInWaveBlocks bool
	
	for wave, blocks := range wc.waveBlocks {
		for _, block := range blocks {
			if fmt.Sprintf("%x", block.ComputeHash()) == blockHash || 
			   fmt.Sprintf("%x", block.OriginalHash) == blockHash {
				blockWave = wave
				found = true
				isInWaveBlocks = true
				break
			}
		}
		if found {
			break
		}
	}
	
	// If not found in wave blocks but transitively confirmed, it's still confirmed
	if !found && isTransitivelyConfirmed {
		return map[string]interface{}{
			"block_hash": blockHash,
			"status": "finalized",
			"message": "Block confirmed transitively through DAG references",
			"is_finalized": true,
			"is_transitively_confirmed": true,
			"votes": map[string]interface{}{
				"commit_votes": "N/A",
				"total_votes": "N/A", 
				"required_votes": "N/A",
				"total_validators": len(wc.validators),
			},
			"wave": "N/A",
			"current_wave": wc.currentWave,
		}
	}
	
	if !found {
		return map[string]interface{}{
			"block_hash": blockHash,
			"status": "not_found",
			"message": "Block not found in wave consensus",
			"is_transitively_confirmed": false,
		}
	}
	
	// Check if wave is finalized
	isWaveFinalized := wc.finalizedWaves[blockWave]
	
	// Count votes for this block
	waveVotes := wc.waveVotes[blockWave]
	blockVotes := 0
	commitVotes := 0
	
	if waveVotes != nil {
		for _, vote := range waveVotes {
			if fmt.Sprintf("%x", vote.BlockHash) == blockHash {
				blockVotes++
				if vote.VoteType == "commit" {
					commitVotes++
				}
			}
		}
	}
	
	totalValidators := len(wc.validators)
	requiredVotes := (totalValidators / 2) + 1
	
	// Determine status - prioritize transitive confirmation
	var status string
	var message string
	var isFinalized bool
	
	if isTransitivelyConfirmed || isWaveFinalized {
		status = "finalized"
		isFinalized = true
		if isTransitivelyConfirmed && isWaveFinalized {
			message = fmt.Sprintf("Block finalized in wave %d and confirmed transitively", blockWave)
		} else if isTransitivelyConfirmed {
			message = fmt.Sprintf("Block confirmed transitively through DAG references")
		} else {
			message = fmt.Sprintf("Block finalized in wave %d", blockWave)
		}
	} else if blockWave > wc.currentWave {
		status = "pending"
		message = fmt.Sprintf("Block scheduled for future wave %d", blockWave)
	} else if commitVotes >= requiredVotes {
		status = "confirming"
		message = fmt.Sprintf("Block has sufficient votes, awaiting wave finalization")
	} else if blockVotes > 0 {
		status = "voting"
		message = fmt.Sprintf("Block is being voted on (%d/%d votes)", commitVotes, totalValidators)
	} else {
		status = "proposed"
		message = fmt.Sprintf("Block proposed in wave %d, awaiting votes", blockWave)
	}
	
	return map[string]interface{}{
		"block_hash": blockHash,
		"wave": blockWave,
		"status": status,
		"message": message,
		"is_finalized": isFinalized,
		"is_transitively_confirmed": isTransitivelyConfirmed,
		"is_in_wave_blocks": isInWaveBlocks,
		"votes": map[string]interface{}{
			"commit_votes": commitVotes,
			"total_votes": blockVotes,
			"required_votes": requiredVotes,
			"total_validators": totalValidators,
		},
		"current_wave": wc.currentWave,
	}
}

// GetAllBlocksConfirmationStatus returns confirmation status for all blocks
func (wc *WaveConsensus) GetAllBlocksConfirmationStatus() map[string]interface{} {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	
	blockStatuses := make(map[string]interface{})
	
	// Process all blocks in all waves
	for wave, blocks := range wc.waveBlocks {
		for _, block := range blocks {
			blockHash := fmt.Sprintf("%x", block.ComputeHash())
			if len(block.OriginalHash) > 0 {
				blockHash = fmt.Sprintf("%x", block.OriginalHash)
			}
			
			// Check if wave is finalized
			isWaveFinalized := wc.finalizedWaves[wave]
			isTransitivelyConfirmed := wc.confirmedBlocks[blockHash]
			
			// Count votes for this block
			waveVotes := wc.waveVotes[wave]
			blockVotes := 0
			commitVotes := 0
			
			if waveVotes != nil {
				for _, vote := range waveVotes {
					if fmt.Sprintf("%x", vote.BlockHash) == blockHash {
						blockVotes++
						if vote.VoteType == "commit" {
							commitVotes++
						}
					}
				}
			}
			
			totalValidators := len(wc.validators)
			requiredVotes := (totalValidators / 2) + 1
			
			// Determine status - prioritize transitive confirmation
			var status string
			var isFinalized bool
			
			if isTransitivelyConfirmed || isWaveFinalized {
				status = "finalized"
				isFinalized = true
			} else if wave > wc.currentWave {
				status = "pending"
			} else if commitVotes >= requiredVotes {
				status = "confirming"
			} else if blockVotes > 0 {
				status = "voting"
			} else {
				status = "proposed"
			}
			
			blockStatuses[blockHash] = map[string]interface{}{
				"wave": wave,
				"status": status,
				"is_finalized": isFinalized,
				"is_transitively_confirmed": isTransitivelyConfirmed,
				"is_wave_finalized": isWaveFinalized,
				"commit_votes": commitVotes,
				"total_votes": blockVotes,
				"required_votes": requiredVotes,
				"total_validators": totalValidators,
			}
		}
	}
	
	// Also include blocks that are transitively confirmed but not in wave blocks
	for blockHash, confirmed := range wc.confirmedBlocks {
		if confirmed && blockStatuses[blockHash] == nil {
			blockStatuses[blockHash] = map[string]interface{}{
				"wave": "N/A",
				"status": "finalized",
				"is_finalized": true,
				"is_transitively_confirmed": true,
				"is_wave_finalized": false,
				"commit_votes": "N/A",
				"total_votes": "N/A",
				"required_votes": "N/A",
				"total_validators": len(wc.validators),
			}
		}
	}
	
	return map[string]interface{}{
		"current_wave": wc.currentWave,
		"total_finalized_waves": len(wc.finalizedWaves),
		"total_confirmed_blocks": len(wc.confirmedBlocks),
		"blocks": blockStatuses,
	}
}

// handleBlockConfirmation handles the /block_confirmation API endpoint
func (wc *WaveConsensus) handleBlockConfirmation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	blockHash := r.URL.Query().Get("hash")
	if blockHash == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "block hash parameter required"})
		return
	}
	
	status := wc.GetBlockConfirmationStatus(blockHash)
	json.NewEncoder(w).Encode(status)
}

// handleAllBlocksConfirmation handles the /all_blocks_confirmation API endpoint
func (wc *WaveConsensus) handleAllBlocksConfirmation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	status := wc.GetAllBlocksConfirmationStatus()
	json.NewEncoder(w).Encode(status)
}

// confirmBlockAndReferences confirms a block and all blocks it references transitively
func (wc *WaveConsensus) confirmBlockAndReferences(block *types.Block) {
	if block == nil {
		return
	}
	
	// Get block hash for tracking
	var blockHash string
	if len(block.OriginalHash) > 0 {
		blockHash = fmt.Sprintf("%x", block.OriginalHash)
	} else {
		blockHash = fmt.Sprintf("%x", block.ComputeHash())
	}
	
	// Check if already confirmed to avoid infinite recursion
	if wc.confirmedBlocks[blockHash] {
		return
	}
	
	// Mark this block as confirmed
	wc.confirmedBlocks[blockHash] = true
	log.Printf("Wave Consensus [%s]: ✅ CONFIRMED block %s (Round: %d, Wave: %d)", 
		wc.validatorID, blockHash[:16], block.Header.Round, block.Header.Wave)
	
	// Recursively confirm all referenced blocks
	if block.Header.References != nil {
		for _, ref := range block.Header.References {
			refHashStr := fmt.Sprintf("%x", ref.BlockHash)
			
			// Try to find the referenced block in our local storage
			referencedBlock := wc.findBlockByHash(ref.BlockHash)
			if referencedBlock != nil {
				log.Printf("Wave Consensus [%s]: 🔗 Transitively confirming referenced block %s (Round: %d, Wave: %d)", 
					wc.validatorID, refHashStr[:16], ref.Round, ref.Wave)
				wc.confirmBlockAndReferences(referencedBlock)
			} else {
				// Block not found locally - mark as confirmed anyway since we're accepting the reference
				if !wc.confirmedBlocks[refHashStr] {
					wc.confirmedBlocks[refHashStr] = true
					log.Printf("Wave Consensus [%s]: ✅ CONFIRMED referenced block %s (not found locally but accepted via reference)", 
						wc.validatorID, refHashStr[:16])
				}
			}
		}
	}
	
	// ENHANCED: Also confirm blocks from the same DAG layer
	// Get all recent blocks and confirm those in the same round or earlier
	// This ensures comprehensive confirmation across all validators
	recentBlocks := wc.dagReader.GetRecentBlocks(100)
	for _, dagBlock := range recentBlocks {
		var dagBlockHash string
		if len(dagBlock.OriginalHash) > 0 {
			dagBlockHash = fmt.Sprintf("%x", dagBlock.OriginalHash)
		} else {
			dagBlockHash = fmt.Sprintf("%x", dagBlock.ComputeHash())
		}
		
		// Confirm blocks from same round or earlier rounds (causal history)
		if dagBlock.Header.Round <= block.Header.Round && !wc.confirmedBlocks[dagBlockHash] {
			wc.confirmedBlocks[dagBlockHash] = true
			log.Printf("Wave Consensus [%s]: 🌊 DAG-LEVEL confirmed block %s from validator %s (Round: %d)", 
				wc.validatorID, dagBlockHash[:16], dagBlock.Header.Validator, dagBlock.Header.Round)
		}
	}
}

// findBlockByHash finds a block by hash in wave blocks
func (wc *WaveConsensus) findBlockByHash(blockHash []byte) *types.Block {
	hashStr := fmt.Sprintf("%x", blockHash)
	
	// Search through all wave blocks
	for _, blocks := range wc.waveBlocks {
		for _, block := range blocks {
			var currentHash string
			if len(block.OriginalHash) > 0 {
				currentHash = fmt.Sprintf("%x", block.OriginalHash)
			} else {
				currentHash = fmt.Sprintf("%x", block.ComputeHash())
			}
			
			if currentHash == hashStr {
				return block
			}
		}
	}
	
	return nil
} 