package evm

import (
	"math/big"
)

// Account represents an EVM account
type Account struct {
	Nonce    uint64
	Balance  *big.Int
	CodeHash []byte
	Storage  map[string][]byte
}

// NewAccount creates a new EVM account
func NewAccount() *Account {
	return &Account{
		Nonce:    0,
		Balance:  big.NewInt(0),
		CodeHash: []byte{},
		Storage:  make(map[string][]byte),
	}
}

// GetBalance returns the account's balance
func (a *Account) GetBalance() *big.Int {
	return a.Balance
}

// SetBalance sets the account's balance
func (a *Account) SetBalance(balance *big.Int) {
	a.Balance = balance
}

// GetNonce returns the account's nonce
func (a *Account) GetNonce() uint64 {
	return a.Nonce
}

// SetNonce sets the account's nonce
func (a *Account) SetNonce(nonce uint64) {
	a.Nonce = nonce
}

// GetCodeHash returns the account's code hash
func (a *Account) GetCodeHash() []byte {
	return a.CodeHash
}

// SetCodeHash sets the account's code hash
func (a *Account) SetCodeHash(hash []byte) {
	a.CodeHash = hash
}

// GetStorage returns the value at the given storage key
func (a *Account) GetStorage(key string) []byte {
	return a.Storage[key]
}

// SetStorage sets the value at the given storage key
func (a *Account) SetStorage(key string, value []byte) {
	a.Storage[key] = value
} 