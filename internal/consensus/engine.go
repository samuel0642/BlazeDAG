package consensus

import (
	"sync"
	"time"
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
	Round     int
	Sender    string
	Timestamp time.Time
}

// Vote represents a vote on a proposal
type Vote struct {
	BlockHash string
	Round     int
	Sender    string
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
func (e *Engine) HasQuorum(blockHash string, round int) bool {
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
			validators[vote.Sender] = true
		}
	}

	// Check if we have enough votes (2f+1)
	return len(validators) >= (2*e.config.FaultTolerance + 1)
} 