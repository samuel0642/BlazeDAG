package transaction

import (
	"testing"

	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_NewPool(t *testing.T) {
	state := state.NewState()
	pool := NewPool(state, 100)

	assert.NotNil(t, pool)
	assert.NotNil(t, pool.state)
}

func TestPool_GetSetAccount(t *testing.T) {
	state := state.NewState()
	pool := NewPool(state, 100)

	// Create test account
	account := &types.Account{
		Balance: 1000,
		Nonce:   0,
	}
	address := []byte("test_address")

	// Set account
	pool.SetAccount(address, account)

	// Get account
	retrieved, err := pool.GetAccount(address)
	assert.NoError(t, err)
	assert.Equal(t, account.Balance, retrieved.Balance)
	assert.Equal(t, account.Nonce, retrieved.Nonce)

	// Get non-existent account
	_, err = pool.GetAccount([]byte("non_existent"))
	assert.Error(t, err)
}

func TestPool_ValidateTransaction(t *testing.T) {
	state := state.NewState()
	pool := NewPool(state, 100)

	// Create sender account with balance
	sender := &types.Account{
		Balance: 1000,
		Nonce:   0,
	}
	senderAddr := []byte("sender")
	pool.SetAccount(senderAddr, sender)

	// Create recipient account
	recipient := &types.Account{
		Balance: 0,
		Nonce:   0,
	}
	recipientAddr := []byte("recipient")
	pool.SetAccount(recipientAddr, recipient)

	// Test valid transaction
	tx := &types.Transaction{
		From:  senderAddr,
		To:    recipientAddr,
		Value: 500,
		Nonce: 0,
	}
	err := pool.ValidateTransaction(tx)
	assert.NoError(t, err)

	// Verify balances were not updated (ValidateTransaction only validates)
	senderAcc, err := pool.GetAccount(senderAddr)
	assert.NoError(t, err)
	recipientAcc, err := pool.GetAccount(recipientAddr)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1000), senderAcc.Balance)  // Balance should remain unchanged
	assert.Equal(t, uint64(0), recipientAcc.Balance)  // Balance should remain unchanged
	assert.Equal(t, uint64(0), senderAcc.Nonce)      // Nonce should remain unchanged

	// Test insufficient balance
	tx.Value = 2000  // More than sender's balance
	err = pool.ValidateTransaction(tx)
	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientFunds, err)

	// Test invalid nonce
	tx.Value = 500  // Reset to valid amount
	tx.Nonce = 1    // Invalid nonce
	err = pool.ValidateTransaction(tx)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidNonce, err)
}

func TestPool_ProcessTransaction(t *testing.T) {
	state := state.NewState()
	pool := NewPool(state, 100)

	// Create sender account with balance
	sender := &types.Account{
		Balance: 1000,
		Nonce:   0,
	}
	senderAddr := []byte("sender")
	pool.SetAccount(senderAddr, sender)

	// Create recipient account
	recipient := &types.Account{
		Balance: 0,
		Nonce:   0,
	}
	recipientAddr := []byte("recipient")
	pool.SetAccount(recipientAddr, recipient)

	// Test valid transaction
	tx := &types.Transaction{
		From:  senderAddr,
		To:    recipientAddr,
		Value: 500,
		Nonce: 0,
	}

	// Process transaction
	result, err := pool.ProcessTransaction(tx)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify account states were updated
	updatedSender, err := pool.GetAccount(senderAddr)
	assert.NoError(t, err)
	updatedRecipient, err := pool.GetAccount(recipientAddr)
	assert.NoError(t, err)

	// Check balances and nonce were updated correctly
	assert.Equal(t, uint64(500), updatedSender.Balance)    // 1000 - 500
	assert.Equal(t, uint64(500), updatedRecipient.Balance) // 0 + 500
	assert.Equal(t, uint64(1), updatedSender.Nonce)        // Incremented

	// Test transaction to non-existent recipient
	newTx := &types.Transaction{
		From:  senderAddr,
		To:    []byte("non_existent"),
		Value: 100,
		Nonce: 1,  // Use correct nonce
	}
	result, err = pool.ProcessTransaction(newTx)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify new recipient was created and balance transferred
	newRecipient, err := pool.GetAccount([]byte("non_existent"))
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), newRecipient.Balance)

	// Verify sender's state after second transaction
	finalSender, err := pool.GetAccount(senderAddr)
	assert.NoError(t, err)
	assert.Equal(t, uint64(400), finalSender.Balance) // 500 - 100
	assert.Equal(t, uint64(2), finalSender.Nonce)

	// Test insufficient balance
	badTx := &types.Transaction{
		From:  senderAddr,
		To:    recipientAddr,
		Value: 1000,  // More than current balance
		Nonce: 2,     // Correct nonce
	}
	result, err = pool.ProcessTransaction(badTx)
	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientFunds, err)
	assert.Nil(t, result)

	// Test invalid nonce
	badNonceTx := &types.Transaction{
		From:  senderAddr,
		To:    recipientAddr,
		Value: 100,
		Nonce: 0,  // Invalid nonce (should be 2)
	}
	result, err = pool.ProcessTransaction(badNonceTx)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidNonce, err)
	assert.Nil(t, result)
}

func TestPool_AddTransaction(t *testing.T) {
	s := state.NewState()
	pool := NewPool(s, 100)

	// Create sender account
	sender := &types.Account{
		Address: []byte("sender"),
		Balance: 1000,
		Nonce:   0,
	}
	s.SetAccount(string(sender.Address), sender)

	// Create valid transaction
	tx := &types.Transaction{
		From:     sender.Address,
		To:       []byte("receiver"),
		Value:    500,
		Nonce:    0,
		GasLimit: 0,
		GasPrice: 0,
	}

	// Add transaction
	err := pool.AddTransaction(tx)
	require.NoError(t, err)

	// Verify transaction was added
	addedTx, exists := pool.GetTransaction(tx.Hash())
	assert.True(t, exists)
	assert.Equal(t, tx, addedTx)
}

func TestPool_DuplicateTransaction(t *testing.T) {
	s := state.NewState()
	pool := NewPool(s, 100)

	// Create sender account
	sender := &types.Account{
		Address: []byte("sender"),
		Balance: 1000,
		Nonce:   0,
	}
	s.SetAccount(string(sender.Address), sender)

	// Create transaction
	tx := &types.Transaction{
		From:     sender.Address,
		To:       []byte("receiver"),
		Value:    500,
		Nonce:    0,
		GasLimit: 0,
		GasPrice: 0,
	}

	// Add transaction first time
	err := pool.AddTransaction(tx)
	require.NoError(t, err)

	// Try to add same transaction again
	err = pool.AddTransaction(tx)
	assert.Equal(t, ErrDuplicateTx, err)
}

func TestPool_InvalidTransaction(t *testing.T) {
	s := state.NewState()
	pool := NewPool(s, 100)

	// Create transaction without sender account
	tx := &types.Transaction{
		From:     []byte("nonexistent"),
		To:       []byte("receiver"),
		Value:    500,
		Nonce:    0,
		GasLimit: 0,
		GasPrice: 0,
	}

	// Try to add transaction
	err := pool.AddTransaction(tx)
	assert.Error(t, err)
}

func TestPool_InsufficientFunds(t *testing.T) {
	s := state.NewState()
	pool := NewPool(s, 100)

	// Create sender account with insufficient funds
	sender := &types.Account{
		Address: []byte("sender"),
		Balance: 100,
		Nonce:   0,
	}
	s.SetAccount(string(sender.Address), sender)

	// Create transaction with value higher than balance
	tx := &types.Transaction{
		From:     sender.Address,
		To:       []byte("receiver"),
		Value:    500,
		Nonce:    0,
		GasLimit: 0,
		GasPrice: 0,
	}

	// Try to add transaction
	err := pool.AddTransaction(tx)
	assert.Equal(t, ErrInsufficientFunds, err)
}

func TestPool_InvalidNonce(t *testing.T) {
	s := state.NewState()
	pool := NewPool(s, 100)

	// Create sender account
	sender := &types.Account{
		Address: []byte("sender"),
		Balance: 1000,
		Nonce:   1,
	}
	s.SetAccount(string(sender.Address), sender)

	// Create transaction with invalid nonce
	tx := &types.Transaction{
		From:     sender.Address,
		To:       []byte("receiver"),
		Value:    500,
		Nonce:    0,
		GasLimit: 0,
		GasPrice: 0,
	}

	// Try to add transaction
	err := pool.AddTransaction(tx)
	assert.Equal(t, ErrInvalidNonce, err)
}

func TestPool_PoolFull(t *testing.T) {
	s := state.NewState()
	pool := NewPool(s, 1)

	// Create sender account
	sender := &types.Account{
		Address: []byte("sender"),
		Balance: 1000,
		Nonce:   0,
	}
	s.SetAccount(string(sender.Address), sender)

	// Create first transaction
	tx1 := &types.Transaction{
		From:     sender.Address,
		To:       []byte("receiver1"),
		Value:    100,
		Nonce:    0,
		GasLimit: 0,
		GasPrice: 0,
	}

	// Add first transaction
	err := pool.AddTransaction(tx1)
	require.NoError(t, err)

	// Create second transaction
	tx2 := &types.Transaction{
		From:     sender.Address,
		To:       []byte("receiver2"),
		Value:    100,
		Nonce:    1,
		GasLimit: 0,
		GasPrice: 0,
	}

	// Try to add second transaction
	err = pool.AddTransaction(tx2)
	assert.Equal(t, ErrPoolFull, err)
} 