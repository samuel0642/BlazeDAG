package transaction

import (
	"container/heap"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

var (
	// ErrAccountNotFound is returned when an account is not found
	ErrAccountNotFound = errors.New("account not found")
	// ErrInvalidNonce is returned when a transaction has an invalid nonce
	ErrInvalidNonce = errors.New("invalid nonce")
	// ErrInsufficientBalance is returned when an account has insufficient balance
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrInvalidTransaction is returned when a transaction is invalid
	ErrInvalidTransaction = errors.New("invalid transaction")
	// ErrInsufficientFunds is returned when an account has insufficient funds
	ErrInsufficientFunds = errors.New("insufficient funds")
	// ErrDuplicateTx is returned when a transaction is duplicated
	ErrDuplicateTx = errors.New("duplicate transaction")
	// ErrPoolFull is returned when the transaction pool is full
	ErrPoolFull = errors.New("transaction pool is full")
)

// Transaction represents a blockchain transaction
type Transaction struct {
	Hash      string
	From      string
	To        string
	Value     uint64
	GasPrice  uint64
	GasLimit  uint64
	Nonce     uint64
	Data      []byte
	Signature []byte
	Timestamp time.Time
}

// TransactionPool manages the pool of pending transactions
type TransactionPool struct {
	pending    map[string]*Transaction
	validated  map[string]*Transaction
	rejected   map[string]*Transaction
	priority   *PriorityQueue
	mu         sync.RWMutex
}

// PriorityQueue implements heap.Interface for transaction priority
type PriorityQueue []*Transaction

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Higher gas price transactions have higher priority
	return pq[i].GasPrice > pq[j].GasPrice
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*Transaction)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

// Pool represents the transaction pool
type Pool struct {
	state    *state.State
	txs      map[string]*types.Transaction
	maxSize  int
	mu       sync.RWMutex
}

// NewPool creates a new transaction pool
func NewPool(state *state.State, maxSize int) *Pool {
	return &Pool{
		state:   state,
		txs:     make(map[string]*types.Transaction),
		maxSize: maxSize,
	}
}

// NewTransactionPool creates a new transaction pool
func NewTransactionPool() *TransactionPool {
	pq := &PriorityQueue{}
	heap.Init(pq)
	return &TransactionPool{
		pending:   make(map[string]*Transaction),
		validated: make(map[string]*Transaction),
		rejected:  make(map[string]*Transaction),
		priority:  pq,
	}
}

// AddTransaction adds a transaction to the pool
func (p *Pool) AddTransaction(tx *types.Transaction) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.txs) >= p.maxSize {
		return ErrPoolFull
	}

	if err := p.ValidateTransaction(tx); err != nil {
		return err
	}

	txHash := string(tx.GetHash())
	if _, exists := p.txs[txHash]; exists {
		return ErrDuplicateTx
	}

	p.txs[txHash] = tx
	return nil
}

// GetTransaction retrieves a transaction by its hash
func (p *Pool) GetTransaction(hash []byte) (*types.Transaction, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tx, exists := p.txs[string(hash)]
	return tx, exists
}

// RemoveTransaction removes a transaction from the pool
func (p *Pool) RemoveTransaction(hash []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.txs, string(hash))
}

// ValidateTransaction validates a transaction
func (p *Pool) ValidateTransaction(tx *types.Transaction) error {
	// Check if sender exists
	account, err := p.state.GetAccount(string(tx.From))
	if err != nil {
		return fmt.Errorf("sender account not found: %v", err)
	}

	if account.Nonce != tx.Nonce {
		return ErrInvalidNonce
	}

	if account.Balance < tx.Value {
		return ErrInsufficientFunds
	}

	return nil
}

// GetTransactions returns all transactions in the pool
func (p *Pool) GetTransactions() []*types.Transaction {
	p.mu.RLock()
	defer p.mu.RUnlock()

	txs := make([]*types.Transaction, 0, len(p.txs))
	for _, tx := range p.txs {
		txs = append(txs, tx)
	}
	return txs
}

// Clear clears the transaction pool
func (p *Pool) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.txs = make(map[string]*types.Transaction)
}

// GetAccount gets an account from the state
func (p *Pool) GetAccount(address []byte) (*types.Account, error) {
	return p.state.GetAccount(string(address))
}

// SetAccount sets an account in the state
func (p *Pool) SetAccount(address []byte, account *types.Account) {
	p.state.SetAccount(string(address), account)
}

// byteToUint64 converts a byte slice to uint64
func byteToUint64(b []byte) uint64 {
	if len(b) < 8 {
		padded := make([]byte, 8)
		copy(padded[8-len(b):], b)
		b = padded
	}
	return binary.BigEndian.Uint64(b)
}

// uint64ToBytes converts uint64 to a byte slice
func uint64ToBytes(u uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, u)
	return b
}

// ProcessTransaction processes a transaction
func (p *Pool) ProcessTransaction(tx *types.Transaction) (*types.TransactionResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Validate transaction
	if err := p.ValidateTransaction(tx); err != nil {
		return nil, err
	}

	// Get account states
	sender, err := p.GetAccount(tx.From)
	if err != nil {
		return nil, err
	}
	
	recipient, err := p.GetAccount(tx.To)
	if err != nil {
		// Create new recipient account if it doesn't exist
		recipient = &types.Account{
			Balance: 0,
			Nonce:   0,
		}
	}

	// Update balances
	sender.Balance -= tx.Value
	recipient.Balance += tx.Value
	sender.Nonce++

	// Update accounts in state
	p.SetAccount(tx.From, sender)
	p.SetAccount(tx.To, recipient)

	// Create transaction result
	result := &types.TransactionResult{
		Success:          true,
		TransactionHash:  tx.GetHash(),
		GasUsed:         tx.GasLimit,
		SenderBalance:    sender.Balance,
		RecipientBalance: recipient.Balance,
		SenderNonce:      sender.Nonce,
	}

	return result, nil
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

func (p *Pool) HasAccount(address []byte) bool {
	_, err := p.state.GetAccount(string(address))
	return err == nil
}

// AddTransaction adds a transaction to the pool
func (tp *TransactionPool) AddTransaction(tx *Transaction) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tp.pending[tx.Hash] = tx
	heap.Push(tp.priority, tx)
}

// RemoveTransaction removes a transaction from the pool
func (tp *TransactionPool) RemoveTransaction(hash string) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	delete(tp.pending, hash)
	delete(tp.validated, hash)
	delete(tp.rejected, hash)
}

// GetTransaction returns a transaction by hash
func (tp *TransactionPool) GetTransaction(hash string) *Transaction {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	if tx, exists := tp.pending[hash]; exists {
		return tx
	}
	if tx, exists := tp.validated[hash]; exists {
		return tx
	}
	return nil
}

// GetPendingTransactions returns all pending transactions
func (tp *TransactionPool) GetPendingTransactions() []*Transaction {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	txs := make([]*Transaction, 0, len(tp.pending))
	for _, tx := range tp.pending {
		txs = append(txs, tx)
	}
	return txs
}

// GetValidatedTransactions returns all validated transactions
func (tp *TransactionPool) GetValidatedTransactions() []*Transaction {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	txs := make([]*Transaction, 0, len(tp.validated))
	for _, tx := range tp.validated {
		txs = append(txs, tx)
	}
	return txs
}

// GetNextTransaction returns the next transaction to process
func (tp *TransactionPool) GetNextTransaction() *Transaction {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tp.priority.Len() == 0 {
		return nil
	}

	tx := heap.Pop(tp.priority).(*Transaction)
	delete(tp.pending, tx.Hash)
	return tx
}

// UpdateTransactionPriority updates a transaction's priority
func (tp *TransactionPool) UpdateTransactionPriority(hash string, newGasPrice uint64) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tx, exists := tp.pending[hash]; exists {
		tx.GasPrice = newGasPrice
		heap.Fix(tp.priority, 0)
	}
}

// MarkTransactionValidated marks a transaction as validated
func (tp *TransactionPool) MarkTransactionValidated(hash string) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tx, exists := tp.pending[hash]; exists {
		tp.validated[hash] = tx
		delete(tp.pending, hash)
	}
}

// MarkTransactionRejected marks a transaction as rejected
func (tp *TransactionPool) MarkTransactionRejected(hash string) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tx, exists := tp.pending[hash]; exists {
		tp.rejected[hash] = tx
		delete(tp.pending, hash)
	}
} 