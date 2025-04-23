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
func (s *State) BatchUpdateBalances(updates map[string]types.Value) error {
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
	if uint64(sender.Balance) < uint64(tx.Value) {
		return errors.New("insufficient balance")
	}

	// Check sender's nonce
	if uint64(sender.Nonce) != uint64(tx.Nonce) {
		return errors.New("invalid nonce")
	}

	// Get or create recipient account
	recipientAddr := string(tx.To)
	recipient, exists := s.accounts[recipientAddr]
	if !exists {
		recipient = &types.Account{
			Address: tx.To,
			Balance: types.Value(0),
			Nonce:   types.Nonce(0),
		}
		s.accounts[recipientAddr] = recipient
		s.storage[recipientAddr] = make(map[string][]byte)
	}

	// Update balances
	sender.Balance = types.Value(uint64(sender.Balance) - uint64(tx.Value))
	recipient.Balance = types.Value(uint64(recipient.Balance) + uint64(tx.Value))

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

// GetState returns the current state
func (s *State) GetState() *State {
	return s
}

// CreateAccount creates a new account
func (s *State) CreateAccount(address []byte, balance types.Value) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	addr := string(address)
	if _, exists := s.accounts[addr]; exists {
		return errors.New("account already exists")
	}

	s.accounts[addr] = &types.Account{
		Address: types.Address(address),
		Balance: balance,
		Nonce:   0,
	}
	s.storage[addr] = make(map[string][]byte)
	return nil
}

// UpdateBalance updates an account's balance
func (s *State) UpdateBalance(address []byte, balance types.Value) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	addr := string(address)
	account, exists := s.accounts[addr]
	if !exists {
		return errors.New("account not found")
	}

	account.Balance = balance
	return nil
}

// UpdateNonce updates an account's nonce
func (s *State) UpdateNonce(address []byte, nonce types.Nonce) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	addr := string(address)
	account, exists := s.accounts[addr]
	if !exists {
		return errors.New("account not found")
	}

	account.Nonce = nonce
	return nil
}

// SetCode sets the code for an account
func (s *State) SetCode(address []byte, code []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	addr := string(address)
	if _, exists := s.accounts[addr]; !exists {
		return errors.New("account not found")
	}

	s.storage[addr]["code"] = code
	return nil
}

// GetCode gets the code for an account
func (s *State) GetCode(address []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	addr := string(address)
	if _, exists := s.accounts[addr]; !exists {
		return nil, errors.New("account not found")
	}

	if code, exists := s.storage[addr]["code"]; exists {
		return code, nil
	}
	return nil, nil
}

// Equal checks if two states are equal
func (s *State) Equal(other *State) bool {
	s.mu.RLock()
	other.mu.RLock()
	defer s.mu.RUnlock()
	defer other.mu.RUnlock()

	if len(s.accounts) != len(other.accounts) {
		return false
	}

	for addr, account := range s.accounts {
		otherAccount, exists := other.accounts[addr]
		if !exists {
			return false
		}
		if uint64(account.Balance) != uint64(otherAccount.Balance) ||
			uint64(account.Nonce) != uint64(otherAccount.Nonce) {
			return false
		}
	}

	return true
}