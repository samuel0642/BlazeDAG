package state

import (
	"testing"

	"github.com/samuel0642/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateManager(t *testing.T) {
	sm := NewStateManager()

	// Test account creation
	addr := []byte("test_account")
	err := sm.CreateAccount(addr, 1000)
	require.NoError(t, err)

	// Test get account
	account, err := sm.GetAccount(string(addr))
	require.NoError(t, err)
	assert.Equal(t, uint64(1000), account.Balance)
	assert.Equal(t, uint64(0), account.Nonce)

	// Test update balance
	err = sm.UpdateBalance(addr, 2000)
	require.NoError(t, err)
	account, err = sm.GetAccount(string(addr))
	require.NoError(t, err)
	assert.Equal(t, uint64(2000), account.Balance)

	// Test update nonce
	err = sm.UpdateNonce(addr, 1)
	require.NoError(t, err)
	account, err = sm.GetAccount(string(addr))
	require.NoError(t, err)
	assert.Equal(t, uint64(1), account.Nonce)

	// Test storage
	key := []byte("test_key")
	value := []byte("test_value")
	err = sm.SetStorage(addr, key, value)
	require.NoError(t, err)

	storedValue, err := sm.GetStorage(addr, key)
	require.NoError(t, err)
	assert.Equal(t, value, storedValue)
}

func TestStateManager_ExecuteTransaction(t *testing.T) {
	sm := NewStateManager()

	// Create sender account
	senderAddr := []byte("sender")
	err := sm.CreateAccount(senderAddr, 1000)
	require.NoError(t, err)

	// Create receiver account
	receiverAddr := []byte("receiver")
	err = sm.CreateAccount(receiverAddr, 0)
	require.NoError(t, err)

	// Create and execute transaction
	tx := &types.Transaction{
		From:     senderAddr,
		To:       receiverAddr,
		Value:    500,
		Nonce:    0,
		GasLimit: 0,
		GasPrice: 0,
	}

	err = sm.state.ExecuteTransaction(tx)
	require.NoError(t, err)

	// Verify balances
	sender, err := sm.GetAccount(string(senderAddr))
	require.NoError(t, err)
	assert.Equal(t, uint64(500), sender.Balance)

	receiver, err := sm.GetAccount(string(receiverAddr))
	require.NoError(t, err)
	assert.Equal(t, uint64(500), receiver.Balance)
}

func TestStateManager_NonexistentAccount(t *testing.T) {
	sm := NewStateManager()

	// Try to get nonexistent account
	_, err := sm.GetAccount(string([]byte("nonexistent")))
	assert.Error(t, err)
	assert.Equal(t, ErrAccountNotFound, err)
}

func TestStateManager_Concurrency(t *testing.T) {
	sm := NewStateManager()

	// Create test accounts
	addr1 := []byte("account1")
	addr2 := []byte("account2")
	err := sm.CreateAccount(addr1, 1000)
	require.NoError(t, err)
	err = sm.CreateAccount(addr2, 1000)
	require.NoError(t, err)

	// Concurrently update accounts
	done := make(chan bool, 2)
	go func() {
		err := sm.UpdateBalance(addr1, 2000)
		assert.NoError(t, err)
		done <- true
	}()

	go func() {
		err := sm.UpdateBalance(addr2, 2000)
		assert.NoError(t, err)
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final balances
	account1, err := sm.GetAccount(string(addr1))
	require.NoError(t, err)
	assert.Equal(t, uint64(2000), account1.Balance)

	account2, err := sm.GetAccount(string(addr2))
	require.NoError(t, err)
	assert.Equal(t, uint64(2000), account2.Balance)
} 