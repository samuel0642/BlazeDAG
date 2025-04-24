package types

import (
	"fmt"
	"time"
)

// Address represents a node address
type Address string

// Hash represents a block or transaction hash
type Hash []byte

// Round represents a consensus round
type Round uint64

// Wave represents a consensus wave
type Wave uint64

// Value represents a numeric value
type Value uint64

// Nonce represents a transaction nonce
type Nonce uint64

// BlockNumber represents a block number
type BlockNumber uint64

// Block represents a block in the DAG
type Block struct {
	Header      *BlockHeader
	Body        *BlockBody
	Certificate *Certificate
	hash        []byte // cached hash
}

// BlockHeader represents a block header
type BlockHeader struct {
	Version     uint64
	Timestamp   time.Time
	Round       Round
	Wave        Wave
	Height      BlockNumber
	ParentHash  Hash
	References  []*Reference
	StateRoot   Hash
	Validator   Address
	Signature   Signature
}

// BlockBody represents a block body
type BlockBody struct {
	Transactions []*Transaction
	Receipts     []*Receipt
	Events       []*Event
}

// Reference represents a block reference
type Reference struct {
	BlockHash Hash
	Round     Round
	Wave      Wave
	Type      ReferenceType
}

// Transaction represents a transaction
type Transaction struct {
	Nonce     Nonce
	From      Address
	To        Address
	Value     Value
	GasLimit  uint64
	GasPrice  uint64
	Data      []byte
	Signature Signature
}

// Receipt represents a transaction receipt
type Receipt struct {
	TransactionHash Hash
	BlockHash       Hash
	BlockNumber     BlockNumber
	From            Address
	To              Address
	ContractAddress *Address
	GasUsed         uint64
	Status          uint64
	Logs            []*Log
}

// Event represents an event
type Event struct {
	Address         Address
	Topics          []Hash
	Data            []byte
	BlockNumber     BlockNumber
	TransactionHash Hash
	LogIndex        uint64
	BlockHash       Hash
	TransactionIndex uint64
}

// Log represents a log entry
type Log struct {
	Address         Address
	Topics          []Hash
	Data            []byte
	BlockNumber     BlockNumber
	TransactionHash Hash
	LogIndex        uint64
	BlockHash       Hash
	TransactionIndex uint64
}

// Signature represents a cryptographic signature
type Signature struct {
	Validator  Address
	Signature  []byte
	Timestamp  time.Time
}

// Proposal represents a consensus proposal
type Proposal struct {
	ID        Hash
	BlockHash Hash
	Round     Round
	Wave      Wave
	Block     *Block
	Proposer  Address
	Timestamp time.Time
	Status    ProposalStatus
	Signature Signature
}

// Vote represents a consensus vote
type Vote struct {
	ProposalID Hash
	BlockHash  Hash
	Validator  Address
	Round      Round
	Wave       Wave
	Timestamp  time.Time
	Type       VoteType
	Signature  Signature
}

// Complaint represents a consensus complaint
type Complaint struct {
	ID        Hash
	BlockHash Hash
	Validator Address
	Round     Round
	Wave      Wave
	Timestamp time.Time
	Reason    string
	Signature Signature
}

// NetworkMessage represents a network message
type NetworkMessage struct {
	Type      MessageType
	Payload   []byte
	From      Address
	To        Address
	Sender    Address
	Recipient Address
	Timestamp time.Time
	TTL       int
	Signature []byte
	Priority  int
}

// ConsensusMessage represents a consensus message
type ConsensusMessage struct {
	Type      MessageType
	Proposal  *Proposal
	Vote      *Vote
	Complaint *Complaint
}

// Peer represents a network peer
type Peer struct {
	ID        string
	Address   Address
	Active    bool
	LastSeen  time.Time
	Score     int
	IsValidator bool
	Connection interface{} // Will be replaced with actual connection type
}

// Account represents an account in the state
type Account struct {
	Address     Address
	Balance     Value
	Nonce       Nonce
	Code        []byte
	StorageRoot Hash
	CodeHash    Hash
}

// BalanceUpdate represents a balance update for an account
type BalanceUpdate struct {
	Address Address
	Balance Value
}

// ConsensusConfig holds the configuration for the consensus engine
type ConsensusConfig struct {
	WaveTimeout   time.Duration
	RoundDuration time.Duration
	ValidatorSet  []Address
	QuorumSize    int
	ListenAddr    Address
	Seeds         []Address
}

// ConsensusStatus represents the current status of the consensus engine
type ConsensusStatus struct {
	CurrentWave Wave
	IsLeader    bool
	Leader      Address
	WaveStatus  WaveStatus
}

// MessageType represents the type of a message
type MessageType uint8

const (
	MessageTypeBlock MessageType = iota
	MessageTypeVote
	MessageTypeProposal
	MessageTypeComplaint
	MessageTypeSyncRequest
	MessageTypeSyncResponse
	MessageTypeRecoveryRequest
	MessageTypeRecoveryResponse
)

// ReferenceType represents the type of a reference
type ReferenceType uint8

const (
	ReferenceTypeStandard ReferenceType = iota
	ReferenceTypeVote
	ReferenceTypeComplaint
)

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

// VoteType represents the type of a vote
type VoteType uint8

const (
	VoteTypeApprove VoteType = iota
	VoteTypeReject
	VoteTypeAbstain
)

// WaveStatus represents the status of a wave
type WaveStatus uint8

const (
	WaveStatusIdle WaveStatus = iota
	WaveStatusProposing
	WaveStatusVoting
	WaveStatusCommitting
	WaveStatusFinalized
	WaveStatusFailed
)

// SyncRequestType represents the type of sync request
type SyncRequestType uint8

const (
	SyncRequestTypeFull SyncRequestType = iota
	SyncRequestTypeIncremental
	SyncRequestTypeFast
	SyncRequestTypeCertificate
)

// SyncResponseType represents the type of sync response
type SyncResponseType uint8

const (
	SyncResponseTypeFull SyncResponseType = iota
	SyncResponseTypeIncremental
	SyncResponseTypeFast
)

// RecoveryRequestType represents the type of recovery request
type RecoveryRequestType uint8

const (
	RecoveryRequestTypeBlock RecoveryRequestType = iota
	RecoveryRequestTypeState
	RecoveryRequestTypeCertificate
)

// RecoveryResponseType represents the type of recovery response
type RecoveryResponseType uint8

const (
	RecoveryResponseTypeBlock RecoveryResponseType = iota
	RecoveryResponseTypeState
	RecoveryResponseTypeCertificate
)

// Certificate represents a block certificate
type Certificate struct {
	BlockHash    Hash
	Signatures   [][]byte
	Round        Round
	Wave         Wave
	ValidatorSet [][]byte
	Timestamp    time.Time
}

// NewAccount creates a new account
func NewAccount(address Address, balance Value) *Account {
	return &Account{
		Address:     address,
		Balance:     balance,
		Nonce:       0,
		Code:        nil,
		StorageRoot: make(Hash, 0),
		CodeHash:    make(Hash, 0),
	}
}

// Equal checks if two accounts are equal
func (a *Account) Equal(other *Account) bool {
	if a == nil || other == nil {
		return a == other
	}
	return string(a.Address) == string(other.Address) &&
		a.Balance == other.Balance &&
		a.Nonce == other.Nonce &&
		string(a.Code) == string(other.Code) &&
		string(a.StorageRoot) == string(other.StorageRoot) &&
		string(a.CodeHash) == string(other.CodeHash)
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