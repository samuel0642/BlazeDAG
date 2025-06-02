package types

// State represents the core state of the blockchain
type State struct {
	// Block state
	CurrentWave     uint64
	CurrentRound    uint64
	Height          uint64
	LatestBlock     *Block
	PendingBlocks   map[string]*Block
	FinalizedBlocks map[string]*Block

	// Consensus state
	ActiveProposals map[string]*Proposal
	Votes           map[string][]*Vote

	// Network state
	ConnectedPeers map[Address]*Peer

	// Account state
	Accounts map[string]*Account
}

// StateChange represents a state change
type StateChange struct {
	Type      StateChangeType
	OldValue  []byte
	NewValue  []byte
	BlockHash string
}

// StateChangeType represents the type of state change
type StateChangeType int

const (
	StateChangeTypeAccount StateChangeType = iota
	StateChangeTypeStorage
	StateChangeTypeCode
	StateChangeTypeBalance
	StateChangeTypeNonce
)

// NewState creates a new state instance
func NewState() *State {
	return &State{
		CurrentWave:     1,
		CurrentRound:    0,
		Height:         0,
		PendingBlocks:   make(map[string]*Block),
		FinalizedBlocks: make(map[string]*Block),
		ActiveProposals: make(map[string]*Proposal),
		Votes:           make(map[string][]*Vote),
		ConnectedPeers:  make(map[Address]*Peer),
		Accounts:        make(map[string]*Account),
	}
}

// ComputeRootHash computes the state root hash
func (s *State) ComputeRootHash() Hash {
	// TODO: Implement proper state root calculation
	return Hash{}
}

// GetAccount retrieves an account by its address
func (s *State) GetAccount(address []byte) (*Account, bool) {
	account, exists := s.Accounts[string(address)]
	return account, exists
}

// SetAccount sets an account in the state
func (s *State) SetAccount(address []byte, account *Account) {
	s.Accounts[string(address)] = account
}

// CleanupOldBlocks removes old blocks to prevent memory overflow
func (s *State) CleanupOldBlocks() {
	// Clean up pending blocks
	if len(s.PendingBlocks) > 10 {
		// Keep only 5 most recent blocks
		keepCount := 5
		removed := 0
		for hash := range s.PendingBlocks {
			if removed >= len(s.PendingBlocks)-keepCount {
				break
			}
			delete(s.PendingBlocks, hash)
			removed++
		}
	}
	
	// Clean up finalized blocks  
	if len(s.FinalizedBlocks) > 20 {
		// Keep only 10 most recent blocks
		keepCount := 10
		removed := 0
		for hash := range s.FinalizedBlocks {
			if removed >= len(s.FinalizedBlocks)-keepCount {
				break
			}
			delete(s.FinalizedBlocks, hash)
			removed++
		}
	}
}

// StateTransition represents a state transition
type StateTransition struct {
	PreState     *State
	PostState    *State
	Transactions []Transaction
	Receipts     []Receipt
	Events       []Event
	StateRoot    []byte
	BlockHash    []byte
	Wave         uint64
	Round        uint64
	CausalOrder  uint64
	StateChanges map[string]StateChange
} 