package transaction

import (
	"errors"
	"sync"
	"time"

	"BlazeDAG/internal/types"
)

// Pool represents a pool of transactions
type Pool struct {
	mu           sync.RWMutex
	transactions map[string]*types.Transaction
	accounts     map[string]*types.Account
}

// NewPool creates a new transaction pool
func NewPool() *Pool {
	return &Pool{
		transactions: make(map[string]*types.Transaction),
		accounts:     make(map[string]*types.Account),
	}
}

// AddTransaction adds a transaction to the pool
func (p *Pool) AddTransaction(tx *types.Transaction) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Validate transaction
	if err := p.validateTransaction(tx); err != nil {
		return err
	}

	// Add transaction to pool
	txHash := string(tx.Hash())
	p.transactions[txHash] = tx

	return nil
}

// GetTransaction retrieves a transaction by its hash
func (p *Pool) GetTransaction(hash []byte) (*types.Transaction, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tx, exists := p.transactions[string(hash)]
	return tx, exists
}

// RemoveTransaction removes a transaction from the pool
func (p *Pool) RemoveTransaction(hash []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.transactions, string(hash))
}

// GetAccount retrieves an account by its address
func (p *Pool) GetAccount(address []byte) (*types.Account, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	account, exists := p.accounts[string(address)]
	return account, exists
}

// SetAccount sets an account in the pool
func (p *Pool) SetAccount(account *types.Account) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.accounts[string(account.Address)] = account
}

// validateTransaction validates a transaction
func (p *Pool) validateTransaction(tx *types.Transaction) error {
	// Check if sender account exists
	sender, exists := p.GetAccount(tx.From)
	if !exists {
		return ErrAccountNotFound
	}

	// Check nonce
	if tx.Nonce != sender.Nonce {
		return ErrInvalidNonce
	}

	// Check balance
	totalCost := tx.Value + (tx.GasLimit * tx.GasPrice)
	if sender.Balance < totalCost {
		return ErrInsufficientBalance
	}

	return nil
}

// ProcessTransaction processes a transaction and returns the result
func (p *Pool) ProcessTransaction(tx *types.Transaction) (*types.TransactionResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Get sender and recipient accounts
	sender, exists := p.GetAccount(tx.From)
	if !exists {
		return nil, ErrAccountNotFound
	}

	recipient, exists := p.GetAccount(tx.To)
	if !exists {
		recipient = &types.Account{
			Address: tx.To,
			Balance: 0,
			Nonce:   0,
		}
	}

	// Calculate gas cost
	gasCost := tx.GasLimit * tx.GasPrice
	totalCost := tx.Value + gasCost

	// Update balances
	sender.Balance -= totalCost
	recipient.Balance += tx.Value

	// Update nonce
	sender.Nonce++

	// Save accounts
	p.SetAccount(sender)
	p.SetAccount(recipient)

	// Remove transaction from pool
	delete(p.transactions, string(tx.Hash()))

	// Return result
	return &types.TransactionResult{
		Success:         true,
		TransactionHash: tx.Hash(),
		GasUsed:         tx.GasLimit,
		SenderBalance:   sender.Balance,
		RecipientBalance: recipient.Balance,
		SenderNonce:     sender.Nonce,
	}, nil
}

// ProcessBatch processes a batch of transactions
func (p *Pool) ProcessBatch(txs []*types.Transaction) ([]*types.TransactionResult, error) {
	results := make([]*types.TransactionResult, 0, len(txs))

	for _, tx := range txs {
		result, err := p.ProcessTransaction(tx)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// DetectConflicts detects conflicts between transactions
func (p *Pool) DetectConflicts(txs []*types.Transaction) []*types.TransactionConflict {
	conflicts := make([]*types.TransactionConflict, 0)
	nonceMap := make(map[string]uint64)

	for i, tx1 := range txs {
		// Check nonce conflicts
		prevNonce, exists := nonceMap[string(tx1.From)]
		if exists && tx1.Nonce != prevNonce+1 {
			for j := 0; j < i; j++ {
				tx2 := txs[j]
				if string(tx1.From) == string(tx2.From) {
					conflicts = append(conflicts, &types.TransactionConflict{
						Transaction1: tx1.Hash(),
						Transaction2: tx2.Hash(),
						Type:         types.ConflictTypeNonce,
					})
				}
			}
		}
		nonceMap[string(tx1.From)] = tx1.Nonce
	}

	return conflicts
}

// GenerateReceipt generates a receipt for a processed transaction
func (p *Pool) GenerateReceipt(tx *types.Transaction, blockHash []byte, blockNumber uint64) *types.TransactionReceipt {
	return &types.TransactionReceipt{
		TransactionHash: tx.Hash(),
		Success:         true,
		GasUsed:         tx.GasLimit,
		BlockHash:       blockHash,
		BlockNumber:     blockNumber,
	}
}

// Errors
var (
	ErrAccountNotFound    = errors.New("account not found")
	ErrInvalidNonce       = errors.New("invalid nonce")
	ErrInsufficientBalance = errors.New("insufficient balance")
) 