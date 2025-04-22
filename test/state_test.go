package test

import (
	"testing"

	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestStateManager(t *testing.T) {
	// Create state manager
	sm := state.NewStateManager()
	assert.NotNil(t, sm)

	// Create test accounts
	account1 := &types.Account{
		Address: []byte("account1"),
		Balance: 100,
		Nonce:   0,
	}

	account2 := &types.Account{
		Address: []byte("account2"),
		Balance: 200,
		Nonce:   0,
	}

	// Create accounts
	err := sm.CreateAccount(account1)
	assert.NoError(t, err)

	err = sm.CreateAccount(account2)
	assert.NoError(t, err)

	// Get accounts
	acc1, err := sm.GetAccount(account1.Address)
	assert.NoError(t, err)
	assert.Equal(t, account1.Balance, acc1.Balance)
	assert.Equal(t, account1.Nonce, acc1.Nonce)

	acc2, err := sm.GetAccount(account2.Address)
	assert.NoError(t, err)
	assert.Equal(t, account2.Balance, acc2.Balance)
	assert.Equal(t, account2.Nonce, acc2.Nonce)

	// Update balances
	err = sm.UpdateBalance(account1.Address, 150)
	assert.NoError(t, err)

	err = sm.UpdateBalance(account2.Address, 250)
	assert.NoError(t, err)

	// Verify updated balances
	acc1, err = sm.GetAccount(account1.Address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(150), acc1.Balance)

	acc2, err = sm.GetAccount(account2.Address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(250), acc2.Balance)

	// Update nonces
	err = sm.UpdateNonce(account1.Address, 1)
	assert.NoError(t, err)

	err = sm.UpdateNonce(account2.Address, 1)
	assert.NoError(t, err)

	// Verify updated nonces
	acc1, err = sm.GetAccount(account1.Address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), acc1.Nonce)

	acc2, err = sm.GetAccount(account2.Address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), acc2.Nonce)

	// Test storage
	err = sm.SetStorage(account1.Address, []byte("key1"), []byte("value1"))
	assert.NoError(t, err)

	value, err := sm.GetStorage(account1.Address, []byte("key1"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), value)

	// Test code
	err = sm.SetCode(account1.Address, []byte("code1"))
	assert.NoError(t, err)

	code, err := sm.GetCode(account1.Address)
	assert.NoError(t, err)
	assert.Equal(t, []byte("code1"), code)

	// Test state proof
	proof, err := sm.GenerateProof(account1.Address)
	assert.NoError(t, err)
	assert.NotNil(t, proof)

	valid, err := sm.VerifyProof(proof)
	assert.NoError(t, err)
	assert.True(t, valid)

	// Test batch operations
	updates := []types.BalanceUpdate{
		{
			Address: []byte("account3"),
			Balance: 300,
		},
		{
			Address: []byte("account4"),
			Balance: 400,
		},
	}

	err = sm.BatchCreateAccounts(updates)
	assert.NoError(t, err)

	acc3, err := sm.GetAccount([]byte("account3"))
	assert.NoError(t, err)
	assert.Equal(t, uint64(300), acc3.Balance)

	acc4, err := sm.GetAccount([]byte("account4"))
	assert.NoError(t, err)
	assert.Equal(t, uint64(400), acc4.Balance)
}

func TestStateTransactions(t *testing.T) {
	s := state.NewState()

	// Create initial accounts
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

	err := s.SetAccount(sender.Address, sender)
	assert.NoError(t, err)

	err = s.SetAccount(recipient.Address, recipient)
	assert.NoError(t, err)

	// Create and validate transaction
	tx := &types.Transaction{
		From:  sender.Address,
		To:    recipient.Address,
		Value: 500,
		Nonce: 0,
	}

	// Update balances
	sender.Balance -= tx.Value
	recipient.Balance += tx.Value
	sender.Nonce++

	err = s.UpdateAccount(sender.Address, sender)
	assert.NoError(t, err)

	err = s.UpdateAccount(recipient.Address, recipient)
	assert.NoError(t, err)

	// Verify state changes
	updatedSender, exists := s.GetAccount(sender.Address)
	assert.True(t, exists)
	assert.Equal(t, uint64(500), updatedSender.Balance)
	assert.Equal(t, uint64(1), updatedSender.Nonce)

	updatedRecipient, exists := s.GetAccount(recipient.Address)
	assert.True(t, exists)
	assert.Equal(t, uint64(500), updatedRecipient.Balance)
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