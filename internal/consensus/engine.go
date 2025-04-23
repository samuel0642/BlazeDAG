package consensus

import (
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
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
}

// NewEngine creates a new consensus engine with the given configuration
func NewEngine(config *Config) *Engine {
	return &Engine{
		config:    config,
		proposals: make(map[string]*types.Proposal),
		votes:     make(map[string][]*types.Vote),
	}
}

// HandleProposal handles a new block proposal
func (e *Engine) HandleProposal(proposal *types.Proposal) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.proposals[string(proposal.BlockHash)] = proposal
	return nil
}

// HandleVote handles a new vote on a proposal
func (e *Engine) HandleVote(vote *types.Vote) error {
	e.mu.Lock()
	defer e.mu.Unlock()

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
			validators[types.Address(vote.Validator)] = true
		}
	}

	// Check if we have enough votes (2/3 + 1)
	return len(validators) >= (2*e.config.TotalValidators/3 + 1)
}

// ConsensusEngine manages the consensus process
type ConsensusEngine struct {
	dag          *core.DAG
	stateManager *state.StateManager
	executor     *core.EVMExecutor
	isValidator  bool
	running      bool
	config       *Config
	proposals    map[string]*types.Proposal
	votes        map[string][]*types.Vote
	mu           sync.RWMutex
	validators   []types.Address
	nodeID       types.Address
	waveManager  *WaveManager
	currentWave  *Wave
}

// NewConsensusEngine creates a new consensus engine
func NewConsensusEngine(dag *core.DAG, stateManager *state.StateManager, executor *core.EVMExecutor, isValidator bool, nodeID types.Address) *ConsensusEngine {
	config := DefaultConfig()
	return &ConsensusEngine{
		dag:          dag,
		stateManager: stateManager,
		executor:     executor,
		isValidator:  isValidator,
		config:       config,
		proposals:    make(map[string]*types.Proposal),
		votes:        make(map[string][]*types.Vote),
		validators:   make([]types.Address, 0),
		nodeID:       nodeID,
		waveManager:  NewWaveManager(config),
		currentWave:  nil,
	}
}

// Start starts the consensus engine
func (ce *ConsensusEngine) Start() error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if ce.running {
		return nil
	}

	ce.running = true
	return nil
}

// Stop stops the consensus engine
func (ce *ConsensusEngine) Stop() error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return nil
	}

	ce.running = false
	return nil
}

// HandleBlock handles a new block
func (ce *ConsensusEngine) HandleBlock(block *types.Block) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return nil
	}

	// Validate block
	if err := ce.validateBlock(block); err != nil {
		return err
	}

	// Add block to DAG
	if err := ce.dag.AddBlock(block); err != nil {
		return err
	}

	// Create proposal
	proposal := &types.Proposal{
		ID:        []byte(block.ComputeHash()),
		Wave:      block.Header.Wave,
		Round:     block.Header.Round,
		Block:     block,
		Proposer:  types.Address(block.Header.Validator),
		Timestamp: time.Now(),
		Status:    types.WaveStatusProposing,
	}

	// Handle proposal
	ce.proposals[string(proposal.ID)] = proposal
	return nil
}

// HandleVote handles a vote
func (ce *ConsensusEngine) HandleVote(vote *types.Vote) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return nil
	}

	// Validate vote
	if err := ce.validateVote(vote); err != nil {
		return err
	}

	// Add vote
	ce.votes[string(vote.ProposalID)] = append(ce.votes[string(vote.ProposalID)], vote)

	// Check quorum
	if ce.HasQuorum() {
		// Create certificate
		cert := &types.Certificate{
			BlockHash:  []byte(vote.ProposalID),
			Wave:      vote.Wave,
			Round:     vote.Round,
			Timestamp: time.Now(),
		}

		// Add certificate to block
		block, err := ce.dag.GetBlock(string(vote.ProposalID))
		if err == nil {
			block.Certificate = cert
		}
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

	// Validate certificate
	if err := ce.validateCertificate(cert); err != nil {
		return err
	}

	// Get block
	block, err := ce.dag.GetBlock(string(cert.BlockHash))
	if err != nil {
		return err
	}

	// Add certificate to block
	block.Certificate = cert

	// Update state
	return ce.stateManager.CommitBlock(block)
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
	_, err := ce.dag.GetBlock(string(vote.ProposalID))
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
	_, err := ce.dag.GetBlock(string(cert.BlockHash))
	if err != nil {
		return err
	}

	return nil
}

// validateReferences validates block references
func (ce *ConsensusEngine) validateReferences(block *types.Block) error {
	// Check parent exists
	if len(block.Header.ParentHash) > 0 {
		_, err := ce.dag.GetBlock(string(block.Header.ParentHash))
		if err != nil {
			return err
		}
	}

	// Check other references
	for _, ref := range block.Body.References {
		_, err := ce.dag.GetBlock(string(ref.BlockHash))
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

// GetCurrentWave returns the current wave
func (ce *ConsensusEngine) GetCurrentWave() *Wave {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.currentWave
}

// StartNewWave starts a new consensus wave
func (ce *ConsensusEngine) StartNewWave() error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	wave, err := ce.waveManager.StartNewWave()
	if err != nil {
		return err
	}

	leaderIndex := uint64(wave.Number) % uint64(len(ce.validators))
	wave.Leader = ce.validators[leaderIndex]
	ce.currentWave = wave

	return nil
}

// SelectLeader selects a leader for the current wave
func (ce *ConsensusEngine) SelectLeader() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return
	}

	// Get current wave
	currentWave := ce.GetCurrentWave()
	if currentWave == nil {
		return
	}

	// Select leader based on wave number
	leaderIndex := int(currentWave.Number) % len(ce.validators)
	currentWave.Leader = ce.validators[leaderIndex]
}

// IsLeader checks if this node is the current wave leader
func (ce *ConsensusEngine) IsLeader() bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	if !ce.running || !ce.isValidator {
		return false
	}

	currentWave := ce.GetCurrentWave()
	if currentWave == nil {
		return false
	}

	return currentWave.Leader == ce.nodeID
}

// CreateBlock creates a new block
func (ce *ConsensusEngine) CreateBlock(wave types.Wave, round types.Round) (*types.Block, error) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Get latest block
	latestBlock, err := ce.dag.GetBlockByHeight(ce.dag.GetHeight())
	if err != nil {
		return nil, err
	}

	// Create new block
	block := &types.Block{
		Header: &types.BlockHeader{
			Version:    1,
			Wave:       wave,
			Round:      round,
			Height:     ce.dag.GetHeight() + 1,
			ParentHash: []byte(latestBlock.ComputeHash()),
			StateRoot:  nil, // Will be set after execution
			Validator:  []byte(ce.nodeID),
			Timestamp:  time.Now(),
		},
		Body: &types.BlockBody{
			Transactions: make([]types.Transaction, 0),
			Receipts:     make([]types.Receipt, 0),
			Events:       make([]types.Event, 0),
			References:   make([]types.Reference, 0),
		},
		Timestamp: time.Now(),
	}

	return block, nil
}

// ProcessVote processes a vote from a validator
func (ce *ConsensusEngine) ProcessVote(validator types.Address, vote bool) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if ce.currentWave == nil {
		return ErrNoActiveWave
	}

	ce.currentWave.Votes[validator] = vote

	// Check if we have enough votes (2/3 + 1)
	validators := make(map[types.Address]bool)
	for v, voted := range ce.currentWave.Votes {
		if voted {
			validators[v] = true
		}
	}

	if len(validators) >= (2*ce.config.TotalValidators/3 + 1) {
		ce.currentWave.Status = types.WaveStatusFinalized
	}

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

// HandleWaveTimeout handles wave timeout
func (ce *ConsensusEngine) HandleWaveTimeout() {
	ce.waveManager.HandleWaveTimeout()
}

// FinalizeWave finalizes the current wave
func (ce *ConsensusEngine) FinalizeWave() {
	ce.waveManager.FinalizeWave()
}

// ProcessTimeout handles wave timeout
func (ce *ConsensusEngine) ProcessTimeout() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if ce.currentWave != nil && ce.currentWave.Status == types.WaveStatusProposing {
		ce.currentWave.Status = types.WaveStatusFailed
	}
}

// HasQuorum checks if a proposal has received enough votes to reach quorum
func (ce *ConsensusEngine) HasQuorum() bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	// Count unique validators who voted
	validators := make(map[types.Address]bool)
	for _, votes := range ce.votes {
		for _, vote := range votes {
			validators[types.Address(vote.Validator)] = true
		}
	}

	// Check if we have enough votes (2/3 + 1)
	return len(validators) >= (2*ce.config.TotalValidators/3 + 1)
} 