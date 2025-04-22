package core

import (
	"encoding/hex"
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// Mempool represents the transaction pool
type Mempool struct {
	transactions map[string]*types.Transaction
	mu           sync.RWMutex
}

// NewMempool creates a new mempool
func NewMempool() *Mempool {
	return &Mempool{
		transactions: make(map[string]*types.Transaction),
	}
}

// AddTransaction adds a transaction to the mempool
func (m *Mempool) AddTransaction(tx *types.Transaction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions[hex.EncodeToString(tx.Hash())] = tx
}

// GetTransactions returns all transactions in the mempool
func (m *Mempool) GetTransactions() []*types.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txs := make([]*types.Transaction, 0, len(m.transactions))
	for _, tx := range m.transactions {
		txs = append(txs, tx)
	}
	return txs
}

// RemoveTransaction removes a transaction from the mempool
func (m *Mempool) RemoveTransaction(txHash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.transactions, txHash)
}

// Clear clears the mempool
func (m *Mempool) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions = make(map[string]*types.Transaction)
} 