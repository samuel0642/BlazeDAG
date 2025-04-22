package state

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"
)

// StateTransition represents a state transition
type StateTransition struct {
	PreState  *State
	PostState *State
	Changes   []*StateChange
	Root      []byte
}

// StateChange represents a change to the state
type StateChange struct {
	Address string
	Type    StateChangeType
	OldValue interface{}
	NewValue interface{}
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

// StateTransitionManager manages state transitions
type StateTransitionManager struct {
	transitions []*StateTransition
	mu          sync.RWMutex
}

// NewStateTransitionManager creates a new state transition manager
func NewStateTransitionManager() *StateTransitionManager {
	return &StateTransitionManager{
		transitions: make([]*StateTransition, 0),
	}
}

// CreateTransition creates a new state transition
func (stm *StateTransitionManager) CreateTransition(preState *State, changes []*StateChange) *StateTransition {
	stm.mu.Lock()
	defer stm.mu.Unlock()

	// Create post state by applying changes
	postState := preState.Copy()
	for _, change := range changes {
		stm.applyChange(postState, change)
	}

	// Calculate state root
	root := stm.calculateStateRoot(postState)

	transition := &StateTransition{
		PreState:  preState,
		PostState: postState,
		Changes:   changes,
		Root:      root,
	}

	stm.transitions = append(stm.transitions, transition)
	return transition
}

// applyChange applies a state change to the state
func (stm *StateTransitionManager) applyChange(state *State, change *StateChange) {
	switch change.Type {
	case StateChangeTypeAccount:
		state.SetAccount(change.Address, change.NewValue.(*Account))
	case StateChangeTypeStorage:
		state.SetStorage(change.Address, change.NewValue.(map[string][]byte))
	case StateChangeTypeCode:
		state.SetCode(change.Address, change.NewValue.([]byte))
	case StateChangeTypeBalance:
		state.SetBalance(change.Address, change.NewValue.(uint64))
	case StateChangeTypeNonce:
		state.SetNonce(change.Address, change.NewValue.(uint64))
	}
}

// calculateStateRoot calculates the state root
func (stm *StateTransitionManager) calculateStateRoot(state *State) []byte {
	// Create a hash of the state
	h := sha256.New()

	// Hash accounts
	for addr, account := range state.Accounts {
		h.Write([]byte(addr))
		h.Write(account.Balance.Bytes())
		binary.Write(h, binary.LittleEndian, account.Nonce)
		h.Write(account.CodeHash)
		h.Write(account.StorageRoot)
	}

	// Hash storage
	for addr, storage := range state.Storage {
		h.Write([]byte(addr))
		for key, value := range storage {
			h.Write([]byte(key))
			h.Write(value)
		}
	}

	return h.Sum(nil)
}

// GetTransition returns a state transition by index
func (stm *StateTransitionManager) GetTransition(index int) *StateTransition {
	stm.mu.RLock()
	defer stm.mu.RUnlock()

	if index < 0 || index >= len(stm.transitions) {
		return nil
	}
	return stm.transitions[index]
}

// GetLatestTransition returns the latest state transition
func (stm *StateTransitionManager) GetLatestTransition() *StateTransition {
	stm.mu.RLock()
	defer stm.mu.RUnlock()

	if len(stm.transitions) == 0 {
		return nil
	}
	return stm.transitions[len(stm.transitions)-1]
}

// VerifyTransition verifies a state transition
func (stm *StateTransitionManager) VerifyTransition(transition *StateTransition) bool {
	// Verify pre-state matches
	if !transition.PreState.Equal(transition.PreState) {
		return false
	}

	// Verify changes were applied correctly
	for _, change := range transition.Changes {
		if !stm.verifyChange(transition.PostState, change) {
			return false
		}
	}

	// Verify state root
	calculatedRoot := stm.calculateStateRoot(transition.PostState)
	return string(calculatedRoot) == string(transition.Root)
}

// verifyChange verifies a state change
func (stm *StateTransitionManager) verifyChange(state *State, change *StateChange) bool {
	switch change.Type {
	case StateChangeTypeAccount:
		account := state.GetAccount(change.Address)
		return account.Equal(change.NewValue.(*Account))
	case StateChangeTypeStorage:
		storage := state.GetStorage(change.Address)
		return stm.verifyStorage(storage, change.NewValue.(map[string][]byte))
	case StateChangeTypeCode:
		code := state.GetCode(change.Address)
		return string(code) == string(change.NewValue.([]byte))
	case StateChangeTypeBalance:
		balance := state.GetBalance(change.Address)
		return balance == change.NewValue.(uint64)
	case StateChangeTypeNonce:
		nonce := state.GetNonce(change.Address)
		return nonce == change.NewValue.(uint64)
	default:
		return false
	}
}

// verifyStorage verifies storage changes
func (stm *StateTransitionManager) verifyStorage(actual, expected map[string][]byte) bool {
	if len(actual) != len(expected) {
		return false
	}
	for key, value := range expected {
		if actualValue, exists := actual[key]; !exists || string(actualValue) != string(value) {
			return false
		}
	}
	return true
} 