package core

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/dag"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// WaveConsensus handles BlazeDAG consensus waves independently from DAG transport
type WaveConsensus struct {
	dagTransport  *dag.DAGTransport
	currentWave   types.Wave
	waveTimeout   time.Duration
	validatorID   types.Address
	validators    []types.Address
	f             int // fault tolerance threshold
	
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	
	// Wave management
	proposals     map[types.Wave]*types.Proposal
	votes         map[types.Wave]map[string]*types.Vote
	complaints    map[types.Wave]map[string]*types.Complaint
	waveTimer     *time.Timer
	
	// Committed blocks
	committedBlocks map[string]*types.Block
	commitOrder     []*types.Block
}

// NewWaveConsensus creates a new wave consensus engine
func NewWaveConsensus(dagTransport *dag.DAGTransport, validatorID types.Address, validators []types.Address, waveTimeout time.Duration) *WaveConsensus {
	ctx, cancel := context.WithCancel(context.Background())
	f := (len(validators) - 1) / 3 // Byzantine fault tolerance
	
	return &WaveConsensus{
		dagTransport:    dagTransport,
		currentWave:     1,
		waveTimeout:     waveTimeout,
		validatorID:     validatorID,
		validators:      validators,
		f:               f,
		ctx:             ctx,
		cancel:          cancel,
		proposals:       make(map[types.Wave]*types.Proposal),
		votes:           make(map[types.Wave]map[string]*types.Vote),
		complaints:      make(map[types.Wave]map[string]*types.Complaint),
		committedBlocks: make(map[string]*types.Block),
		commitOrder:     make([]*types.Block, 0),
	}
}

// Start begins the independent wave consensus operation
func (wc *WaveConsensus) Start() {
	go wc.runWaveConsensus()
	log.Printf("Wave Consensus started - operating independently from DAG rounds")
}

// Stop stops the wave consensus
func (wc *WaveConsensus) Stop() {
	wc.cancel()
	if wc.waveTimer != nil {
		wc.waveTimer.Stop()
	}
}

// runWaveConsensus runs the wave consensus independently
func (wc *WaveConsensus) runWaveConsensus() {
	wc.startNewWave()
	
	for {
		select {
		case <-wc.ctx.Done():
			return
		default:
			// Process consensus operations
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// startNewWave starts a new consensus wave
func (wc *WaveConsensus) startNewWave() {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	
	wave := wc.currentWave
	
	// Initialize wave data structures
	wc.votes[wave] = make(map[string]*types.Vote)
	wc.complaints[wave] = make(map[string]*types.Complaint)
	
	// Set wave timer
	if wc.waveTimer != nil {
		wc.waveTimer.Stop()
	}
	wc.waveTimer = time.AfterFunc(wc.waveTimeout, func() {
		wc.handleWaveTimeout(wave)
	})
	
	// Check if this validator is the leader for this wave
	if wc.isLeader(wave) {
		go wc.createProposal(wave)
	}
	
	log.Printf("Wave Consensus: Started wave %d (independent of DAG rounds)", wave)
}

// isLeader determines if this validator is the leader for the given wave
func (wc *WaveConsensus) isLeader(wave types.Wave) bool {
	leaderIndex := int(wave) % len(wc.validators)
	return wc.validators[leaderIndex] == wc.validatorID
}

// createProposal creates a proposal for the current wave
func (wc *WaveConsensus) createProposal(wave types.Wave) {
	// Get uncommitted blocks from DAG transport
	uncommittedBlocks := wc.dagTransport.GetUncommittedBlocks()
	
	if len(uncommittedBlocks) == 0 {
		log.Printf("Wave Consensus: No uncommitted blocks to propose in wave %d", wave)
		return
	}
	
	// Select the highest DAG round block as the proposal
	var proposalBlock *types.Block
	maxRound := types.Round(0)
	for _, block := range uncommittedBlocks {
		if block.Header.Round > maxRound {
			maxRound = block.Header.Round
			proposalBlock = block
		}
	}
	
	if proposalBlock == nil {
		return
	}
	
	// Set the wave number for the proposal (consensus layer responsibility)
	proposalBlock.Header.Wave = wave
	
	// Create proposal
	proposal := &types.Proposal{
		ID:        proposalBlock.ComputeHash(),
		BlockHash: proposalBlock.ComputeHash(),
		Round:     proposalBlock.Header.Round, // DAG round
		Wave:      wave,                       // Consensus wave
		Block:     proposalBlock,
		Proposer:  wc.validatorID,
		Timestamp: time.Now(),
		Status:    types.ProposalStatusPending,
	}
	
	wc.mu.Lock()
	wc.proposals[wave] = proposal
	wc.mu.Unlock()
	
	log.Printf("Wave Consensus: Created proposal for wave %d with DAG round %d block", wave, proposalBlock.Header.Round)
	
	// Broadcast proposal (simulate)
	wc.broadcastProposal(proposal)
}

// broadcastProposal broadcasts a proposal to other validators
func (wc *WaveConsensus) broadcastProposal(proposal *types.Proposal) {
	// In a real implementation, this would send the proposal over the network
	// For simulation, we'll have validators vote on their own proposals
	go wc.handleProposal(proposal)
}

// handleProposal handles a received proposal
func (wc *WaveConsensus) handleProposal(proposal *types.Proposal) {
	wave := proposal.Wave
	
	// Validate proposal
	if !wc.validateProposal(proposal) {
		log.Printf("Wave Consensus: Invalid proposal for wave %d", wave)
		return
	}
	
	// Create vote
	vote := &types.Vote{
		ProposalID: proposal.ID,
		BlockHash:  proposal.BlockHash,
		Round:      proposal.Round,
		Wave:       wave,
		Validator:  wc.validatorID,
		Timestamp:  time.Now(),
		Type:       types.VoteTypeApprove,
	}
	
	wc.mu.Lock()
	wc.votes[wave][string(proposal.ID)] = vote
	wc.mu.Unlock()
	
	log.Printf("Wave Consensus: Voted for proposal in wave %d", wave)
	
	// Check if we have enough votes to commit
	wc.checkCommitCondition(wave)
}

// validateProposal validates a proposal
func (wc *WaveConsensus) validateProposal(proposal *types.Proposal) bool {
	// Check if block exists in DAG
	if _, err := wc.dagTransport.GetBlock(proposal.BlockHash); err != nil {
		return false
	}
	
	// Check if proposer is valid leader for this wave
	if !wc.isValidLeader(proposal.Proposer, proposal.Wave) {
		return false
	}
	
	return true
}

// isValidLeader checks if the proposer is the valid leader for the wave
func (wc *WaveConsensus) isValidLeader(proposer types.Address, wave types.Wave) bool {
	leaderIndex := int(wave) % len(wc.validators)
	return wc.validators[leaderIndex] == proposer
}

// checkCommitCondition checks if the commit condition is met for a wave
func (wc *WaveConsensus) checkCommitCondition(wave types.Wave) {
	wc.mu.RLock()
	proposal := wc.proposals[wave]
	votes := wc.votes[wave]
	wc.mu.RUnlock()
	
	if proposal == nil {
		return
	}
	
	// Count votes for this proposal
	voteCount := 0
	for _, vote := range votes {
		if vote.Type == types.VoteTypeApprove {
			voteCount++
		}
	}
	
	// Check if we have f+1 votes (honest majority)
	if voteCount >= wc.f+1 {
		wc.commitProposal(wave, proposal)
	}
}

// commitProposal commits a proposal and all its causal history
func (wc *WaveConsensus) commitProposal(wave types.Wave, proposal *types.Proposal) {
	// Get all blocks in causal history
	causalBlocks := wc.dagTransport.GetBlocksInCausalHistory([]*types.Block{proposal.Block})
	
	wc.mu.Lock()
	// Commit blocks in topological order
	for _, block := range causalBlocks {
		blockHash := string(block.ComputeHash())
		if _, exists := wc.committedBlocks[blockHash]; !exists {
			wc.committedBlocks[blockHash] = block
			wc.commitOrder = append(wc.commitOrder, block)
		}
	}
	
	// 🔥 CLEANUP OLD WAVE DATA TO PREVENT MEMORY LEAKS
	wc.cleanupOldWaveData(wave)
	
	// 🔥 LIMIT COMMITTED BLOCKS TO PREVENT INFINITE GROWTH
	wc.limitCommittedBlocks()
	
	wc.mu.Unlock()
	
	log.Printf("Wave Consensus: Committed proposal for wave %d with %d blocks in causal history", 
		wave, len(causalBlocks))
	
	// Advance to next wave
	wc.advanceWave()
}

// 🧹 cleanupOldWaveData removes old wave data to prevent memory leaks
func (wc *WaveConsensus) cleanupOldWaveData(currentWave types.Wave) {
	const keepWaves = 5 // Keep only last 5 waves
	
	// Clean up old waves
	for wave := range wc.proposals {
		if wave < currentWave-keepWaves {
			delete(wc.proposals, wave)
			delete(wc.votes, wave)
			delete(wc.complaints, wave)
		}
	}
}

// 🧹 limitCommittedBlocks limits the number of committed blocks to prevent memory explosion
func (wc *WaveConsensus) limitCommittedBlocks() {
	const maxCommittedBlocks = 100 // Keep only last 100 committed blocks
	
	if len(wc.commitOrder) > maxCommittedBlocks {
		// Remove oldest blocks
		oldestBlocks := wc.commitOrder[:len(wc.commitOrder)-maxCommittedBlocks]
		for _, block := range oldestBlocks {
			blockHash := string(block.ComputeHash())
			delete(wc.committedBlocks, blockHash)
		}
		
		// Keep only recent blocks
		wc.commitOrder = wc.commitOrder[len(wc.commitOrder)-maxCommittedBlocks:]
		
		log.Printf("🧹 Wave Consensus Cleanup: Limited to %d committed blocks", maxCommittedBlocks)
	}
}

// handleWaveTimeout handles wave timeout (complaint mechanism)
func (wc *WaveConsensus) handleWaveTimeout(wave types.Wave) {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	
	// Create complaint
	complaint := &types.Complaint{
		ID:        types.Hash("complaint_" + string(wave)),
		Validator: wc.validatorID,
		Round:     0, // Round doesn't matter for complaints
		Wave:      wave,
		Timestamp: time.Now(),
		Reason:    "Wave timeout",
	}
	
	wc.complaints[wave][string(complaint.ID)] = complaint
	
	log.Printf("Wave Consensus: Wave %d timed out, created complaint", wave)
	
	// Check if we have 2f+1 complaints to advance to next wave
	if len(wc.complaints[wave]) >= 2*wc.f+1 {
		log.Printf("Wave Consensus: Enough complaints for wave %d, advancing", wave)
		wc.advanceWave()
	}
}

// advanceWave advances to the next wave
func (wc *WaveConsensus) advanceWave() {
	wc.mu.Lock()
	wc.currentWave++
	nextWave := wc.currentWave
	wc.mu.Unlock()
	
	log.Printf("Wave Consensus: Advanced to wave %d", nextWave)
	wc.startNewWave()
}

// GetCurrentWave returns the current consensus wave
func (wc *WaveConsensus) GetCurrentWave() types.Wave {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	return wc.currentWave
}

// GetCommittedBlocks returns the committed blocks in order
func (wc *WaveConsensus) GetCommittedBlocks() []*types.Block {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	
	// Return a copy of the commit order
	result := make([]*types.Block, len(wc.commitOrder))
	copy(result, wc.commitOrder)
	return result
}

// GetBlockCommitStatus returns whether a block is committed
func (wc *WaveConsensus) GetBlockCommitStatus(blockHash types.Hash) bool {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	
	_, exists := wc.committedBlocks[string(blockHash)]
	return exists
} 