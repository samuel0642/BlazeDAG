package consensus

import (
	"context"
	"testing"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestWaveState_NewWaveState(t *testing.T) {
	tests := []struct {
		name    string
		config  *types.ConsensusConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &types.ConsensusConfig{
				WaveTimeout:   time.Second,
				RoundDuration: time.Second,
				ValidatorSet:  []string{"validator1", "validator2", "validator3"},
				QuorumSize:    2,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ws, err := NewWaveState(ctx, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ws)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ws)
				assert.Equal(t, types.WaveStatusInitializing, ws.Status())
			}
		})
	}
}

func TestWaveState_StartWave(t *testing.T) {
	ctx := context.Background()
	config := &types.ConsensusConfig{
		ValidatorSet: []string{"validator1", "validator2"},
		QuorumSize:   2,
	}

	state, err := NewWaveState(ctx, config)
	assert.NoError(t, err)
	assert.NotNil(t, state)

	// Start a new wave
	err = state.StartWave()
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), state.CurrentWave())
	assert.NotEmpty(t, state.CurrentLeader())
}

func TestWaveState_AddProposal(t *testing.T) {
	ctx := context.Background()
	config := &types.ConsensusConfig{
		ValidatorSet: []string{"validator1", "validator2"},
		QuorumSize:   2,
	}

	state, err := NewWaveState(ctx, config)
	assert.NoError(t, err)

	// Start a new wave
	err = state.StartWave()
	assert.NoError(t, err)

	// Create a proposal from the leader
	leader := state.CurrentLeader()
	proposal := &types.Proposal{
		ID:        []byte("proposal1"),
		Round:     1,
		Wave:      1,
		Proposer:  leader,
		Timestamp: time.Now(),
	}

	// Add the proposal
	err = state.AddProposal(proposal)
	assert.NoError(t, err)

	// Try to add a proposal from a non-leader
	proposal.Proposer = []byte("validator2")
	err = state.AddProposal(proposal)
	assert.Error(t, err)

	// Try to add a proposal with wrong wave number
	proposal.Wave = 2
	err = state.AddProposal(proposal)
	assert.Error(t, err)
}

func TestWaveState_AddVote(t *testing.T) {
	ctx := context.Background()
	config := &types.ConsensusConfig{
		ValidatorSet: []string{"validator1", "validator2"},
		QuorumSize:   2,
	}

	state, err := NewWaveState(ctx, config)
	assert.NoError(t, err)

	// Start a new wave
	err = state.StartWave()
	assert.NoError(t, err)

	// Create a proposal
	leader := state.CurrentLeader()
	proposal := &types.Proposal{
		ID:        []byte("proposal1"),
		Round:     1,
		Wave:      1,
		Proposer:  leader,
		Timestamp: time.Now(),
	}

	// Add the proposal
	err = state.AddProposal(proposal)
	assert.NoError(t, err)

	// Create a vote
	vote := &types.Vote{
		ProposalID: proposal.ID,
		Validator:  []byte("validator1"),
		Round:      1,
		Wave:       1,
		Timestamp:  time.Now(),
	}

	// Add the vote
	err = state.AddVote(vote)
	assert.NoError(t, err)
}

func TestWaveState_CompleteWave(t *testing.T) {
	ctx := context.Background()
	config := &types.ConsensusConfig{
		ValidatorSet: []string{"validator1", "validator2"},
		QuorumSize:   2,
	}

	state, err := NewWaveState(ctx, config)
	assert.NoError(t, err)

	// Start a new wave
	err = state.StartWave()
	assert.NoError(t, err)

	// Create and add a proposal
	leader := state.CurrentLeader()
	proposal := &types.Proposal{
		ID:        []byte("proposal1"),
		Round:     1,
		Wave:      1,
		Proposer:  leader,
		Timestamp: time.Now(),
	}
	err = state.AddProposal(proposal)
	assert.NoError(t, err)

	// Add enough votes to reach quorum
	for _, validator := range config.ValidatorSet {
		vote := &types.Vote{
			ProposalID: proposal.ID,
			Validator:  []byte(validator),
			Round:      1,
			Wave:       1,
			Timestamp:  time.Now(),
		}
		err = state.AddVote(vote)
		assert.NoError(t, err)
	}

	// Complete the wave
	err = state.CompleteWave()
	assert.NoError(t, err)
}

func TestWaveState_Concurrency(t *testing.T) {
	ctx := context.Background()
	config := &types.ConsensusConfig{
		ValidatorSet: []string{"validator1", "validator2"},
		QuorumSize:   2,
	}

	state, err := NewWaveState(ctx, config)
	assert.NoError(t, err)

	// Start a new wave
	err = state.StartWave()
	assert.NoError(t, err)

	// Create a proposal
	leader := state.CurrentLeader()
	proposal := &types.Proposal{
		ID:        []byte("proposal1"),
		Round:     1,
		Wave:      1,
		Proposer:  leader,
		Timestamp: time.Now(),
	}

	// Test concurrent proposal and vote additions
	done := make(chan bool)
	go func() {
		err := state.AddProposal(proposal)
		assert.NoError(t, err)
		done <- true
	}()

	go func() {
		vote := &types.Vote{
			ProposalID: proposal.ID,
			Validator:  []byte("validator1"),
			Round:      1,
			Wave:       1,
			Timestamp:  time.Now(),
		}
		err := state.AddVote(vote)
		assert.NoError(t, err)
		done <- true
	}()

	// Wait for goroutines to complete
	<-done
	<-done

	// Verify wave completion
	time.Sleep(100 * time.Millisecond)
	cert, exists := state.GetCertificate(proposal.ID)
	assert.True(t, exists)
	assert.NotNil(t, cert)
} 