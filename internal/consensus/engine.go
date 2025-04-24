package consensus

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

var (
	ErrNoActiveWave = errors.New("no active wave")
)

// Proposal represents a block proposal in the consensus process
type Proposal struct {
	BlockHash string
	Round     uint64
	Sender    string
	Timestamp time.Time
}

// Vote represents a vote on a proposal
type Vote struct {
	BlockHash string
	Round     uint64
	Wave      uint64
	Validator []byte
	Timestamp time.Time
}

// Wave represents a consensus wave
type Wave struct {
	Number    types.Wave
	StartTime time.Time
	EndTime   time.Time
	Status    types.WaveStatus
	Leader    types.Address
	Votes     map[types.Address]bool
}

// Engine represents the consensus engine
type Engine struct {
	config    *Config
	proposals map[string]*types.Proposal
	votes     map[string][]*types.Vote
	mu        sync.RWMutex
	logger    *log.Logger
}

// NewEngine creates a new consensus engine with the given configuration
func NewEngine(config *Config) *Engine {
	return &Engine{
		config:    config,
		proposals: make(map[string]*types.Proposal),
		votes:     make(map[string][]*types.Vote),
		logger:    log.New(log.Writer(), "[Consensus] ", log.LstdFlags),
	}
}

// HandleProposal handles a new block proposal
func (e *Engine) HandleProposal(proposal *types.Proposal) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Printf("Received proposal for block %s in wave %d, round %d", 
		proposal.BlockHash, proposal.Wave, proposal.Round)
	e.proposals[string(proposal.BlockHash)] = proposal
	return nil
}

// HandleVote handles a new vote on a proposal
func (e *Engine) HandleVote(vote *types.Vote) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Printf("Received vote for block %s from validator %s in wave %d, round %d",
		vote.BlockHash, vote.Validator, vote.Wave, vote.Round)
	e.votes[string(vote.BlockHash)] = append(e.votes[string(vote.BlockHash)], vote)
	return nil
}

// HasQuorum checks if a proposal has received enough votes to reach quorum
func (e *Engine) HasQuorum() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Count unique validators who voted
	validators := make(map[types.Address]bool)
	for _, votes := range e.votes {
		for _, vote := range votes {
			validators[vote.Validator] = true
		}
	}

	// Check if we have enough votes (2/3 + 1)
	hasQuorum := len(validators) >= (2*e.config.TotalValidators/3 + 1)
	if hasQuorum {
		e.logger.Printf("Quorum reached with %d validators", len(validators))
	}
	return hasQuorum
}

// ConsensusEngine represents the consensus engine
type ConsensusEngine struct {
	mu            sync.RWMutex
	running       bool
	nodeID        types.Address
	dag           *core.DAG
	stateManager  *state.StateManager
	waveManager   *WaveManager
	currentWave   *WaveState
	currentRound  types.Round
	currentHeight types.BlockNumber
	currentLeader types.Address
	votes         map[string]map[types.Address]bool
	config        *Config
	proposals     map[string]*types.Proposal
	validators    []types.Address
	waves         map[types.Wave]*WaveState
	logger        *log.Logger
}

// NewConsensusEngine creates a new consensus engine
func NewConsensusEngine(dag *core.DAG, stateManager *state.StateManager, nodeID types.Address, config *Config) *ConsensusEngine {
	return &ConsensusEngine{
		dag:           dag,
		stateManager:  stateManager,
		nodeID:        nodeID,
		config:        config,
		proposals:     make(map[string]*types.Proposal),
		votes:         make(map[string]map[types.Address]bool),
		validators:    make([]types.Address, 0),
		waveManager:   NewWaveManager(config),
		currentWave:   nil,
		currentRound:  0,
		currentHeight: 0,
		waves:         make(map[types.Wave]*WaveState),
		logger:        log.New(log.Writer(), "[Consensus] ", log.LstdFlags),
	}
}

// Start starts the consensus engine
func (ce *ConsensusEngine) Start() error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if ce.running {
		return nil
	}

	// Initialize validator set
	ce.validators = ce.config.ValidatorSet
	ce.logger.Printf("Initialized validator set with %d validators", len(ce.validators))

	// Start with wave 0
	ce.currentWave = &WaveState{
		Number:    0,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(ce.config.WaveTimeout),
		Status:    types.WaveStatusProposing,
		Votes:     make(map[types.Address]bool),
	}

	// Select initial leader
	ce.selectLeader()
	ce.logger.Printf("Selected initial leader: %s", ce.currentLeader)

	ce.running = true
	ce.logger.Printf("Consensus engine started for node %s", ce.nodeID)

	// Start wave timer
	go ce.waveTimer()

	return nil
}

// waveTimer handles wave timeouts
func (ce *ConsensusEngine) waveTimer() {
	for ce.running {
		time.Sleep(ce.config.WaveTimeout)
		ce.mu.Lock()
		if ce.currentWave != nil {
			ce.logger.Printf("Wave %d timed out", ce.currentWave.Number)
			ce.currentWave.Number++
			ce.selectLeader()
			ce.logger.Printf("Selected new leader for wave %d: %s", 
				ce.currentWave.Number, ce.currentLeader)
		}
		ce.mu.Unlock()
	}
}

// selectLeader selects a leader for the current wave
func (ce *ConsensusEngine) selectLeader() {
	if len(ce.validators) == 0 {
		ce.logger.Printf("No validators available for leader selection")
		return
	}

	// Select leader based on wave number
	leaderIndex := int(ce.currentWave.Number) % len(ce.validators)
	ce.currentLeader = ce.validators[leaderIndex]
	ce.logger.Printf("Selected leader %s for wave %d", 
		ce.currentLeader, ce.currentWave.Number)
}

// IsLeader checks if the current node is the leader
func (ce *ConsensusEngine) IsLeader() bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	if !ce.running || ce.currentWave == nil {
		return false
	}

	isLeader := ce.currentLeader == ce.nodeID
	if isLeader {
		ce.logger.Printf("Node %s is leader for wave %d", 
			ce.nodeID, ce.currentWave.Number)
	}
	return isLeader
}

// GetCurrentWave returns the current wave
func (ce *ConsensusEngine) GetCurrentWave() types.Wave {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	if ce.currentWave == nil {
		return 0
	}
	return ce.currentWave.Number
}

// HandleBlock handles a block
func (ce *ConsensusEngine) HandleBlock(block *types.Block) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return nil
	}

	ce.logger.Printf("Processing block %s in wave %d, round %d", 
		block.ComputeHash(), block.Header.Wave, block.Header.Round)

	// Validate block
	if err := ce.validateBlock(block); err != nil {
		ce.logger.Printf("Block validation failed: %v", err)
		return err
	}

	// Add block to DAG
	if err := ce.dag.AddBlock(block); err != nil {
		ce.logger.Printf("Failed to add block to DAG: %v", err)
		return err
	}

	// Create proposal
	proposal := &types.Proposal{
		ID:        block.ComputeHash(),
		BlockHash: block.ComputeHash(),
		Wave:      block.Header.Wave,
		Round:     block.Header.Round,
		Block:     block,
		Proposer:  types.Address(block.Header.Validator),
		Timestamp: time.Now(),
		Status:    types.ProposalStatusPending,
	}

	// Handle proposal
	ce.proposals[string(proposal.ID)] = proposal
	ce.logger.Printf("Created proposal for block %s", proposal.ID)
	return nil
}

// HandleVote handles a vote from a validator
func (ce *ConsensusEngine) HandleVote(validator types.Address, proposalID string) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if ce.currentWave == nil {
		return ErrNoActiveWave
	}

	ce.logger.Printf("Received vote from validator %s for proposal %s in wave %d", 
		validator, proposalID, ce.currentWave.Number)

	// Initialize votes map for proposal if not exists
	if _, exists := ce.votes[proposalID]; !exists {
		ce.votes[proposalID] = make(map[types.Address]bool)
	}

	// Record vote
	ce.votes[proposalID][validator] = true

	// Check if we have enough votes (2/3 + 1)
	if ce.HasQuorum() {
		ce.logger.Printf("Quorum reached for proposal %s in wave %d", 
			proposalID, ce.currentWave.Number)
		ce.currentWave.Number++
	}

	return nil
}

// HandleCertificate handles a certificate
func (ce *ConsensusEngine) HandleCertificate(cert *types.Certificate) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return nil
	}

	ce.logger.Printf("Processing certificate for block %s in wave %d, round %d", 
		cert.BlockHash, cert.Wave, cert.Round)

	// Validate certificate
	if err := ce.validateCertificate(cert); err != nil {
		ce.logger.Printf("Certificate validation failed: %v", err)
		return err
	}

	// Get block
	block, err := ce.dag.GetBlock(cert.BlockHash)
	if err != nil {
		ce.logger.Printf("Failed to get block for certificate: %v", err)
		return err
	}

	// Add certificate to block
	block.Certificate = cert

	// Update state
	if err := ce.stateManager.CommitBlock(block); err != nil {
		ce.logger.Printf("Failed to commit block: %v", err)
		return err
	}

	ce.logger.Printf("Block %s committed with certificate", cert.BlockHash)
	return nil
}

// validateBlock validates a block
func (ce *ConsensusEngine) validateBlock(block *types.Block) error {
	// Check block structure
	if block == nil || block.Header == nil || block.Body == nil {
		return types.ErrInvalidBlock
	}

	// Check block references
	if err := ce.validateReferences(block); err != nil {
		return err
	}

	// Check transactions
	if err := ce.validateTransactions(block); err != nil {
		return err
	}

	return nil
}

// validateVote validates a vote
func (ce *ConsensusEngine) validateVote(vote *types.Vote) error {
	// Check vote structure
	if vote == nil || len(vote.Validator) == 0 {
		return types.ErrInvalidVote
	}

	// Check if block exists
	_, err := ce.dag.GetBlock(vote.BlockHash)
	if err != nil {
		return err
	}

	return nil
}

// validateCertificate validates a certificate
func (ce *ConsensusEngine) validateCertificate(cert *types.Certificate) error {
	// Check certificate structure
	if cert == nil || len(cert.BlockHash) == 0 {
		return types.ErrInvalidCertificate
	}

	// Check if block exists
	_, err := ce.dag.GetBlock(cert.BlockHash)
	if err != nil {
		return err
	}

	return nil
}

// validateReferences validates block references
func (ce *ConsensusEngine) validateReferences(block *types.Block) error {
	// Check parent exists
	if len(block.Header.ParentHash) > 0 {
		_, err := ce.dag.GetBlock(block.Header.ParentHash)
		if err != nil {
			return err
		}
	}

	// Check other references
	for _, ref := range block.Header.References {
		_, err := ce.dag.GetBlock(ref.BlockHash)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateTransactions validates block transactions
func (ce *ConsensusEngine) validateTransactions(block *types.Block) error {
	for _, tx := range block.Body.Transactions {
		// Check transaction structure
		if len(tx.From) == 0 || len(tx.To) == 0 {
			return types.ErrInvalidTransaction
		}

		// Check sender balance
		sender, err := ce.stateManager.GetAccount(string(tx.From))
		if err != nil {
			return err
		}

		if sender.Balance < tx.Value {
			return types.ErrInsufficientBalance
		}

		// Check nonce
		if sender.Nonce != tx.Nonce {
			return types.ErrInvalidNonce
		}
	}

	return nil
}

// StartNewWave starts a new consensus wave
func (ce *ConsensusEngine) StartNewWave() error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	wave, err := ce.waveManager.StartNewWave()
	if err != nil {
		ce.logger.Printf("Failed to start new wave: %v", err)
		return err
	}

	leaderIndex := uint64(wave.Number) % uint64(len(ce.validators))
	wave.Leader = ce.validators[leaderIndex]
	ce.currentWave = wave

	ce.logger.Printf("Started new wave %d with leader %s", 
		wave.Number, wave.Leader)
	return nil
}

// CreateBlock creates a new block
func (ce *ConsensusEngine) CreateBlock() (*types.Block, error) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return nil, errors.New("engine is not running")
	}

	ce.logger.Printf("Creating new block in wave %d, round %d", 
		ce.currentWave.Number, ce.currentRound)

	// Create block header
	header := &types.BlockHeader{
		Version:    1,
		Timestamp:  time.Now(),
		Round:      ce.currentRound,
		Wave:       ce.currentWave.Number,
		Height:     ce.getNextHeight(),
		ParentHash: ce.getParentHash(),
		References: ce.selectReferences(),
		StateRoot:  ce.calculateStateRoot(),
		Validator:  ce.nodeID,
	}

	// Create block body
	body := &types.BlockBody{
		Transactions: make([]*types.Transaction, 0),
		Receipts:     make([]*types.Receipt, 0),
		Events:       make([]*types.Event, 0),
	}

	// Create block
	block := &types.Block{
		Header: header,
		Body:   body,
	}

	// Sign block
	if err := ce.signBlock(block); err != nil {
		ce.logger.Printf("Failed to sign block: %v", err)
		return nil, err
	}

	ce.logger.Printf("Created block %s in wave %d, round %d", 
		block.ComputeHash(), ce.currentWave.Number, ce.currentRound)
	return block, nil
}

// ProcessVote processes a vote from a validator
func (ce *ConsensusEngine) ProcessVote(validator types.Address, vote bool) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if ce.currentWave == nil {
		return fmt.Errorf("no active wave")
	}

	ce.logger.Printf("Processing vote from validator %s in wave %d", 
		validator, ce.currentWave.Number)

	// Record vote
	ce.currentWave.Votes[validator] = vote
	ce.processVote(vote)

	return nil
}

// verifyVote verifies a vote
func (ce *ConsensusEngine) verifyVote(vote *types.Vote) error {
	// TODO: Implement vote verification
	return nil
}

// GetValidators gets all validators
func (ce *ConsensusEngine) GetValidators() []types.Address {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	validators := make([]types.Address, len(ce.validators))
	copy(validators, ce.validators)
	return validators
}

// ProcessTimeout handles wave timeout
func (ce *ConsensusEngine) ProcessTimeout() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if ce.currentWave != nil {
		ce.logger.Printf("Processing timeout for wave %d", ce.currentWave.Number)
		ce.currentWave.Number = 0
	}
}

// HasQuorum checks if there is a quorum of votes
func (ce *ConsensusEngine) HasQuorum() bool {
	validators := make(map[types.Address]bool)
	for validator, voted := range ce.currentWave.Votes {
		if voted {
			validators[validator] = true
		}
	}
	hasQuorum := len(validators) >= (2*ce.config.TotalValidators/3 + 1)
	if hasQuorum {
		ce.logger.Printf("Quorum reached with %d validators in wave %d", 
			len(validators), ce.currentWave.Number)
	}
	return hasQuorum
}

// getNextHeight returns the next block height
func (ce *ConsensusEngine) getNextHeight() types.BlockNumber {
	return ce.currentHeight + 1
}

// getParentHash returns the parent block hash
func (ce *ConsensusEngine) getParentHash() types.Hash {
	if ce.currentHeight == 0 {
		return types.Hash{}
	}
	block, err := ce.dag.GetBlockByHeight(ce.currentHeight)
	if err != nil {
		return types.Hash{}
	}
	return block.ComputeHash()
}

// selectReferences selects references for the new block
func (ce *ConsensusEngine) selectReferences() []*types.Reference {
	// TODO: Implement reference selection logic
	return make([]*types.Reference, 0)
}

// calculateStateRoot calculates the state root
func (ce *ConsensusEngine) calculateStateRoot() types.Hash {
	// TODO: Implement state root calculation
	return types.Hash{}
}

// signBlock signs a block
func (ce *ConsensusEngine) signBlock(block *types.Block) error {
	// TODO: Implement block signing
	block.Header.Signature = types.Signature{
		Validator:  ce.nodeID,
		Signature:  []byte("dummy_signature"),
		Timestamp:  time.Now(),
	}
	return nil
}

// BroadcastBlock broadcasts a block to all peers
func (ce *ConsensusEngine) BroadcastBlock(block *types.Block) error {
	// TODO: Implement block broadcasting
	return nil
}

func (ce *ConsensusEngine) createProposal() *types.Proposal {
	return &types.Proposal{
		Timestamp:  time.Now(),
		Round:      ce.currentRound,
		Wave:       ce.currentWave.Number,
		BlockHash:  ce.getParentHash(),
		Proposer:   ce.nodeID,
	}
}

func (ce *ConsensusEngine) processVote(vote bool) {
	if vote {
		ce.currentWave.Status = types.WaveStatusProposing
	} else {
		ce.currentWave.Status = types.WaveStatusFailed
	}

	// Check if we have enough votes (2/3 + 1)
	validators := make(map[types.Address]bool)
	for validator, voted := range ce.currentWave.Votes {
		if voted {
			validators[validator] = true
		}
	}

	if len(validators) >= (2*ce.config.TotalValidators/3 + 1) {
		ce.currentWave.Status = types.WaveStatusFinalized
	}
}

func (ce *ConsensusEngine) advanceWave() {
	if ce.currentWave.Status != types.WaveStatusFinalized {
		ce.currentWave.Number++
		ce.currentWave.Status = types.WaveStatusProposing
	}
}

func (ce *ConsensusEngine) resetWave() {
	if ce.currentWave.Status != types.WaveStatusFinalized {
		ce.currentWave.Number = 0
		ce.currentWave.Status = types.WaveStatusProposing
	}
} 