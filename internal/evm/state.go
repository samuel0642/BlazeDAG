package evm

import (
	"crypto/sha256"
	"errors"
	"math/big"
)

// State represents the EVM state
type State struct {
	accounts map[string]*Account
	root     []byte
}

// NewState creates a new state
func NewState() *State {
	return &State{
		accounts: make(map[string]*Account),
	}
}

// GetAccount gets an account by address
func (s *State) GetAccount(address []byte) (*Account, error) {
	account, exists := s.accounts[string(address)]
	if !exists {
		return nil, errors.New("account not found")
	}
	return account, nil
}

// GetBalance gets the balance of an account
func (s *State) GetBalance(address []byte) (*big.Int, error) {
	account, err := s.GetAccount(address)
	if err != nil {
		return big.NewInt(0), nil
	}
	return account.GetBalance(), nil
}

// SetBalance sets the balance of an account
func (s *State) SetBalance(address []byte, balance *big.Int) error {
	account, err := s.GetAccount(address)
	if err != nil {
		account = NewAccount()
		s.accounts[string(address)] = account
	}
	account.SetBalance(balance)
	return nil
}

// GetNonce gets the nonce of an account
func (s *State) GetNonce(address []byte) (uint64, error) {
	account, err := s.GetAccount(address)
	if err != nil {
		return 0, nil
	}
	return account.GetNonce(), nil
}

// SetNonce sets the nonce of an account
func (s *State) SetNonce(address []byte, nonce uint64) error {
	account, err := s.GetAccount(address)
	if err != nil {
		account = NewAccount()
		s.accounts[string(address)] = account
	}
	account.SetNonce(nonce)
	return nil
}

// GetCode gets the code of an account
func (s *State) GetCode(address []byte) ([]byte, error) {
	account, err := s.GetAccount(address)
	if err != nil {
		return nil, err
	}
	return account.GetCodeHash(), nil
}

// SetCode sets the code of an account
func (s *State) SetCode(address []byte, code []byte) error {
	account, err := s.GetAccount(address)
	if err != nil {
		account = NewAccount()
		s.accounts[string(address)] = account
	}
	account.SetCodeHash(code)
	return nil
}

// GetStorage gets the storage value of an account
func (s *State) GetStorage(address []byte, key string) ([]byte, error) {
	account, err := s.GetAccount(address)
	if err != nil {
		return nil, err
	}
	return account.GetStorage(key), nil
}

// SetStorage sets the storage value of an account
func (s *State) SetStorage(address []byte, key string, value []byte) error {
	account, err := s.GetAccount(address)
	if err != nil {
		account = NewAccount()
		s.accounts[string(address)] = account
	}
	account.SetStorage(key, value)
	return nil
}

// Root returns the state root
func (s *State) Root() []byte {
	if s.root == nil {
		s.updateRoot()
	}
	return s.root
}

// updateRoot updates the state root
func (s *State) updateRoot() {
	h := sha256.New()
	for addr, account := range s.accounts {
		h.Write([]byte(addr))
		h.Write(account.GetCodeHash())
		// TODO: Include storage in root calculation
	}
	s.root = h.Sum(nil)
} 