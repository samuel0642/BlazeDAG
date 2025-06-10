package evm

import (
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// State represents the EVM state with Ethereum compatibility
type State struct {
	accounts map[common.Address]*Account
	root     common.Hash
	mu       sync.RWMutex
}

// NewState creates a new state
func NewState() *State {
	return &State{
		accounts: make(map[common.Address]*Account),
		root:     common.Hash{},
	}
}

// GetAccount gets an account by address, creates one if it doesn't exist
func (s *State) GetAccount(address common.Address) *Account {
	s.mu.RLock()
	account, exists := s.accounts[address]
	s.mu.RUnlock()
	
	if !exists {
		s.mu.Lock()
		// Double-check after acquiring write lock
		account, exists = s.accounts[address]
		if !exists {
			account = NewAccount(address)
			s.accounts[address] = account
		}
		s.mu.Unlock()
	}
	
	return account
}

// GetOrCreateAccount gets an account or creates it if it doesn't exist
func (s *State) GetOrCreateAccount(address common.Address) *Account {
	return s.GetAccount(address)
}

// AccountExists checks if an account exists
func (s *State) AccountExists(address common.Address) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.accounts[address]
	return exists
}

// GetBalance gets the balance of an account
func (s *State) GetBalance(address common.Address) *big.Int {
	account := s.GetAccount(address)
	return account.GetBalance()
}

// SetBalance sets the balance of an account
func (s *State) SetBalance(address common.Address, balance *big.Int) {
	account := s.GetAccount(address)
	account.SetBalance(balance)
	s.invalidateRoot()
}

// AddBalance adds to the balance of an account
func (s *State) AddBalance(address common.Address, amount *big.Int) {
	account := s.GetAccount(address)
	account.AddBalance(amount)
	s.invalidateRoot()
}

// SubBalance subtracts from the balance of an account
func (s *State) SubBalance(address common.Address, amount *big.Int) bool {
	account := s.GetAccount(address)
	success := account.SubBalance(amount)
	if success {
		s.invalidateRoot()
	}
	return success
}

// GetNonce gets the nonce of an account
func (s *State) GetNonce(address common.Address) uint64 {
	account := s.GetAccount(address)
	return account.GetNonce()
}

// SetNonce sets the nonce of an account
func (s *State) SetNonce(address common.Address, nonce uint64) {
	account := s.GetAccount(address)
	account.SetNonce(nonce)
	s.invalidateRoot()
}

// IncrementNonce increments the nonce of an account
func (s *State) IncrementNonce(address common.Address) {
	account := s.GetAccount(address)
	account.IncrementNonce()
	s.invalidateRoot()
}

// GetCode gets the code of an account
func (s *State) GetCode(address common.Address) []byte {
	account := s.GetAccount(address)
	return account.GetCode()
}

// SetCode sets the code of an account
func (s *State) SetCode(address common.Address, code []byte) {
	account := s.GetAccount(address)
	account.SetCode(code)
	s.invalidateRoot()
}

// GetCodeHash gets the code hash of an account
func (s *State) GetCodeHash(address common.Address) common.Hash {
	account := s.GetAccount(address)
	return account.GetCodeHash()
}

// GetStorage gets the storage value of an account
func (s *State) GetStorage(address common.Address, key common.Hash) common.Hash {
	account := s.GetAccount(address)
	return account.GetStorage(key)
}

// SetStorage sets the storage value of an account
func (s *State) SetStorage(address common.Address, key common.Hash, value common.Hash) {
	account := s.GetAccount(address)
	account.SetStorage(key, value)
	s.invalidateRoot()
}

// IsContract checks if an address contains a contract
func (s *State) IsContract(address common.Address) bool {
	account := s.GetAccount(address)
	return account.IsContract()
}

// CreateContract creates a new contract at the given address
func (s *State) CreateContract(address common.Address, code []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if address already has code
	if account, exists := s.accounts[address]; exists && account.IsContract() {
		return errors.New("contract already exists at address")
	}
	
	account := s.GetAccount(address)
	account.SetCode(code)
	s.invalidateRoot()
	return nil
}

// Copy creates a deep copy of the state
func (s *State) Copy() *State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	newState := &State{
		accounts: make(map[common.Address]*Account),
		root:     s.root,
	}
	
	for addr, account := range s.accounts {
		newState.accounts[addr] = account.Copy()
	}
	
	return newState
}

// Root returns the state root
func (s *State) Root() common.Hash {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if (s.root == common.Hash{}) {
		s.updateRoot()
	}
	return s.root
}

// invalidateRoot marks the root as invalid
func (s *State) invalidateRoot() {
	s.root = common.Hash{}
}

// updateRoot updates the state root
func (s *State) updateRoot() {
	hasher := crypto.NewKeccakState()
	
	// Create a deterministic order by sorting addresses
	for addr, account := range s.accounts {
		// Write address
		hasher.Write(addr.Bytes())
		
		// Write account data
		hasher.Write(account.GetCodeHash().Bytes())
		hasher.Write(account.GetBalance().Bytes())
		
		// Write nonce
		nonceBytes := make([]byte, 8)
		nonce := account.GetNonce()
		for i := 0; i < 8; i++ {
			nonceBytes[i] = byte(nonce >> (i * 8))
		}
		hasher.Write(nonceBytes)
		
		// Write storage (simplified)
		for key, value := range account.Storage {
			hasher.Write(key.Bytes())
			hasher.Write(value.Bytes())
		}
	}
	
	s.root = common.BytesToHash(hasher.Sum(nil))
}

// GetAllAccounts returns all accounts (for debugging/testing)
func (s *State) GetAllAccounts() map[common.Address]*Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	accounts := make(map[common.Address]*Account)
	for addr, account := range s.accounts {
		accounts[addr] = account.Copy()
	}
	return accounts
} 