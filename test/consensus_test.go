package test

import (
	"context"
	"testing"
	"time"

	"BlazeDAG/internal/types"
	"BlazeDAG/pkg/consensus"
	"github.com/stretchr/testify/assert"
)

func TestConsensusEngine(t *testing.T) {
	// Create config
	config := &consensus.Config{
		RoundDuration:  1 * time.Second,
		WaveTimeout:    5 * time.Second,
		MaxValidators:  100,
		MinValidators:  4,
		QuorumSize:     3,
		LeaderRotation: true,
	}

	// Create engine
	engine := consensus.NewEngine(config)

	// Start engine
	err := engine.Start()
	assert.NoError(t, err)

	// Create proposal
	proposal := &types.Proposal{
		ID:        []byte("proposal1"),
		Round:     1,
		Wave:      1,
		Proposer:  []byte("proposer1"),
		Timestamp: time.Now(),
		Status:    types.ProposalStatusPending,
		Block: types.Block{
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
		},
	}

	// Submit proposal
	err = engine.SubmitProposal(proposal)
	assert.NoError(t, err)

	// Create votes
	votes := make([]*types.Vote, 3)
	for i := 0; i < 3; i++ {
		votes[i] = &types.Vote{
			ProposalID: []byte("proposal1"),
			Validator:  []byte("validator"),
			Round:      1,
			Wave:       1,
			Timestamp:  time.Now(),
			Signature:  []byte("signature"),
			Type:       types.VoteTypeApprove,
		}
	}

	// Submit votes
	for _, vote := range votes {
		err = engine.SubmitVote(vote)
		assert.NoError(t, err)
	}

	// Wait for block
	select {
	case block := <-engine.GetBlockChannel():
		assert.NotNil(t, block)
		assert.Equal(t, uint32(1), block.Header.Version)
		assert.Equal(t, uint64(1), block.Header.Round)
		assert.Equal(t, uint64(1), block.Header.Wave)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for block")
	}

	// Stop engine
	err = engine.Stop()
	assert.NoError(t, err)
}

func TestConsensusEngineValidation(t *testing.T) {
	// Create config
	config := &consensus.Config{
		RoundDuration:  1 * time.Second,
		WaveTimeout:    5 * time.Second,
		MaxValidators:  100,
		MinValidators:  4,
		QuorumSize:     3,
		LeaderRotation: true,
	}

	// Create engine
	engine := consensus.NewEngine(config)

	// Start engine
	err := engine.Start()
	assert.NoError(t, err)

	// Create invalid proposal
	invalidProposal := &types.Proposal{
		ID:        []byte("invalid"),
		Round:     0, // Invalid round
		Wave:      0, // Invalid wave
		Proposer:  []byte("proposer"),
		Timestamp: time.Now(),
		Status:    types.ProposalStatusPending,
	}

	// Submit invalid proposal
	err = engine.SubmitProposal(invalidProposal)
	assert.NoError(t, err)

	// Create invalid vote
	invalidVote := &types.Vote{
		ProposalID: []byte("invalid"),
		Validator:  []byte("validator"),
		Round:      0, // Invalid round
		Wave:       0, // Invalid wave
		Timestamp:  time.Now(),
		Signature:  []byte("signature"),
		Type:       types.VoteTypeApprove,
	}

	// Submit invalid vote
	err = engine.SubmitVote(invalidVote)
	assert.NoError(t, err)

	// Wait to ensure no block is produced
	select {
	case <-engine.GetBlockChannel():
		t.Fatal("Received block for invalid proposal/vote")
	case <-time.After(2 * time.Second):
		// Expected timeout
	}

	// Stop engine
	err = engine.Stop()
	assert.NoError(t, err)
}

func TestConsensusEngineQuorum(t *testing.T) {
	// Create config
	config := &consensus.Config{
		RoundDuration:  1 * time.Second,
		WaveTimeout:    5 * time.Second,
		MaxValidators:  100,
		MinValidators:  4,
		QuorumSize:     3,
		LeaderRotation: true,
	}

	// Create engine
	engine := consensus.NewEngine(config)

	// Start engine
	err := engine.Start()
	assert.NoError(t, err)

	// Create proposal
	proposal := &types.Proposal{
		ID:        []byte("proposal1"),
		Round:     1,
		Wave:      1,
		Proposer:  []byte("proposer1"),
		Timestamp: time.Now(),
		Status:    types.ProposalStatusPending,
		Block: types.Block{
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
		},
	}

	// Submit proposal
	err = engine.SubmitProposal(proposal)
	assert.NoError(t, err)

	// Create insufficient votes
	votes := make([]*types.Vote, 2) // Only 2 votes, need 3 for quorum
	for i := 0; i < 2; i++ {
		votes[i] = &types.Vote{
			ProposalID: []byte("proposal1"),
			Validator:  []byte("validator"),
			Round:      1,
			Wave:       1,
			Timestamp:  time.Now(),
			Signature:  []byte("signature"),
			Type:       types.VoteTypeApprove,
		}
	}

	// Submit insufficient votes
	for _, vote := range votes {
		err = engine.SubmitVote(vote)
		assert.NoError(t, err)
	}

	// Wait to ensure no block is produced
	select {
	case <-engine.GetBlockChannel():
		t.Fatal("Received block without quorum")
	case <-time.After(2 * time.Second):
		// Expected timeout
	}

	// Stop engine
	err = engine.Stop()
	assert.NoError(t, err)
} 