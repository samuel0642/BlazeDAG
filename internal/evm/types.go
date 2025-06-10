package evm

import (
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Transaction represents an EVM transaction with Ethereum compatibility
type Transaction struct {
	From      common.Address
	To        *common.Address // nil for contract creation
	Value     *big.Int
	GasPrice  *big.Int
	GasLimit  uint64
	Nonce     uint64
	Data      []byte
	Timestamp time.Time
	Hash      common.Hash
	Signature []byte
}

// ExecutionContext represents the context for transaction execution
type ExecutionContext struct {
	State       *State
	GasPrice    *big.Int
	BlockNumber *big.Int
	Timestamp   time.Time
	BlockHash   common.Hash
	Coinbase    common.Address
}

// Note: ExecutionResult is defined in execution.go

// Contract represents an EVM contract
type Contract struct {
	Address common.Address
	Code    []byte
	Storage map[common.Hash]common.Hash
}

// NewContract creates a new contract instance
func NewContract(address common.Address, code []byte) *Contract {
	return &Contract{
		Address: address,
		Code:    code,
		Storage: make(map[common.Hash]common.Hash),
	}
}

// CreateContractAddress creates a new contract address from deployer and nonce
func CreateContractAddress(deployer common.Address, nonce uint64) common.Address {
	return crypto.CreateAddress(deployer, nonce)
}

// CreateTransactionHash creates a hash for the transaction
func (tx *Transaction) CreateHash() common.Hash {
	// Simple hash creation - in production, use proper RLP encoding
	data := append(tx.From.Bytes(), tx.Data...)
	if tx.To != nil {
		data = append(data, tx.To.Bytes()...)
	}
	return crypto.Keccak256Hash(data)
}

// ToEthTransaction converts to ethereum transaction type
func (tx *Transaction) ToEthTransaction() *types.Transaction {
	if tx.To == nil {
		// Contract creation
		return types.NewContractCreation(tx.Nonce, tx.Value, tx.GasLimit, tx.GasPrice, tx.Data)
	}
	return types.NewTransaction(tx.Nonce, *tx.To, tx.Value, tx.GasLimit, tx.GasPrice, tx.Data)
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
	ErrInvalidSignature    = errors.New("invalid signature")
	ErrContractCreationFailed = errors.New("contract creation failed")
) 