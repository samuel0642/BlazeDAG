package consensus

import (
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// Config represents the configuration for the consensus engine
type Config struct {
	TotalValidators int
	FaultTolerance  int
	RoundDuration   time.Duration
	WaveTimeout     time.Duration
}

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

// Engine represents the consensus engine
type Engine struct {
	config    Config
	proposals map[string]*Proposal
	votes     map[string][]*Vote
	mu        sync.RWMutex
}

// NewEngine creates a new consensus engine with the given configuration
func NewEngine(config Config) *Engine {
	return &Engine{
		config:    config,
		proposals: make(map[string]*Proposal),
		votes:     make(map[string][]*Vote),
	}
}

// HandleProposal handles a new block proposal
func (e *Engine) HandleProposal(proposal *Proposal) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.proposals[proposal.BlockHash] = proposal
	return nil
}

// HandleVote handles a new vote on a proposal
func (e *Engine) HandleVote(vote *Vote) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.votes[vote.BlockHash] = append(e.votes[vote.BlockHash], vote)
	return nil
}

// HasQuorum checks if a proposal has received enough votes to reach quorum
func (e *Engine) HasQuorum(blockHash string, round uint64) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	votes := e.votes[blockHash]
	if len(votes) == 0 {
		return false
	}

	// Count unique validators who voted
	validators := make(map[string]bool)
	for _, vote := range votes {
		if vote.Round == round {
			validators[string(vote.Validator)] = true
		}
	}

	// Check if we have enough votes (2f+1)
	return len(validators) >= (2*e.config.FaultTolerance + 1)
}

// ConsensusEngine represents the consensus engine
type ConsensusEngine struct {
	dag          *core.DAG
	stateManager *state.StateManager
	executor     *core.EVMExecutor
	isValidator  bool
	running      bool
	config       Config
	proposals    map[string]*Proposal
	votes        map[string][]*Vote
	mu           sync.RWMutex
}

// NewConsensusEngine creates a new consensus engine
func NewConsensusEngine(dag *core.DAG, stateManager *state.StateManager, executor *core.EVMExecutor, isValidator bool) *ConsensusEngine {
	return &ConsensusEngine{
		dag:          dag,
		stateManager: stateManager,
		executor:     executor,
		isValidator:  isValidator,
		config: Config{
			TotalValidators: 100,
			FaultTolerance:  33,
			RoundDuration:   time.Second * 5,
			WaveTimeout:     time.Second * 30,
		},
		proposals: make(map[string]*Proposal),
		votes:     make(map[string][]*Vote),
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
	// TODO: Implement block handling
	return nil
}

// HandleVote handles a vote
func (ce *ConsensusEngine) HandleVote(vote *Vote) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	ce.votes[vote.BlockHash] = append(ce.votes[vote.BlockHash], vote)
	return nil
}

// HandleCertificate handles a certificate
func (ce *ConsensusEngine) HandleCertificate(cert *types.Certificate) error {
	// TODO: Implement certificate handling
	return nil
}

// HasQuorum checks if a proposal has received enough votes to reach quorum
func (ce *ConsensusEngine) HasQuorum(blockHash string, round uint64) bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	votes := ce.votes[blockHash]
	if len(votes) == 0 {
		return false
	}

	// Count unique validators who voted
	validators := make(map[string]bool)
	for _, vote := range votes {
		if vote.Round == round {
			validators[string(vote.Validator)] = true
		}
	}

	// Check if we have enough votes (2f+1)
	return len(validators) >= (2*ce.config.FaultTolerance + 1)
}

// Certificate represents a block certificate
type Certificate struct {
	BlockHash   string
	Signatures  []string
	Round       uint64
	Wave        uint64
	Validators  []string
} 