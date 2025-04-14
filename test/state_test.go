package test

import (
	"testing"

	"BlazeDAG/internal/state"
	"BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestStateManagement(t *testing.T) {
	// Create state manager
	sm := state.NewStateManager()

	// Test account creation
	account := &types.Account{
		Address: []byte("test-address"),
		Balance: 1000,
		Nonce:   0,
	}

	err := sm.CreateAccount(account)
	assert.NoError(t, err)

	// Test account retrieval
	retrieved, err := sm.GetAccount(account.Address)
	assert.NoError(t, err)
	assert.Equal(t, account, retrieved)

	// Test balance update
	err = sm.UpdateBalance(account.Address, 2000)
	assert.NoError(t, err)

	updated, err := sm.GetAccount(account.Address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2000), updated.Balance)

	// Test nonce update
	err = sm.UpdateNonce(account.Address, 1)
	assert.NoError(t, err)

	updated, err = sm.GetAccount(account.Address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), updated.Nonce)

	// Test storage
	key := []byte("test-key")
	value := []byte("test-value")

	err = sm.SetStorage(account.Address, key, value)
	assert.NoError(t, err)

	retrievedValue, err := sm.GetStorage(account.Address, key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrievedValue)

	// Test code
	code := []byte("test-code")
	err = sm.SetCode(account.Address, code)
	assert.NoError(t, err)

	retrievedCode, err := sm.GetCode(account.Address)
	assert.NoError(t, err)
	assert.Equal(t, code, retrievedCode)
}

func TestStateTransitions(t *testing.T) {
	sm := state.NewStateManager()

	// Create initial state
	account := &types.Account{
		Address: []byte("test-address"),
		Balance: 1000,
		Nonce:   0,
	}

	err := sm.CreateAccount(account)
	assert.NoError(t, err)

	// Create transaction
	tx := &types.Transaction{
		From:  account.Address,
		To:    []byte("recipient"),
		Value: 500,
		Nonce: 0,
	}

	// Execute transaction
	err = sm.ExecuteTransaction(tx)
	assert.NoError(t, err)

	// Verify state changes
	sender, err := sm.GetAccount(account.Address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(500), sender.Balance)
	assert.Equal(t, uint64(1), sender.Nonce)

	recipient, err := sm.GetAccount(tx.To)
	assert.NoError(t, err)
	assert.Equal(t, uint64(500), recipient.Balance)
}

func TestStateSnapshots(t *testing.T) {
	sm := state.NewStateManager()

	// Create initial state
	account := &types.Account{
		Address: []byte("test-address"),
		Balance: 1000,
		Nonce:   0,
	}

	err := sm.CreateAccount(account)
	assert.NoError(t, err)

	// Create snapshot
	snapshotID, err := sm.CreateSnapshot()
	assert.NoError(t, err)

	// Modify state
	err = sm.UpdateBalance(account.Address, 2000)
	assert.NoError(t, err)

	// Verify modification
	updated, err := sm.GetAccount(account.Address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2000), updated.Balance)

	// Restore snapshot
	err = sm.RestoreSnapshot(snapshotID)
	assert.NoError(t, err)

	// Verify restoration
	restored, err := sm.GetAccount(account.Address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1000), restored.Balance)
}

func TestStateProofs(t *testing.T) {
	sm := state.NewStateManager()

	// Create account
	account := &types.Account{
		Address: []byte("test-address"),
		Balance: 1000,
		Nonce:   0,
	}

	err := sm.CreateAccount(account)
	assert.NoError(t, err)

	// Generate proof
	proof, err := sm.GenerateProof(account.Address)
	assert.NoError(t, err)
	assert.NotNil(t, proof)

	// Verify proof
	valid, err := sm.VerifyProof(proof)
	assert.NoError(t, err)
	assert.True(t, valid)

	// Test invalid proof
	invalidProof := &types.StateProof{
		Address: account.Address,
		Value:   []byte("invalid"),
		Proof:   []byte("invalid"),
	}

	valid, err = sm.VerifyProof(invalidProof)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestStateBatchOperations(t *testing.T) {
	sm := state.NewStateManager()

	// Create multiple accounts
	accounts := make([]*types.Account, 10)
	for i := 0; i < 10; i++ {
		accounts[i] = &types.Account{
			Address: []byte("test-address-" + string(rune(i))),
			Balance: uint64(i * 100),
			Nonce:   0,
		}
	}

	// Batch create accounts
	err := sm.BatchCreateAccounts(accounts)
	assert.NoError(t, err)

	// Verify accounts
	for _, account := range accounts {
		retrieved, err := sm.GetAccount(account.Address)
		assert.NoError(t, err)
		assert.Equal(t, account, retrieved)
	}

	// Batch update balances
	updates := make([]*types.BalanceUpdate, 10)
	for i := 0; i < 10; i++ {
		updates[i] = &types.BalanceUpdate{
			Address: accounts[i].Address,
			Balance: uint64(i * 200),
		}
	}

	err = sm.BatchUpdateBalances(updates)
	assert.NoError(t, err)

	// Verify updates
	for i, account := range accounts {
		retrieved, err := sm.GetAccount(account.Address)
		assert.NoError(t, err)
		assert.Equal(t, uint64(i*200), retrieved.Balance)
	}
} 