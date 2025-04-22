package state

import (
	"errors"
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

var (
	ErrAccountNotFound = errors.New("account not found")
	ErrInvalidBalance  = errors.New("invalid balance")
)

// State represents the current state of the blockchain
type State struct {
	accounts map[string]*types.Account
	mu       sync.RWMutex
	storage  map[string]map[string][]byte
}

// NewState creates a new state
func NewState() *State {
	return &State{
		accounts: make(map[string]*types.Account),
		storage:  make(map[string]map[string][]byte),
	}
}

// GetAccount retrieves an account by address
func (s *State) GetAccount(address string) (*types.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, exists := s.accounts[address]
	if !exists {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

// SetAccount sets an account in the state
func (s *State) SetAccount(address string, account *types.Account) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts[address] = account
	if _, exists := s.storage[address]; !exists {
		s.storage[address] = make(map[string][]byte)
	}
}

// UpdateAccount updates an account in the state
func (s *State) UpdateAccount(address string, account *types.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.accounts[address]; !exists {
		return ErrAccountNotFound
	}
	s.accounts[address] = account
	return nil
}

// BatchUpdateBalances updates multiple account balances in a batch
func (s *State) BatchUpdateBalances(updates map[string]uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for address, newBalance := range updates {
		if account, exists := s.accounts[address]; exists {
			account.Balance = newBalance
		} else {
			return ErrAccountNotFound
		}
	}
	return nil
}

// SetStorage sets a storage value for an account
func (s *State) SetStorage(address, key, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	addr := string(address)
	if _, exists := s.accounts[addr]; !exists {
		return errors.New("account not found")
	}

	if _, exists := s.storage[addr]; !exists {
		s.storage[addr] = make(map[string][]byte)
	}
	s.storage[addr][string(key)] = value
	return nil
}

// GetStorage retrieves a storage value for an account
func (s *State) GetStorage(address, key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	addr := string(address)
	if _, exists := s.accounts[addr]; !exists {
		return nil, errors.New("account not found")
	}

	if storage, exists := s.storage[addr]; exists {
		if value, exists := storage[string(key)]; exists {
			return value, nil
		}
	}
	return nil, errors.New("storage key not found")
}

// ExecuteTransaction executes a transaction and updates the state
func (s *State) ExecuteTransaction(tx *types.Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get sender account
	senderAddr := string(tx.From)
	sender, exists := s.accounts[senderAddr]
	if !exists {
		return errors.New("sender account not found")
	}

	// Check sender's balance
	if sender.Balance < tx.Value {
		return errors.New("insufficient balance")
	}

	// Check sender's nonce
	if sender.Nonce != tx.Nonce {
		return errors.New("invalid nonce")
	}

	// Get or create recipient account
	recipientAddr := string(tx.To)
	recipient, exists := s.accounts[recipientAddr]
	if !exists {
		recipient = &types.Account{
			Address: tx.To,
			Balance: 0,
			Nonce:   0,
		}
		s.accounts[recipientAddr] = recipient
		s.storage[recipientAddr] = make(map[string][]byte)
	}

	// Update balances
	sender.Balance -= tx.Value
	recipient.Balance += tx.Value

	// Update sender's nonce
	sender.Nonce++

	return nil
}

// CreateSnapshot creates a snapshot of the current state
func (s *State) CreateSnapshot() ([]byte, error) {
	// TODO: Implement state snapshot creation
	return nil, nil
}

// RestoreSnapshot restores the state from a snapshot
func (s *State) RestoreSnapshot(snapshot []byte) error {
	// TODO: Implement state snapshot restoration
	return nil
}

// GenerateProof generates a proof for an account's state
func (s *State) GenerateProof(address []byte) (*types.StateProof, error) {
	// TODO: Implement state proof generation
	return nil, nil
}

// VerifyProof verifies a state proof
func (s *State) VerifyProof(proof *types.StateProof) (bool, error) {
	// TODO: Implement state proof verification
	return false, nil
}

// Copy creates a deep copy of the state
func (s *State) Copy() *State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	newState := NewState()
	
	// Copy accounts
	for addr, account := range s.accounts {
		newState.accounts[addr] = &types.Account{
			Address: account.Address,
			Balance: account.Balance,
			Nonce:   account.Nonce,
		}
	}

	// Copy storage
	for addr, storage := range s.storage {
		newState.storage[addr] = make(map[string][]byte)
		for key, value := range storage {
			newState.storage[addr][key] = value
		}
	}

	return newState
}

// StateManager manages the blockchain state
type StateManager struct {
	state *State
	mu    sync.RWMutex
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		state: NewState(),
	}
}

// GetAccount retrieves an account by address
func (sm *StateManager) GetAccount(address string) (*types.Account, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.GetAccount(address)
}

// SetAccount sets an account in the state
func (sm *StateManager) SetAccount(address string, account *types.Account) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.SetAccount(address, account)
	return nil
}

// UpdateAccount updates an account in the state
func (sm *StateManager) UpdateAccount(address string, account *types.Account) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state.UpdateAccount(address, account)
}

// ExecuteTransaction executes a transaction and updates the state
func (sm *StateManager) ExecuteTransaction(tx *types.Transaction) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state.ExecuteTransaction(tx)
}

// GetState returns the current state
func (sm *StateManager) GetState() *State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// CreateAccount creates a new account
func (sm *StateManager) CreateAccount(address []byte, balance uint64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	account := &types.Account{
		Address: address,
		Balance: balance,
		Nonce:   0,
	}
	sm.state.SetAccount(string(address), account)
	return nil
}

// UpdateBalance updates an account's balance
func (sm *StateManager) UpdateBalance(address []byte, balance uint64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	account, err := sm.state.GetAccount(string(address))
	if err != nil {
		return err
	}
	account.Balance = balance
	sm.state.SetAccount(string(address), account)
	return nil
}

// UpdateNonce updates an account's nonce
func (sm *StateManager) UpdateNonce(address []byte, nonce uint64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	account, err := sm.state.GetAccount(string(address))
	if err != nil {
		return err
	}
	account.Nonce = nonce
	sm.state.SetAccount(string(address), account)
	return nil
}

// SetStorage sets a storage value for an account
func (sm *StateManager) SetStorage(address, key, value []byte) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state.SetStorage(address, key, value)
}

// GetStorage retrieves a storage value for an account
func (sm *StateManager) GetStorage(address, key []byte) ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.GetStorage(address, key)
}

// SetCode sets the code for an account
func (sm *StateManager) SetCode(address, code []byte) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	account, err := sm.state.GetAccount(string(address))
	if err != nil {
		return err
	}
	account.Code = code
	return sm.state.UpdateAccount(string(address), account)
}

// GetCode retrieves the code for an account
func (s *State) GetCode(address []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	addr := string(address)
	if _, exists := s.accounts[addr]; !exists {
		return nil, ErrAccountNotFound
	}

	// TODO: Implement code storage and retrieval
	return nil, nil
}

// SetCode sets the code for an account
func (s *State) SetCode(address, code []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	addr := string(address)
	if _, exists := s.accounts[addr]; !exists {
		return ErrAccountNotFound
	}

	// TODO: Implement code storage
	return nil
}

// Equal checks if two states are equal
func (s *State) Equal(other *State) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check accounts
	if len(s.accounts) != len(other.accounts) {
		return false
	}
	for addr, account := range s.accounts {
		otherAccount, exists := other.accounts[addr]
		if !exists {
			return false
		}
		if account.Balance != otherAccount.Balance || account.Nonce != otherAccount.Nonce {
			return false
		}
	}

	// Check storage
	if len(s.storage) != len(other.storage) {
		return false
	}
	for addr, storage := range s.storage {
		otherStorage, exists := other.storage[addr]
		if !exists {
			return false
		}
		if len(storage) != len(otherStorage) {
			return false
		}
		for key, value := range storage {
			otherValue, exists := otherStorage[key]
			if !exists {
				return false
			}
			if string(value) != string(otherValue) {
				return false
			}
		}
	}

	return true
}