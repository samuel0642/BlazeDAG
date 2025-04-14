package types

// State represents the system state
type State struct {
	Accounts map[string]Account
	Storage  map[string]map[string][]byte
	Code     map[string][]byte
	Nonce    map[string]uint64
	Balance  map[string][]byte
}

// StateChange represents a state change
type StateChange struct {
	Address    []byte
	Type       StateChangeType
	OldValue   []byte
	NewValue   []byte
	BlockHash  []byte
	CausalOrder uint64
}

// StateChangeType represents the type of state change
type StateChangeType uint8

const (
	StateChangeTypeAccount StateChangeType = iota
	StateChangeTypeStorage
	StateChangeTypeCode
	StateChangeTypeBalance
	StateChangeTypeNonce
)

// StateProof represents a state proof
type StateProof struct {
	Key   []byte
	Value []byte
	Proof [][]byte
	Root  []byte
}

// StateTransition represents a state transition
type StateTransition struct {
	PreState     State
	PostState    State
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