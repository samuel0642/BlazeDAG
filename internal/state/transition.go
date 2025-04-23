package state

import (
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// StateTransition represents a state transition
type StateTransition struct {
	BlockNumber types.BlockNumber
	Changes     []*StateChange
	Proof       *types.StateProof
}

// StateChange represents a state change
type StateChange struct {
	Type       StateChangeType
	Address    string
	StorageKey string
	OldValue   interface{}
	NewValue   interface{}
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

// StateTransitionManager handles state transitions
type StateTransitionManager struct {
	transitions []*StateTransition
	state       *State
	mu          sync.RWMutex
}

// NewStateTransitionManager creates a new state transition manager
func NewStateTransitionManager(state *State) *StateTransitionManager {
	return &StateTransitionManager{
		transitions: make([]*StateTransition, 0),
		state:      state,
	}
}

// CreateTransition creates a new state transition
func (stm *StateTransitionManager) CreateTransition(blockNumber types.BlockNumber) *StateTransition {
	stm.mu.Lock()
	defer stm.mu.Unlock()

	transition := &StateTransition{
		BlockNumber: blockNumber,
		Changes:     make([]*StateChange, 0),
	}

	stm.transitions = append(stm.transitions, transition)

	return transition
}

// ApplyChange applies a state change
func (stm *StateTransitionManager) ApplyChange(transition *StateTransition, change *StateChange) error {
	stm.mu.Lock()
	defer stm.mu.Unlock()

	// Apply change
	switch change.Type {
	case StateChangeTypeAccount:
		account, err := stm.state.GetAccount(change.Address)
		if err != nil {
			return err
		}
		account.Balance = change.NewValue.(types.Value)
		stm.state.SetAccount(change.Address, account)
	case StateChangeTypeStorage:
		err := stm.state.SetStorage([]byte(change.Address), []byte(change.StorageKey), change.NewValue.([]byte))
		if err != nil {
			return err
		}
	case StateChangeTypeCode:
		err := stm.state.SetCode([]byte(change.Address), change.NewValue.([]byte))
		if err != nil {
			return err
		}
	case StateChangeTypeBalance:
		account, err := stm.state.GetAccount(change.Address)
		if err != nil {
			return err
		}
		account.Balance = change.NewValue.(types.Value)
		stm.state.SetAccount(change.Address, account)
	case StateChangeTypeNonce:
		account, err := stm.state.GetAccount(change.Address)
		if err != nil {
			return err
		}
		account.Nonce = change.NewValue.(types.Nonce)
		stm.state.SetAccount(change.Address, account)
	}

	// Add change to transition
	transition.Changes = append(transition.Changes, change)

	return nil
}

// GetTransition gets a transition by block number
func (stm *StateTransitionManager) GetTransition(blockNumber types.BlockNumber) *StateTransition {
	stm.mu.RLock()
	defer stm.mu.RUnlock()

	for _, transition := range stm.transitions {
		if transition.BlockNumber == blockNumber {
			return transition
		}
	}

	return nil
}

// GetLatestTransition gets the latest transition
func (stm *StateTransitionManager) GetLatestTransition() *StateTransition {
	stm.mu.RLock()
	defer stm.mu.RUnlock()

	if len(stm.transitions) == 0 {
		return nil
	}

	return stm.transitions[len(stm.transitions)-1]
}

// VerifyTransition verifies a transition
func (stm *StateTransitionManager) VerifyTransition(transition *StateTransition) bool {
	if transition == nil {
		return false
	}

	// Verify each change
	for _, change := range transition.Changes {
		if !stm.verifyChange(change) {
			return false
		}
	}

	return true
}

// verifyChange verifies a state change
func (stm *StateTransitionManager) verifyChange(change *StateChange) bool {
	switch change.Type {
	case StateChangeTypeAccount:
		account, err := stm.state.GetAccount(change.Address)
		if err != nil {
			return false
		}
		return account.Balance == change.NewValue.(types.Value)
	case StateChangeTypeStorage:
		value, err := stm.state.GetStorage([]byte(change.Address), []byte(change.StorageKey))
		if err != nil {
			return false
		}
		return string(value) == string(change.NewValue.([]byte))
	case StateChangeTypeCode:
		code, err := stm.state.GetCode([]byte(change.Address))
		if err != nil {
			return false
		}
		return string(code) == string(change.NewValue.([]byte))
	case StateChangeTypeBalance:
		account, err := stm.state.GetAccount(change.Address)
		if err != nil {
			return false
		}
		return account.Balance == change.NewValue.(types.Value)
	case StateChangeTypeNonce:
		account, err := stm.state.GetAccount(change.Address)
		if err != nil {
			return false
		}
		return account.Nonce == change.NewValue.(types.Nonce)
	default:
		return false
	}
} 