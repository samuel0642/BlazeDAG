package test

import (
	"testing"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestTypes(t *testing.T) {
	// Test Block
	block := &types.Block{
		Header: types.BlockHeader{
			Version:    1,
			Round:      1,
			Wave:       1,
			Height:     1,
			ParentHash: []byte("parent"),
		},
		References: []types.Reference{},
	}
	assert.NotNil(t, block)

	// Test Transaction
	tx := &types.Transaction{
		From:     []byte("sender"),
		To:       []byte("recipient"),
		Value:    100,
		GasLimit: 21000,
		GasPrice: 1,
		Nonce:    0,
	}
	assert.NotNil(t, tx)

	// Test Account
	account := &types.Account{
		Balance: 1000,
		Nonce:   0,
	}
	assert.NotNil(t, account)
}

func TestBlock(t *testing.T) {
	block := types.Block{
		Header: types.BlockHeader{
			Version:        1,
			Round:          1,
			Wave:           1,
			Height:         1,
			ParentHash:     []byte("parent"),
			StateRoot:      []byte("state"),
			TransactionRoot: []byte("transactions"),
			ReceiptRoot:    []byte("receipts"),
			Validator:      []byte("validator"),
		},
		Body: types.BlockBody{
			Transactions: []types.Transaction{},
			Receipts:     []types.Receipt{},
			Events:       []types.Event{},
		},
		References: []types.Reference{
			{
				BlockHash: []byte("ref"),
				Round:     1,
				Wave:      1,
				Type:      types.ReferenceTypeStandard,
			},
		},
		Signature: []byte("signature"),
		Timestamp: time.Now(),
	}

	assert.NotNil(t, block)
	assert.Equal(t, uint32(1), block.Header.Version)
	assert.Equal(t, uint64(1), block.Header.Round)
	assert.Equal(t, uint64(1), block.Header.Wave)
	assert.Equal(t, uint64(1), block.Header.Height)
	assert.Equal(t, []byte("parent"), block.Header.ParentHash)
	assert.Equal(t, []byte("state"), block.Header.StateRoot)
	assert.Equal(t, []byte("transactions"), block.Header.TransactionRoot)
	assert.Equal(t, []byte("receipts"), block.Header.ReceiptRoot)
	assert.Equal(t, []byte("validator"), block.Header.Validator)
	assert.Equal(t, 1, len(block.References))
	assert.Equal(t, []byte("ref"), block.References[0].BlockHash)
	assert.Equal(t, uint64(1), block.References[0].Round)
	assert.Equal(t, uint64(1), block.References[0].Wave)
	assert.Equal(t, types.ReferenceTypeStandard, block.References[0].Type)
}

func TestTransaction(t *testing.T) {
	tx := types.Transaction{
		Nonce:    1,
		From:     []byte("from"),
		To:       []byte("to"),
		Value:    []byte("value"),
		GasLimit: 21000,
		GasPrice: []byte("gasprice"),
		Data:     []byte("data"),
		Signature: []byte("signature"),
	}

	assert.NotNil(t, tx)
	assert.Equal(t, uint64(1), tx.Nonce)
	assert.Equal(t, []byte("from"), tx.From)
	assert.Equal(t, []byte("to"), tx.To)
	assert.Equal(t, []byte("value"), tx.Value)
	assert.Equal(t, uint64(21000), tx.GasLimit)
	assert.Equal(t, []byte("gasprice"), tx.GasPrice)
	assert.Equal(t, []byte("data"), tx.Data)
	assert.Equal(t, []byte("signature"), tx.Signature)
}

func TestState(t *testing.T) {
	state := types.State{
		Accounts: make(map[string]types.Account),
		Storage:  make(map[string]map[string][]byte),
		Code:     make(map[string][]byte),
		Nonce:    make(map[string]uint64),
		Balance:  make(map[string][]byte),
	}

	state.Accounts["account1"] = types.Account{
		Nonce:      1,
		Balance:    []byte("balance"),
		CodeHash:   []byte("codehash"),
		StorageRoot: []byte("storageroot"),
	}

	assert.NotNil(t, state)
	assert.Equal(t, 1, len(state.Accounts))
	assert.Equal(t, uint64(1), state.Accounts["account1"].Nonce)
	assert.Equal(t, []byte("balance"), state.Accounts["account1"].Balance)
	assert.Equal(t, []byte("codehash"), state.Accounts["account1"].CodeHash)
	assert.Equal(t, []byte("storageroot"), state.Accounts["account1"].StorageRoot)
}

func TestProposal(t *testing.T) {
	proposal := types.Proposal{
		ID:        []byte("id"),
		Round:     1,
		Wave:      1,
		Proposer:  []byte("proposer"),
		Timestamp: time.Now(),
		Status:    types.ProposalStatusPending,
	}

	assert.NotNil(t, proposal)
	assert.Equal(t, []byte("id"), proposal.ID)
	assert.Equal(t, uint64(1), proposal.Round)
	assert.Equal(t, uint64(1), proposal.Wave)
	assert.Equal(t, []byte("proposer"), proposal.Proposer)
	assert.Equal(t, types.ProposalStatusPending, proposal.Status)
}

func TestNetworkMessage(t *testing.T) {
	msg := types.NetworkMessage{
		Type:      types.MessageTypeBlock,
		Payload:   []byte("payload"),
		Signature: []byte("signature"),
		Timestamp: time.Now(),
		Sender:    []byte("sender"),
		Recipient: []byte("recipient"),
		Priority:  1,
		TTL:       1000,
	}

	assert.NotNil(t, msg)
	assert.Equal(t, types.MessageTypeBlock, msg.Type)
	assert.Equal(t, []byte("payload"), msg.Payload)
	assert.Equal(t, []byte("signature"), msg.Signature)
	assert.Equal(t, []byte("sender"), msg.Sender)
	assert.Equal(t, []byte("recipient"), msg.Recipient)
	assert.Equal(t, uint8(1), msg.Priority)
	assert.Equal(t, uint64(1000), msg.TTL)
}

func TestBlockHash(t *testing.T) {
	block := &types.Block{
		Header: types.BlockHeader{
			Version:    0,
			Round:      1,
			Wave:       0,
			Height:     1,
			ParentHash: []byte("parent"),
		},
		References: []types.Reference{},
	}

	hash := block.Hash()
	assert.NotEmpty(t, hash)
} 