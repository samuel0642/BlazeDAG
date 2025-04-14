package types

// Account represents an account in the BlazeDAG system
type Account struct {
	Address     []byte `json:"address"`
	Balance     []byte `json:"balance"`
	Nonce       uint64 `json:"nonce"`
	CodeHash    []byte `json:"codeHash"`
	StorageRoot []byte `json:"storageRoot"`
} 