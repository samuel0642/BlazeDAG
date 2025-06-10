package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Account represents an EVM account with Ethereum compatibility
type Account struct {
	Address  common.Address
	Nonce    uint64
	Balance  *big.Int
	CodeHash common.Hash
	Code     []byte
	Storage  map[common.Hash]common.Hash
}

// NewAccount creates a new EVM account
func NewAccount(address common.Address) *Account {
	return &Account{
		Address:  address,
		Nonce:    0,
		Balance:  big.NewInt(0),
		CodeHash: common.Hash{},
		Code:     []byte{},
		Storage:  make(map[common.Hash]common.Hash),
	}
}

// NewAccountFromPrivateKey creates an account from a private key
func NewAccountFromPrivateKey(privateKey []byte) (*Account, error) {
	privKey, err := crypto.ToECDSA(privateKey)
	if err != nil {
		return nil, err
	}
	
	address := crypto.PubkeyToAddress(privKey.PublicKey)
	return NewAccount(address), nil
}

// GetAddress returns the account's address
func (a *Account) GetAddress() common.Address {
	return a.Address
}

// GetBalance returns the account's balance
func (a *Account) GetBalance() *big.Int {
	if a.Balance == nil {
		a.Balance = big.NewInt(0)
	}
	return a.Balance
}

// SetBalance sets the account's balance
func (a *Account) SetBalance(balance *big.Int) {
	a.Balance = new(big.Int).Set(balance)
}

// AddBalance adds to the account's balance
func (a *Account) AddBalance(amount *big.Int) {
	if a.Balance == nil {
		a.Balance = big.NewInt(0)
	}
	a.Balance.Add(a.Balance, amount)
}

// SubBalance subtracts from the account's balance
func (a *Account) SubBalance(amount *big.Int) bool {
	if a.Balance == nil {
		a.Balance = big.NewInt(0)
	}
	if a.Balance.Cmp(amount) < 0 {
		return false
	}
	a.Balance.Sub(a.Balance, amount)
	return true
}

// GetNonce returns the account's nonce
func (a *Account) GetNonce() uint64 {
	return a.Nonce
}

// SetNonce sets the account's nonce
func (a *Account) SetNonce(nonce uint64) {
	a.Nonce = nonce
}

// IncrementNonce increments the account's nonce
func (a *Account) IncrementNonce() {
	a.Nonce++
}

// GetCodeHash returns the account's code hash
func (a *Account) GetCodeHash() common.Hash {
	return a.CodeHash
}

// SetCodeHash sets the account's code hash
func (a *Account) SetCodeHash(hash common.Hash) {
	a.CodeHash = hash
}

// GetCode returns the account's code
func (a *Account) GetCode() []byte {
	return a.Code
}

// SetCode sets the account's code and updates the code hash
func (a *Account) SetCode(code []byte) {
	a.Code = make([]byte, len(code))
	copy(a.Code, code)
	a.CodeHash = crypto.Keccak256Hash(code)
}

// IsContract returns true if the account is a contract
func (a *Account) IsContract() bool {
	return len(a.Code) > 0
}

// GetStorage returns the value at the given storage key
func (a *Account) GetStorage(key common.Hash) common.Hash {
	return a.Storage[key]
}

// SetStorage sets the value at the given storage key
func (a *Account) SetStorage(key common.Hash, value common.Hash) {
	if a.Storage == nil {
		a.Storage = make(map[common.Hash]common.Hash)
	}
	a.Storage[key] = value
}

// Copy creates a deep copy of the account
func (a *Account) Copy() *Account {
	acc := &Account{
		Address:  a.Address,
		Nonce:    a.Nonce,
		Balance:  new(big.Int).Set(a.Balance),
		CodeHash: a.CodeHash,
		Code:     make([]byte, len(a.Code)),
		Storage:  make(map[common.Hash]common.Hash),
	}
	
	copy(acc.Code, a.Code)
	for k, v := range a.Storage {
		acc.Storage[k] = v
	}
	
	return acc
} 