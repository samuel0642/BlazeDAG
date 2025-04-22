package evm

import (
	"errors"
	"math/big"
	"time"
)

// Transaction represents an EVM transaction
type Transaction struct {
	From      []byte
	To        []byte
	Value     *big.Int
	GasPrice  *big.Int
	GasLimit  uint64
	Nonce     uint64
	Data      []byte
	Timestamp time.Time
}

// ExecutionContext represents the context for transaction execution
type ExecutionContext struct {
	State       *State
	GasPrice    *big.Int
	BlockNumber *big.Int
	Timestamp   time.Time
}

// Contract represents an EVM contract
type Contract struct {
	Code []byte
	Data []byte
}

// NewContract creates a new contract instance
func NewContract(code, data []byte) *Contract {
	return &Contract{
		Code: code,
		Data: data,
	}
}

// Execute executes the contract
func (c *Contract) Execute(context *ExecutionContext) (*ExecutionResult, error) {
	// TODO: Implement contract execution
	return &ExecutionResult{
		Success:   true,
		GasUsed:   0,
		ReturnData: []byte{},
		StateRoot:  context.State.Root(),
	}, nil
}

// Errors
var (
	ErrInsufficientGas     = errors.New("insufficient gas")
	ErrInvalidNonce        = errors.New("invalid nonce")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrContractNotFound    = errors.New("contract not found")
) 