package consensus

import (
	"testing"
	"time"

	"github.com/samuel0642/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	config := &Config{
		RoundDuration:  time.Second,
		WaveTimeout:    time.Second,
		MaxValidators:  10,
		MinValidators:  4,
		QuorumSize:     3,
		LeaderRotation: true,
	}

	engine := NewEngine(config)
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.config)
	assert.NotNil(t, engine.waveState)
	assert.NotNil(t, engine.proposals)
	assert.NotNil(t, engine.votes)
	assert.NotNil(t, engine.proposalCh)
	assert.NotNil(t, engine.voteCh)
	assert.NotNil(t, engine.blockCh)
}

func TestEngine_StartStop(t *testing.T) {
	config := &Config{
		RoundDuration:  time.Second,
		WaveTimeout:    time.Second,
		MaxValidators:  10,
		MinValidators:  4,
		QuorumSize:     3,
		LeaderRotation: true,
	}

	engine := NewEngine(config)
	err := engine.Start()
	assert.NoError(t, err)

	// Wait a bit to ensure the engine is running
	time.Sleep(100 * time.Millisecond)

	err = engine.Stop()
	assert.NoError(t, err)
}

func TestEngine_HandleProposal(t *testing.T) {
	config := &Config{
		RoundDuration:  time.Second,
		WaveTimeout:    time.Second,
		MaxValidators:  10,
		MinValidators:  4,
		QuorumSize:     3,
		LeaderRotation: true,
	}

	engine := NewEngine(config)
	engine.Start()
	defer engine.Stop()

	// Create a valid proposal
	proposal := &types.Proposal{
		ID:        []byte("test-proposal"),
		Wave:      1,
		Round:     1,
		Block:     &types.Block{},
		Proposer:  []byte("leader"),
		Timestamp: time.Now(),
		Status:    types.ConsensusStatusPending,
	}

	err := engine.SubmitProposal(proposal)
	assert.NoError(t, err)

	// Wait for proposal to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify proposal was stored
	engine.mu.RLock()
	storedProposal := engine.proposals[string(proposal.ID)]
	engine.mu.RUnlock()
	assert.NotNil(t, storedProposal)
	assert.Equal(t, proposal.ID, storedProposal.ID)
}

func TestEngine_HandleVote(t *testing.T) {
	config := &Config{
		RoundDuration:  time.Second,
		WaveTimeout:    time.Second,
		MaxValidators:  10,
		MinValidators:  4,
		QuorumSize:     3,
		LeaderRotation: true,
	}

	engine := NewEngine(config)
	engine.Start()
	defer engine.Stop()

	// Create a proposal first
	proposal := &types.Proposal{
		ID:        []byte("test-proposal"),
		Wave:      1,
		Round:     1,
		Block:     &types.Block{},
		Proposer:  []byte("leader"),
		Timestamp: time.Now(),
		Status:    types.ConsensusStatusPending,
	}
	err := engine.SubmitProposal(proposal)
	require.NoError(t, err)

	// Create and submit votes
	votes := []*types.Vote{
		{
			ProposalID: []byte("test-proposal"),
			Wave:      1,
			Round:     1,
			Validator: []byte("validator1"),
			Type:      types.VoteTypeApprove,
			Timestamp: time.Now(),
		},
		{
			ProposalID: []byte("test-proposal"),
			Wave:      1,
			Round:     1,
			Validator: []byte("validator2"),
			Type:      types.VoteTypeApprove,
			Timestamp: time.Now(),
		},
		{
			ProposalID: []byte("test-proposal"),
			Wave:      1,
			Round:     1,
			Validator: []byte("validator3"),
			Type:      types.VoteTypeApprove,
			Timestamp: time.Now(),
		},
	}

	for _, vote := range votes {
		err := engine.SubmitVote(vote)
		assert.NoError(t, err)
	}

	// Wait for votes to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify votes were stored and quorum was reached
	engine.mu.RLock()
	storedVotes := engine.votes[string(proposal.ID)]
	engine.mu.RUnlock()
	assert.Len(t, storedVotes, 3)
}

func TestEngine_HandleRound(t *testing.T) {
	config := &Config{
		RoundDuration:  time.Second,
		WaveTimeout:    time.Second,
		MaxValidators:  10,
		MinValidators:  4,
		QuorumSize:     3,
		LeaderRotation: true,
	}

	engine := NewEngine(config)
	engine.Start()
	defer engine.Stop()

	// Wait for a round to complete
	time.Sleep(1100 * time.Millisecond)

	// Verify wave state was updated
	engine.mu.RLock()
	assert.Equal(t, uint64(1), engine.waveState.CurrentWave)
	assert.Equal(t, types.WaveStatusActive, engine.waveState.WaveStatus)
	engine.mu.RUnlock()
}

func TestEngine_Concurrency(t *testing.T) {
	config := &Config{
		RoundDuration:  time.Second,
		WaveTimeout:    time.Second,
		MaxValidators:  10,
		MinValidators:  4,
		QuorumSize:     3,
		LeaderRotation: true,
	}

	engine := NewEngine(config)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// Wait for engine to start
	time.Sleep(100 * time.Millisecond)

	// Create a proposal
	proposal := &types.Proposal{
		ID:        []byte("test-proposal"),
		Wave:      1,
		Round:     1,
		Block:     &types.Block{},
		Proposer:  []byte("leader"),
		Timestamp: time.Now(),
		Status:    types.ConsensusStatusPending,
	}

	// Concurrently submit proposal and votes
	done := make(chan bool, 2)
	go func() {
		err := engine.SubmitProposal(proposal)
		assert.NoError(t, err)
		done <- true
	}()

	// Wait a bit for proposal to be processed
	time.Sleep(50 * time.Millisecond)

	go func() {
		vote := &types.Vote{
			ProposalID: []byte("test-proposal"),
			Wave:      1,
			Round:     1,
			Validator: []byte("validator1"),
			Type:      types.VoteTypeApprove,
			Timestamp: time.Now(),
		}
		err := engine.SubmitVote(vote)
		assert.NoError(t, err)
		done <- true
	}()

	// Wait for both goroutines
	for i := 0; i < 2; i++ {
		select {
		case <-done:
			continue
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for goroutines")
		}
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify final state
	engine.mu.RLock()
	storedProposal := engine.proposals[string(proposal.ID)]
	storedVotes := engine.votes[string(proposal.ID)]
	engine.mu.RUnlock()

	require.NotNil(t, storedProposal)
	require.NotNil(t, storedVotes)
	assert.Len(t, storedVotes, 1)
} 