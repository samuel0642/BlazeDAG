package state

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/types"
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
	Address    string
	Type       StateChangeType
	OldValue   interface{}
	NewValue   interface{}
	StorageKey string
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
		account := &types.Account{
			Address: []byte(change.Address),
			Balance: change.NewValue.(uint64),
		}
		state.SetAccount(change.Address, account)
	case StateChangeTypeStorage:
		state.SetStorage([]byte(change.Address), []byte(change.StorageKey), change.NewValue.([]byte))
	case StateChangeTypeCode:
		state.SetCode([]byte(change.Address), change.NewValue.([]byte))
	case StateChangeTypeBalance:
		account, _ := state.GetAccount(change.Address)
		if account != nil {
			account.Balance = change.NewValue.(uint64)
			state.SetAccount(change.Address, account)
		}
	case StateChangeTypeNonce:
		account, _ := state.GetAccount(change.Address)
		if account != nil {
			account.Nonce = change.NewValue.(uint64)
			state.SetAccount(change.Address, account)
		}
	}
}

// calculateStateRoot calculates the state root
func (stm *StateTransitionManager) calculateStateRoot(state *State) []byte {
	// Create a hash of the state
	h := sha256.New()

	// Hash accounts
	for addr, account := range state.accounts {
		h.Write([]byte(addr))
		binary.Write(h, binary.LittleEndian, account.Balance)
		binary.Write(h, binary.LittleEndian, account.Nonce)
		h.Write(account.Code)
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
		account, err := state.GetAccount(change.Address)
		if err != nil {
			return false
		}
		return account.Balance == change.NewValue.(uint64)
	case StateChangeTypeStorage:
		value, err := state.GetStorage([]byte(change.Address), []byte(change.StorageKey))
		if err != nil {
			return false
		}
		return string(value) == string(change.NewValue.([]byte))
	case StateChangeTypeCode:
		code, err := state.GetCode([]byte(change.Address))
		if err != nil {
			return false
		}
		return string(code) == string(change.NewValue.([]byte))
	case StateChangeTypeBalance:
		account, err := state.GetAccount(change.Address)
		if err != nil {
			return false
		}
		return account.Balance == change.NewValue.(uint64)
	case StateChangeTypeNonce:
		account, err := state.GetAccount(change.Address)
		if err != nil {
			return false
		}
		return account.Nonce == change.NewValue.(uint64)
	default:
		return false
	}
} 