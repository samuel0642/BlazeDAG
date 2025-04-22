package evm

import (
	"math/big"
	"sync"
)

// ExecutionResult represents the result of a transaction execution
type ExecutionResult struct {
	Success     bool
	GasUsed     uint64
	ReturnData  []byte
	Error       error
	StateRoot   []byte
}

// EVMExecutor handles EVM transaction execution
type EVMExecutor struct {
	state       *State
	gasPrice    *big.Int
	blockNumber *big.Int
	mu          sync.RWMutex
}

// NewEVMExecutor creates a new EVM executor
func NewEVMExecutor(state *State, gasPrice *big.Int) *EVMExecutor {
	return &EVMExecutor{
		state:       state,
		gasPrice:    gasPrice,
		blockNumber: big.NewInt(0),
	}
}

// ExecuteTransaction executes a transaction
func (e *EVMExecutor) ExecuteTransaction(tx *Transaction) *ExecutionResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check gas limit
	if tx.GasLimit < tx.GasPrice {
		return &ExecutionResult{
			Success: false,
			Error:   ErrInsufficientGas,
		}
	}

	// Check nonce
	nonce, err := e.state.GetNonce(tx.From)
	if err != nil || nonce != tx.Nonce {
		return &ExecutionResult{
			Success: false,
			Error:   ErrInvalidNonce,
		}
	}

	// Check balance
	balance, err := e.state.GetBalance(tx.From)
	if err != nil || balance.Cmp(tx.Value) < 0 {
		return &ExecutionResult{
			Success: false,
			Error:   ErrInsufficientBalance,
		}
	}

	// Execute transaction
	result := e.execute(tx)
	if result.Success {
		// Update nonce
		e.state.SetNonce(tx.From, nonce+1)
		// Update balance
		e.state.SetBalance(tx.From, new(big.Int).Sub(balance, tx.Value))
		// Update recipient balance
		recipientBalance, _ := e.state.GetBalance(tx.To)
		e.state.SetBalance(tx.To, new(big.Int).Add(recipientBalance, tx.Value))
	}

	return result
}

// execute executes a transaction
func (e *EVMExecutor) execute(tx *Transaction) *ExecutionResult {
	// Create execution context
	context := &ExecutionContext{
		State:       e.state,
		GasPrice:    e.gasPrice,
		BlockNumber: e.blockNumber,
		Timestamp:   tx.Timestamp,
	}

	// Execute based on transaction type
	if len(tx.Data) == 0 {
		// Simple transfer
		return e.executeTransfer(tx, context)
	} else {
		// Contract execution
		return e.executeContract(tx, context)
	}
}

// executeTransfer executes a simple transfer
func (e *EVMExecutor) executeTransfer(tx *Transaction, context *ExecutionContext) *ExecutionResult {
	// Calculate gas cost
	gasCost := new(big.Int).Mul(e.gasPrice, big.NewInt(int64(tx.GasLimit)))
	
	// Check if sender has enough balance for gas
	balance, _ := e.state.GetBalance(tx.From)
	if balance.Cmp(gasCost) < 0 {
		return &ExecutionResult{
			Success: false,
			Error:   ErrInsufficientGas,
		}
	}

	// Deduct gas cost
	e.state.SetBalance(tx.From, new(big.Int).Sub(balance, gasCost))

	return &ExecutionResult{
		Success:   true,
		GasUsed:   tx.GasLimit,
		StateRoot: e.state.Root(),
	}
}

// executeContract executes a contract transaction
func (e *EVMExecutor) executeContract(tx *Transaction, context *ExecutionContext) *ExecutionResult {
	// Get contract code
	code, err := e.state.GetCode(tx.To)
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   ErrContractNotFound,
		}
	}

	// Create contract instance
	contract := NewContract(code, tx.Data)

	// Execute contract
	result, err := contract.Execute(context)
	if err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   err,
		}
	}

	return &ExecutionResult{
		Success:    true,
		GasUsed:    result.GasUsed,
		ReturnData: result.ReturnData,
		StateRoot:  e.state.Root(),
	}
}

// UpdateBlockNumber updates the block number
func (e *EVMExecutor) UpdateBlockNumber(blockNumber *big.Int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.blockNumber = blockNumber
}

// UpdateGasPrice updates the gas price
func (e *EVMExecutor) UpdateGasPrice(gasPrice *big.Int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.gasPrice = gasPrice
} 