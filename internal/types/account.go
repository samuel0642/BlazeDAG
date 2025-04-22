package types

// Account represents an account in the state
type Account struct {
	Address     []byte
	Balance     uint64
	Nonce       uint64
	Code        []byte
	StorageRoot []byte
	CodeHash    []byte
}

// NewAccount creates a new account
func NewAccount(address []byte, balance uint64) *Account {
	return &Account{
		Address:     address,
		Balance:     balance,
		Nonce:       0,
		Code:        nil,
		StorageRoot: nil,
		CodeHash:    nil,
	}
}

// BalanceUpdate represents a balance update for an account
type BalanceUpdate struct {
	Address []byte
	Balance uint64
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