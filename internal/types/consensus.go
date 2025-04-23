package types

import (
	"fmt"
	"time"
)

// Wave represents a consensus wave number
type Wave uint64

// ConsensusConfig holds the configuration for the consensus engine
type ConsensusConfig struct {
	WaveTimeout   time.Duration
	RoundDuration time.Duration
	ValidatorSet  []string
	QuorumSize    int
	ListenAddr    string
	Seeds         []string
}

// Validate checks if the consensus configuration is valid
func (c *ConsensusConfig) Validate() error {
	if c.WaveTimeout <= 0 {
		return fmt.Errorf("wave timeout must be positive")
	}
	if c.RoundDuration <= 0 {
		return fmt.Errorf("round duration must be positive")
	}
	if len(c.ValidatorSet) == 0 {
		return fmt.Errorf("validator set cannot be empty")
	}
	if c.QuorumSize <= 0 || c.QuorumSize > len(c.ValidatorSet) {
		return fmt.Errorf("invalid quorum size")
	}
	if c.ListenAddr == "" {
		return fmt.Errorf("listen address cannot be empty")
	}
	return nil
}

// VoteType represents the type of vote
type VoteType int

const (
	VoteTypeProposal VoteType = iota
	VoteTypeCommit
	VoteTypeAbort
)

func (v VoteType) String() string {
	switch v {
	case VoteTypeProposal:
		return "Proposal"
	case VoteTypeCommit:
		return "Commit"
	case VoteTypeAbort:
		return "Abort"
	default:
		return "Unknown"
	}
}

// Complaint represents a complaint against a validator
type Complaint struct {
	Validator Address
	Wave      Wave
	Evidence  []byte
}

// ConsensusStatus represents the current status of the consensus engine
type ConsensusStatus struct {
	CurrentWave Wave
	IsLeader    bool
	Leader      Address
	WaveStatus  WaveStatus
}

// WaveStatus represents the status of a wave
type WaveStatus int

const (
	WaveStatusIdle WaveStatus = iota
	WaveStatusProposing
	WaveStatusVoting
	WaveStatusCommitting
	WaveStatusFinalized
	WaveStatusFailed
)

func (w WaveStatus) String() string {
	switch w {
	case WaveStatusIdle:
		return "Idle"
	case WaveStatusProposing:
		return "Proposing"
	case WaveStatusVoting:
		return "Voting"
	case WaveStatusCommitting:
		return "Committing"
	case WaveStatusFinalized:
		return "Finalized"
	case WaveStatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}