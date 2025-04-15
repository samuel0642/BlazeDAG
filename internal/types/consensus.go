package types

import "time"

// ConsensusConfig holds the configuration for the consensus engine
type ConsensusConfig struct {
	// Wave timing parameters
	WaveTimeout   time.Duration
	RoundDuration time.Duration

	// Validator parameters
	ValidatorSet []string
	QuorumSize   int

	// Network parameters
	ListenAddr string
	Seeds     []string
}

// Proposal represents a block proposal in the consensus
type Proposal struct {
	ID        []byte
	Wave      uint64
	Round     uint64
	Block     *Block
	Proposer  []byte
	Timestamp time.Time
	Status    ConsensusStatus
}

// Vote represents a validator's vote on a proposal
type Vote struct {
	ProposalID []byte
	Wave       uint64
	Round      uint64
	Validator  []byte
	Type       VoteType
	Timestamp  time.Time
}

// VoteType represents the type of vote
type VoteType int

const (
	VoteTypeApprove VoteType = iota
	VoteTypeReject
)

// Complaint represents a complaint about a block
type Complaint struct {
	ID        []byte
	BlockHash []byte
	Validator []byte
	Round     uint64
	Wave      uint64
	Timestamp time.Time
	Reason    string
	Signature []byte
}

// WaveState represents the state of a wave
type WaveState struct {
	CurrentWave    uint64
	WaveStartTime  time.Time
	WaveEndTime    time.Time
	Leader         []byte
	Proposals      map[string]Proposal
	Votes          map[string][]Vote
	WaveStatus     WaveStatus
	CommittedBlocks map[string]bool
	PendingBlocks  map[string]bool
	WaveTips       map[uint64]map[string]bool
	CausalOrder    map[string]uint64
}

// ConsensusStatus represents the status of a proposal
type ConsensusStatus int

const (
	ConsensusStatusPending ConsensusStatus = iota
	ConsensusStatusValidating
	ConsensusStatusValid
	ConsensusStatusInvalid
	ConsensusStatusCertified
	ConsensusStatusCommitted
)

// WaveStatus represents the status of a wave
type WaveStatus int

const (
	WaveStatusInitializing WaveStatus = iota
	WaveStatusActive
	WaveStatusCompleting
	WaveStatusCompleted
	WaveStatusFailed
) 