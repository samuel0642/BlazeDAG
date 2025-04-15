package types

// Account represents an account in the blockchain
type Account struct {
	Address  []byte
	Balance  uint64
	Nonce    uint64
	Code     []byte
	Storage  map[string][]byte
}

// NewAccount creates a new account
func NewAccount(address []byte, balance uint64) *Account {
	return &Account{
		Address: address,
		Balance: balance,
		Nonce:   0,
		Code:    nil,
		Storage: make(map[string][]byte),
	}
}

// BalanceUpdate represents a balance update for an account
type BalanceUpdate struct {
	Address []byte
	Balance uint64
} 