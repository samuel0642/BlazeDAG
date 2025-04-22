package test

import (
	"testing"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/transaction"
	"github.com/CrossDAG/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionPool(t *testing.T) {
	// Create state
	state := types.NewState()

	// Create transaction pool
	pool := transaction.NewPool(state)
	assert.NotNil(t, pool)

	// Create test accounts
	sender := &types.Account{
		Address: []byte("sender"),
		Balance: 1000,
		Nonce:   0,
	}

	recipient := &types.Account{
		Address: []byte("recipient"),
		Balance: 0,
		Nonce:   0,
	}

	// Set accounts in pool
	err := pool.SetAccount(sender.Address, sender)
	assert.NoError(t, err)

	err = pool.SetAccount(recipient.Address, recipient)
	assert.NoError(t, err)

	// Create transaction
	tx := &types.Transaction{
		From:     sender.Address,
		To:       recipient.Address,
		Value:    100,
		Nonce:    0,
		GasLimit: 21000,
		GasPrice: 1,
	}

	// Add transaction to pool
	err = pool.AddTransaction(tx)
	assert.NoError(t, err)

	// Get transaction from pool
	retrievedTx := pool.GetTransaction(tx.Hash())
	assert.NotNil(t, retrievedTx)
	assert.Equal(t, tx.Hash(), retrievedTx.Hash())

	// Process transaction
	result, err := pool.ProcessTransaction(tx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// Verify account balances
	senderAcc, err := pool.GetAccount(sender.Address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(900), senderAcc.Balance)
	assert.Equal(t, uint64(1), senderAcc.Nonce)

	recipientAcc, err := pool.GetAccount(recipient.Address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), recipientAcc.Balance)
	assert.Equal(t, uint64(0), recipientAcc.Nonce)
}

func TestTransactionExecution(t *testing.T) {
	pool := transaction.NewPool()

	// Create test accounts
	sender := &types.Account{
		Address: []byte("sender"),
		Balance: 1000,
		Nonce:   0,
	}
	recipient := &types.Account{
		Address: []byte("recipient"),
		Balance: 0,
		Nonce:   0,
	}

	// Add accounts to pool
	pool.SetAccount(sender)
	pool.SetAccount(recipient)

	// Create transaction with insufficient balance
	tx := &types.Transaction{
		From:     sender.Address,
		To:       recipient.Address,
		Value:    2000, // More than sender's balance
		Nonce:    0,
		GasLimit: 10,
		GasPrice: 1,
		Data:     []byte("test"),
	}

	// Test processing transaction with insufficient balance
	_, err := pool.ProcessTransaction(tx)
	assert.Error(t, err)
	assert.Equal(t, transaction.ErrInsufficientBalance, err)

	// Create transaction with invalid nonce
	tx = &types.Transaction{
		From:     sender.Address,
		To:       recipient.Address,
		Value:    100,
		Nonce:    1, // Invalid nonce
		GasLimit: 10,
		GasPrice: 1,
		Data:     []byte("test"),
	}

	// Test processing transaction with invalid nonce
	_, err = pool.ProcessTransaction(tx)
	assert.Error(t, err)
	assert.Equal(t, transaction.ErrInvalidNonce, err)
}

func TestTransactionBatchProcessing(t *testing.T) {
	pool := transaction.NewPool()

	// Create test accounts
	sender := &types.Account{
		Address: []byte("sender"),
		Balance: 1000,
		Nonce:   0,
	}
	recipient := &types.Account{
		Address: []byte("recipient"),
		Balance: 0,
		Nonce:   0,
	}

	// Add accounts to pool
	pool.SetAccount(sender)
	pool.SetAccount(recipient)

	// Create batch of transactions
	txs := make([]*types.Transaction, 3)
	for i := range txs {
		txs[i] = &types.Transaction{
			From:     sender.Address,
			To:       recipient.Address,
			Value:    100,
			Nonce:    uint64(i),
			GasLimit: 10,
			GasPrice: 1,
			Data:     []byte("test"),
		}
	}

	// Test batch processing
	results, err := pool.ProcessBatch(txs)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Verify results
	for i, result := range results {
		assert.True(t, result.Success)
		assert.Equal(t, txs[i].Hash(), result.TransactionHash)
		assert.Equal(t, uint64(10), result.GasUsed)
	}
}

func TestTransactionConflictDetection(t *testing.T) {
	pool := transaction.NewPool()

	// Create test accounts
	sender := &types.Account{
		Address: []byte("sender"),
		Balance: 1000,
		Nonce:   0,
	}
	recipient := &types.Account{
		Address: []byte("recipient"),
		Balance: 0,
		Nonce:   0,
	}

	// Add accounts to pool
	pool.SetAccount(sender)
	pool.SetAccount(recipient)

	// Create transactions with nonce conflicts
	txs := make([]*types.Transaction, 3)
	for i := range txs {
		txs[i] = &types.Transaction{
			From:     sender.Address,
			To:       recipient.Address,
			Value:    100,
			Nonce:    0, // Same nonce for all transactions
			GasLimit: 10,
			GasPrice: 1,
			Data:     []byte("test"),
		}
	}

	// Test conflict detection
	conflicts := pool.DetectConflicts(txs)
	assert.Len(t, conflicts, 2) // Should detect conflicts between txs[0] and txs[1], txs[0] and txs[2]
}

func TestTransactionPerformance(t *testing.T) {
	pool := transaction.NewPool()

	// Create test accounts
	sender := &types.Account{
		Address: []byte("sender"),
		Balance: 1000000, // Large balance for many transactions
		Nonce:   0,
	}
	recipient := &types.Account{
		Address: []byte("recipient"),
		Balance: 0,
		Nonce:   0,
	}

	// Add accounts to pool
	pool.SetAccount(sender)
	pool.SetAccount(recipient)

	// Create large batch of transactions
	const numTxs = 1000
	txs := make([]*types.Transaction, numTxs)
	for i := range txs {
		txs[i] = &types.Transaction{
			From:     sender.Address,
			To:       recipient.Address,
			Value:    100,
			Nonce:    uint64(i),
			GasLimit: 10,
			GasPrice: 1,
			Data:     []byte("test"),
		}
	}

	// Test batch processing performance
	start := time.Now()
	results, err := pool.ProcessBatch(txs)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, results, numTxs)
	assert.True(t, duration < time.Second, "Batch processing took too long: %v", duration)
}

func TestTransactionGasHandling(t *testing.T) {
	pool := transaction.NewPool()

	// Create test accounts
	sender := &types.Account{
		Address: []byte("sender"),
		Balance: 1000,
		Nonce:   0,
	}
	recipient := &types.Account{
		Address: []byte("recipient"),
		Balance: 0,
		Nonce:   0,
	}

	// Add accounts to pool
	pool.SetAccount(sender)
	pool.SetAccount(recipient)

	// Create transaction with high gas price
	tx := &types.Transaction{
		From:     sender.Address,
		To:       recipient.Address,
		Value:    100,
		Nonce:    0,
		GasLimit: 10,
		GasPrice: 100, // High gas price
		Data:     []byte("test"),
	}

	// Test processing transaction with high gas price
	result, err := pool.ProcessTransaction(tx)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, uint64(10), result.GasUsed)
	assert.Equal(t, uint64(0), result.SenderBalance) // 1000 - 100 - (10 * 100)
}

func TestTransactionReceiptGeneration(t *testing.T) {
	pool := transaction.NewPool()

	// Create test transaction
	tx := &types.Transaction{
		From:     []byte("sender"),
		To:       []byte("recipient"),
		Value:    100,
		Nonce:    0,
		GasLimit: 10,
		GasPrice: 1,
		Data:     []byte("test"),
	}

	// Test receipt generation
	blockHash := []byte("block123")
	blockNumber := uint64(1)
	receipt := pool.GenerateReceipt(tx, blockHash, blockNumber)

	assert.Equal(t, tx.Hash(), receipt.TransactionHash)
	assert.True(t, receipt.Success)
	assert.Equal(t, uint64(10), receipt.GasUsed)
	assert.Equal(t, blockHash, receipt.BlockHash)
	assert.Equal(t, blockNumber, receipt.BlockNumber)
} 