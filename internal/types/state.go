package types

// State represents the global state
type State struct {
	Accounts map[string]*Account
	Storage  map[string]map[string][]byte
	Code     map[string][]byte
	Nonce    map[string]uint64
	Balance  map[string]uint64
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

// NewState creates a new state
func NewState() *State {
	return &State{
		Accounts: make(map[string]*Account),
	}
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