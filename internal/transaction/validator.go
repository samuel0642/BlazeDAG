package transaction

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"math/big"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrInvalidNonce     = errors.New("invalid nonce")
	ErrInsufficientGas  = errors.New("insufficient gas")
	ErrInvalidGasPrice  = errors.New("invalid gas price")
)

// Validator handles transaction validation
type Validator struct {
	state State
}

// State interface for accessing account state
type State interface {
	GetNonce(address string) (uint64, error)
	GetBalance(address string) (*big.Int, error)
	GetCode(address string) ([]byte, error)
}

// NewValidator creates a new transaction validator
func NewValidator(state State) *Validator {
	return &Validator{
		state: state,
	}
}

// ValidateTransaction validates a transaction
func (v *Validator) ValidateTransaction(tx *Transaction) error {
	// Validate signature
	if err := v.validateSignature(tx); err != nil {
		return err
	}

	// Validate nonce
	if err := v.validateNonce(tx); err != nil {
		return err
	}

	// Validate gas
	if err := v.validateGas(tx); err != nil {
		return err
	}

	// Validate balance
	if err := v.validateBalance(tx); err != nil {
		return err
	}

	return nil
}

// validateSignature validates the transaction signature
func (v *Validator) validateSignature(tx *Transaction) error {
	// Create hash of transaction data
	hash := sha256.Sum256(tx.Data)

	// Verify signature
	pubKey, err := ecdsa.Verify(tx.Signature, hash[:])
	if err != nil {
		return ErrInvalidSignature
	}

	// Verify sender matches public key
	if pubKey != tx.From {
		return ErrInvalidSignature
	}

	return nil
}

// validateNonce validates the transaction nonce
func (v *Validator) validateNonce(tx *Transaction) error {
	expectedNonce, err := v.state.GetNonce(tx.From)
	if err != nil {
		return err
	}

	if tx.Nonce != expectedNonce {
		return ErrInvalidNonce
	}

	return nil
}

// validateGas validates the transaction gas
func (v *Validator) validateGas(tx *Transaction) error {
	// Check gas price
	if tx.GasPrice == 0 {
		return ErrInvalidGasPrice
	}

	// Check gas limit
	if tx.GasLimit == 0 {
		return ErrInsufficientGas
	}

	// Calculate total gas cost
	totalGasCost := new(big.Int).Mul(
		new(big.Int).SetUint64(tx.GasPrice),
		new(big.Int).SetUint64(tx.GasLimit),
	)

	// Check if sender has enough balance for gas
	balance, err := v.state.GetBalance(tx.From)
	if err != nil {
		return err
	}

	if balance.Cmp(totalGasCost) < 0 {
		return ErrInsufficientGas
	}

	return nil
}

// validateBalance validates the transaction balance
func (v *Validator) validateBalance(tx *Transaction) error {
	// Get sender's balance
	balance, err := v.state.GetBalance(tx.From)
	if err != nil {
		return err
	}

	// Calculate total cost (value + gas)
	totalCost := new(big.Int).Add(
		new(big.Int).SetUint64(tx.Value),
		new(big.Int).Mul(
			new(big.Int).SetUint64(tx.GasPrice),
			new(big.Int).SetUint64(tx.GasLimit),
		),
	)

	// Check if sender has enough balance
	if balance.Cmp(totalCost) < 0 {
		return ErrInsufficientGas
	}

	return nil
}

// BatchValidateTransactions validates multiple transactions
func (v *Validator) BatchValidateTransactions(txs []*Transaction) []error {
	errors := make([]error, len(txs))
	for i, tx := range txs {
		errors[i] = v.ValidateTransaction(tx)
	}
	return errors
} 