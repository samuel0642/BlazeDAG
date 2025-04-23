package state

import (
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// StateManager manages the blockchain state
type StateManager struct {
	accounts    map[string]*types.Account
	validators  []string
	nodeID      string
	mu          sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		accounts:   make(map[string]*types.Account),
		validators: make([]string, 0),
	}
}

// GetAccount returns an account by address
func (sm *StateManager) GetAccount(address string) (*types.Account, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	account, exists := sm.accounts[address]
	if !exists {
		return nil, types.ErrAccountNotFound
	}

	return account, nil
}

// CreateAccount creates a new account
func (sm *StateManager) CreateAccount(address string, initialBalance types.Value) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.accounts[address]; exists {
		return types.ErrAccountExists
	}

	sm.accounts[address] = &types.Account{
		Address: types.Address([]byte(address)),
		Balance: initialBalance,
		Nonce:   types.Nonce(0),
	}

	return nil
}

// UpdateBalance updates an account's balance
func (sm *StateManager) UpdateBalance(address string, amount int64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	account, exists := sm.accounts[address]
	if !exists {
		return types.ErrAccountNotFound
	}

	if int64(account.Balance)+amount < 0 {
		return types.ErrInsufficientBalance
	}

	account.Balance = types.Value(int64(account.Balance) + amount)
	return nil
}

// IncrementNonce increments an account's nonce
func (sm *StateManager) IncrementNonce(address string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	account, exists := sm.accounts[address]
	if !exists {
		return types.ErrAccountNotFound
	}

	account.Nonce++
	return nil
}

// GetNonce returns an account's nonce
func (sm *StateManager) GetNonce(address string) (types.Nonce, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	account, exists := sm.accounts[address]
	if !exists {
		return 0, types.ErrAccountNotFound
	}

	return account.Nonce, nil
}

// AddValidator adds a validator
func (sm *StateManager) AddValidator(address string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if validator already exists
	for _, v := range sm.validators {
		if v == address {
			return nil
		}
	}

	sm.validators = append(sm.validators, address)
	return nil
}

// RemoveValidator removes a validator
func (sm *StateManager) RemoveValidator(address string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, v := range sm.validators {
		if v == address {
			sm.validators = append(sm.validators[:i], sm.validators[i+1:]...)
			return nil
		}
	}

	return types.ErrValidatorNotFound
}

// GetValidators returns all validators
func (sm *StateManager) GetValidators() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	validators := make([]string, len(sm.validators))
	copy(validators, sm.validators)
	return validators
}

// IsValidator checks if an address is a validator
func (sm *StateManager) IsValidator(address string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, v := range sm.validators {
		if v == address {
			return true
		}
	}

	return false
}

// SetNodeID sets the node ID
func (sm *StateManager) SetNodeID(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.nodeID = id
}

// GetNodeID returns the node ID
func (sm *StateManager) GetNodeID() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.nodeID
}

// CommitBlock commits a block to the state
func (sm *StateManager) CommitBlock(block *types.Block) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Process transactions
	for _, tx := range block.Body.Transactions {
		// Check sender account
		sender, exists := sm.accounts[string(tx.From)]
		if !exists {
			return types.ErrAccountNotFound
		}

		// Check balance
		if uint64(sender.Balance) < uint64(tx.Value) {
			return types.ErrInsufficientBalance
		}

		// Check nonce
		if uint64(sender.Nonce) != uint64(tx.Nonce) {
			return types.ErrInvalidNonce
		}

		// Update sender balance and nonce
		sender.Balance = types.Value(uint64(sender.Balance) - uint64(tx.Value))
		sender.Nonce++

		// Update recipient balance
		recipient, exists := sm.accounts[string(tx.To)]
		if !exists {
			// Create recipient account if it doesn't exist
			sm.accounts[string(tx.To)] = &types.Account{
				Address: tx.To,
				Balance: tx.Value,
				Nonce:   types.Nonce(0),
			}
		} else {
			recipient.Balance = types.Value(uint64(recipient.Balance) + uint64(tx.Value))
		}
	}

	return nil
}

// GetStateRoot returns the current state root
func (sm *StateManager) GetStateRoot() types.Hash {
	// TODO: Implement state root calculation
	return types.Hash([]byte("state_root"))
}

// VerifyState verifies the state
func (sm *StateManager) VerifyState() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Check all accounts have valid balances
	for _, account := range sm.accounts {
		if uint64(account.Balance) < 0 {
			return types.ErrInvalidBalance
		}
	}

	return nil
} 