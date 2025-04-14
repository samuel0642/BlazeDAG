package types

import "time"

// Proposal represents a block proposal
type Proposal struct {
	ID         []byte
	Round      uint64
	Wave       uint64
	Block      Block
	Proposer   []byte
	Timestamp  time.Time
	References []Reference
	StateRoot  []byte
	Signature  []byte
	Status     ProposalStatus
}

// Vote represents a vote on a proposal
type Vote struct {
	ProposalID []byte
	Validator  []byte
	Round      uint64
	Wave       uint64
	Timestamp  time.Time
	Signature  []byte
	Type       VoteType
}

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

// ProposalStatus represents the status of a proposal
type ProposalStatus uint8

const (
	ProposalStatusPending ProposalStatus = iota
	ProposalStatusValidating
	ProposalStatusValid
	ProposalStatusInvalid
	ProposalStatusCertified
	ProposalStatusCommitted
)

// VoteType represents the type of vote
type VoteType uint8

const (
	VoteTypeApprove VoteType = iota
	VoteTypeReject
	VoteTypeAbstain
)

// WaveStatus represents the status of a wave
type WaveStatus uint8

const (
	WaveStatusInitializing WaveStatus = iota
	WaveStatusActive
	WaveStatusCompleting
	WaveStatusCompleted
	WaveStatusFailed
) 