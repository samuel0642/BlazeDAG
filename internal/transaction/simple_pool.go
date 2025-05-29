package transaction

import (
	"errors"
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

var (
	// ErrPoolFull is returned when the transaction pool is full
	ErrPoolFull = errors.New("transaction pool is full")
	// ErrDuplicateTx is returned when a transaction is duplicated
	ErrDuplicateTx = errors.New("duplicate transaction")
)

// SimplePool represents a simple transaction pool for independent operation
type SimplePool struct {
	txs     map[string]*types.Transaction
	maxSize int
	mu      sync.RWMutex
}

// NewSimplePool creates a new simple transaction pool
func NewSimplePool() *SimplePool {
	return &SimplePool{
		txs:     make(map[string]*types.Transaction),
		maxSize: 10000, // Default max size
	}
}

// Add adds a transaction to the pool
func (p *SimplePool) Add(tx *types.Transaction) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.txs) >= p.maxSize {
		return ErrPoolFull
	}

	txHash := string(tx.GetHash())
	if _, exists := p.txs[txHash]; exists {
		return ErrDuplicateTx
	}

	p.txs[txHash] = tx
	return nil
}

// GetPending returns up to maxCount pending transactions
func (p *SimplePool) GetPending(maxCount int) []*types.Transaction {
	p.mu.RLock()
	defer p.mu.RUnlock()

	txs := make([]*types.Transaction, 0, maxCount)
	count := 0
	for _, tx := range p.txs {
		if count >= maxCount {
			break
		}
		txs = append(txs, tx)
		count++
	}
	return txs
}

// GetPendingCount returns the number of pending transactions
func (p *SimplePool) GetPendingCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.txs)
}

// Clear clears the transaction pool
func (p *SimplePool) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.txs = make(map[string]*types.Transaction)
} 