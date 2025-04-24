package consensus

import (
	"errors"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

var (
	ErrInvalidWaveState = errors.New("invalid wave state")
)

// WaveState represents a consensus wave state
type WaveState struct {
	mu sync.RWMutex

	// Wave metadata
	Number    types.Wave
	StartTime time.Time
	EndTime   time.Time
	Status    types.WaveStatus
	Leader    types.Address

	// Wave state
	proposals map[string]*types.Proposal
	votes     map[string]map[types.Address]*types.Vote
	quorum    int
}

// NewWaveState creates a new wave state
func NewWaveState(number types.Wave, timeout time.Duration, quorum int) *WaveState {
	return &WaveState{
		Number:    number,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(timeout),
		Status:    types.WaveStatusProposing,
		proposals: make(map[string]*types.Proposal),
		votes:     make(map[string]map[types.Address]*types.Vote),
		quorum:    quorum,
	}
}

// GetWaveNumber returns the current wave number
func (w *WaveState) GetWaveNumber() types.Wave {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Number
}

// GetProposals returns all proposals in the wave
func (w *WaveState) GetProposals() map[string]*types.Proposal {
	w.mu.RLock()
	defer w.mu.RUnlock()
	proposals := make(map[string]*types.Proposal)
	for id, proposal := range w.proposals {
		proposals[id] = proposal
	}
	return proposals
}

// IsTimedOut checks if the wave has timed out
func (w *WaveState) IsTimedOut() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return time.Now().After(w.EndTime)
}

// AddVote adds a vote to the wave
func (w *WaveState) AddVote(validator types.Address) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.votes[string(validator)] = make(map[types.Address]*types.Vote)
}

// GetVotes returns all votes
func (w *WaveState) GetVotes() map[types.Address]bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	votes := make(map[types.Address]bool)
	for addr := range w.votes {
		votes[types.Address(addr)] = true
	}
	return votes
}

// GetStatus returns the wave status
func (w *WaveState) GetStatus() types.WaveStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Status
}

// SetStatus sets the wave status
func (w *WaveState) SetStatus(status types.WaveStatus) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Status = status
}

// GetLeader returns the wave leader
func (w *WaveState) GetLeader() types.Address {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Leader
}

// SetLeader sets the wave leader
func (w *WaveState) SetLeader(leader types.Address) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Leader = leader
}

// AddProposal adds a proposal to the wave state
func (w *WaveState) AddProposal(proposal *types.Proposal) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Status != types.WaveStatusProposing {
		return ErrInvalidWaveState
	}

	w.proposals[string(proposal.ID)] = proposal
	return nil
}

// AddProposalVote adds a vote for a specific proposal
func (w *WaveState) AddProposalVote(vote *types.Vote) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Status != types.WaveStatusVoting {
		return ErrInvalidWaveState
	}

	proposalID := string(vote.ProposalID)
	if _, exists := w.votes[proposalID]; !exists {
		w.votes[proposalID] = make(map[types.Address]*types.Vote)
	}

	w.votes[proposalID][vote.Validator] = vote
	return nil
}

// GetProposal returns a proposal by its ID
func (w *WaveState) GetProposal(id types.Hash) *types.Proposal {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.proposals[string(id)]
}

// GetProposalVotes returns all votes for a proposal
func (w *WaveState) GetProposalVotes(proposalID types.Hash) map[types.Address]*types.Vote {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.votes[string(proposalID)]
}

// HasQuorum checks if a proposal has reached quorum
func (w *WaveState) HasQuorum(proposalID types.Hash) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	votes := w.votes[string(proposalID)]
	return len(votes) >= w.quorum
} 