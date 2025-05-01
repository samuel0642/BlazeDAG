package core

import (
	"github.com/CrossDAG/BlazeDAG/internal/storage"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// StateManager handles state management
type StateManager struct {
	state   *types.State
	storage *storage.Storage
}

// NewStateManager creates a new state manager
func NewStateManager(state *types.State, storage *storage.Storage) *StateManager {
	return &StateManager{
		state:   state,
		storage: storage,
	}
}

// GetState returns the current state
func (sm *StateManager) GetState() *types.State {
	return sm.state
}

// UpdateState updates the state
func (sm *StateManager) UpdateState(newState *types.State) error {
	sm.state = newState
	return sm.storage.SaveState(newState)
} 