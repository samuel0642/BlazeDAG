package types

// Account represents an account in the state
type Account struct {
	Address     Address
	Balance     Value
	Nonce       Nonce
	Code        []byte
	StorageRoot Hash
	CodeHash    Hash
}

// NewAccount creates a new account
func NewAccount(address Address, balance Value) *Account {
	return &Account{
		Address:     address,
		Balance:     balance,
		Nonce:       0,
		Code:        nil,
		StorageRoot: make(Hash, 0),
		CodeHash:    make(Hash, 0),
	}
}

// BalanceUpdate represents a balance update for an account
type BalanceUpdate struct {
	Address Address
	Balance Value
}

// Equal checks if two accounts are equal
func (a *Account) Equal(other *Account) bool {
	if a == nil || other == nil {
		return a == other
	}
	return string(a.Address) == string(other.Address) &&
		a.Balance == other.Balance &&
		a.Nonce == other.Nonce &&
		string(a.Code) == string(other.Code) &&
		string(a.StorageRoot) == string(other.StorageRoot) &&
		string(a.CodeHash) == string(other.CodeHash)
} 