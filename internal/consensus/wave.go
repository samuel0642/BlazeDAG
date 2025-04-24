package consensus

import (
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// WaveState represents a consensus wave state
type WaveState struct {
	Number      types.Wave
	StartTime   time.Time
	EndTime     time.Time
	Status      types.WaveStatus
	Leader      types.Address
	Votes       map[types.Address]bool
}

// NewWaveState creates a new wave state
func NewWaveState(number types.Wave, timeout time.Duration) *WaveState {
	return &WaveState{
		Number:    number,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(timeout),
		Status:    types.WaveStatusProposing,
		Votes:     make(map[types.Address]bool),
	}
}

// IsTimedOut checks if the wave has timed out
func (w *WaveState) IsTimedOut() bool {
	return time.Now().After(w.EndTime)
}

// AddVote adds a vote to the wave
func (w *WaveState) AddVote(validator types.Address) {
	w.Votes[validator] = true
}

// GetVotes returns all votes
func (w *WaveState) GetVotes() map[types.Address]bool {
	return w.Votes
}

// GetStatus returns the wave status
func (w *WaveState) GetStatus() types.WaveStatus {
	return w.Status
}

// SetStatus sets the wave status
func (w *WaveState) SetStatus(status types.WaveStatus) {
	w.Status = status
}

// GetLeader returns the wave leader
func (w *WaveState) GetLeader() types.Address {
	return w.Leader
}

// SetLeader sets the wave leader
func (w *WaveState) SetLeader(leader types.Address) {
	w.Leader = leader
} 