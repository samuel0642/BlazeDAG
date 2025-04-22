package test

import (
	"context"
	"testing"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestConsensus_NewEngine(t *testing.T) {
	config := consensus.Config{
		TotalValidators: 3,
		FaultTolerance:  1,
		RoundDuration:   time.Second,
		WaveTimeout:     time.Second,
	}

	engine := consensus.NewEngine(config)
	assert.NotNil(t, engine)
}

func TestConsensus_HandleProposal(t *testing.T) {
	config := consensus.Config{
		TotalValidators: 3,
		FaultTolerance:  1,
		RoundDuration:   time.Second,
		WaveTimeout:     time.Second,
	}

	engine := consensus.NewEngine(config)

	// Create a proposal
	proposal := &consensus.Proposal{
		BlockHash:  "proposal1",
		Round:      1,
		Sender:     "validator1",
		Timestamp:  time.Now(),
	}

	// Handle proposal
	err := engine.HandleProposal(proposal)
	assert.NoError(t, err)
}

func TestConsensus_HandleVote(t *testing.T) {
	config := consensus.Config{
		TotalValidators: 3,
		FaultTolerance:  1,
		RoundDuration:   time.Second,
		WaveTimeout:     time.Second,
	}

	engine := consensus.NewEngine(config)

	// Create a vote
	vote := &consensus.Vote{
		BlockHash:  "proposal1",
		Round:      1,
		Sender:     "validator1",
		Timestamp:  time.Now(),
	}

	// Handle vote
	err := engine.HandleVote(vote)
	assert.NoError(t, err)
}

func TestConsensus_Quorum(t *testing.T) {
	config := consensus.Config{
		TotalValidators: 3,
		FaultTolerance:  1,
		RoundDuration:   time.Second,
		WaveTimeout:     time.Second,
	}

	engine := consensus.NewEngine(config)

	// Create and handle a proposal
	proposal := &consensus.Proposal{
		BlockHash:  "proposal1",
		Round:      1,
		Sender:     "validator1",
		Timestamp:  time.Now(),
	}
	err := engine.HandleProposal(proposal)
	assert.NoError(t, err)

	// Add votes to reach quorum
	validators := []string{"validator1", "validator2", "validator3"}
	for _, validator := range validators[:2] {
		vote := &consensus.Vote{
			BlockHash:  proposal.BlockHash,
			Round:      1,
			Sender:     validator,
			Timestamp:  time.Now(),
		}
		err = engine.HandleVote(vote)
		assert.NoError(t, err)
	}

	// Check quorum
	hasQuorum := engine.HasQuorum(proposal.BlockHash, 1)
	assert.True(t, hasQuorum)
}

func TestConsensus_CompleteWave(t *testing.T) {
	config := consensus.Config{
		TotalValidators: 3,
		FaultTolerance: 1,
		RoundDuration: time.Second,
		WaveTimeout:   time.Second,
	}

	engine := consensus.NewEngine(config)

	// Create and handle a proposal
	proposal := &consensus.Proposal{
		BlockHash:  "proposal1",
		Round:     1,
		Sender:    "validator1",
		Timestamp: time.Now(),
	}
	err := engine.HandleProposal(proposal)
	assert.NoError(t, err)

	// Add votes to reach quorum
	validators := []string{"validator1", "validator2", "validator3"}
	for _, validator := range validators[:2] {
		vote := &consensus.Vote{
			BlockHash:  proposal.BlockHash,
			Round:     1,
			Sender:    validator,
			Timestamp: time.Now(),
		}
		err = engine.HandleVote(vote)
		assert.NoError(t, err)
	}

	// Check quorum
	hasQuorum := engine.HasQuorum(proposal.BlockHash, 1)
	assert.True(t, hasQuorum)
}

func TestConsensusEngine(t *testing.T) {
	config := consensus.Config{
		TotalValidators: 4,
		FaultTolerance: 1,
		RoundDuration: time.Second,
		WaveTimeout:   5 * time.Second,
	}

	engine := consensus.NewEngine(config)

	// Create proposal
	proposal := &consensus.Proposal{
		BlockHash:  "proposal1",
		Round:     1,
		Sender:    "proposer1",
		Timestamp: time.Now(),
	}

	// Handle proposal
	err := engine.HandleProposal(proposal)
	assert.NoError(t, err)

	// Create votes
	votes := make([]*consensus.Vote, 3)
	for i := 0; i < 3; i++ {
		votes[i] = &consensus.Vote{
			BlockHash:  "proposal1",
			Round:     1,
			Sender:    "validator" + string(i+'1'),
			Timestamp: time.Now(),
		}
	}

	// Submit votes
	for _, vote := range votes {
		err = engine.HandleVote(vote)
		assert.NoError(t, err)
	}

	// Check quorum
	hasQuorum := engine.HasQuorum(proposal.BlockHash, 1)
	assert.True(t, hasQuorum)
}

func TestConsensusEngineValidation(t *testing.T) {
	config := consensus.Config{
		TotalValidators: 4,
		FaultTolerance: 1,
		RoundDuration: time.Second,
		WaveTimeout:   5 * time.Second,
	}

	engine := consensus.NewEngine(config)

	// Test invalid proposal (empty block hash)
	invalidProposal := &consensus.Proposal{
		BlockHash:  "",
		Round:     1,
		Sender:    "proposer1",
		Timestamp: time.Now(),
	}
	err := engine.HandleProposal(invalidProposal)
	assert.Error(t, err)

	// Test invalid vote (empty block hash)
	invalidVote := &consensus.Vote{
		BlockHash:  "",
		Round:     1,
		Sender:    "validator1",
		Timestamp: time.Now(),
	}
	err = engine.HandleVote(invalidVote)
	assert.Error(t, err)

	// Test valid proposal and vote
	validProposal := &consensus.Proposal{
		BlockHash:  "validblock",
		Round:     1,
		Sender:    "proposer1",
		Timestamp: time.Now(),
	}
	err = engine.HandleProposal(validProposal)
	assert.NoError(t, err)

	validVote := &consensus.Vote{
		BlockHash:  "validblock",
		Round:     1,
		Sender:    "validator1",
		Timestamp: time.Now(),
	}
	err = engine.HandleVote(validVote)
	assert.NoError(t, err)
}

func TestConsensusEngineQuorum(t *testing.T) {
	config := consensus.Config{
		TotalValidators: 4,
		FaultTolerance: 1,
		RoundDuration: time.Second,
		WaveTimeout:   5 * time.Second,
	}

	engine := consensus.NewEngine(config)

	proposal := &consensus.Proposal{
		BlockHash:  "block1",
		Round:     1,
		Sender:    "proposer1",
		Timestamp: time.Now(),
	}
	err := engine.HandleProposal(proposal)
	assert.NoError(t, err)

	// Add votes one by one and check quorum
	validators := []string{"validator1", "validator2", "validator3", "validator4"}
	for i, validator := range validators {
		vote := &consensus.Vote{
			BlockHash:  proposal.BlockHash,
			Round:     1,
			Sender:    validator,
			Timestamp: time.Now(),
		}
		err = engine.HandleVote(vote)
		assert.NoError(t, err)

		// Check if quorum is reached (should be reached after 3 votes)
		hasQuorum := engine.HasQuorum(proposal.BlockHash, 1)
		if i < 2 {
			assert.False(t, hasQuorum, "Quorum should not be reached with %d votes", i+1)
		} else {
			assert.True(t, hasQuorum, "Quorum should be reached with %d votes", i+1)
		}
	}
} 