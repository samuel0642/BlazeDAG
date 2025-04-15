package consensus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEngine(t *testing.T) {
	config := Config{
		TotalValidators: 4,
		FaultTolerance:  1,
		RoundDuration:   5 * time.Second,
		WaveTimeout:     2 * time.Second,
	}

	engine := NewEngine(config)
	assert.NotNil(t, engine)
	assert.Equal(t, config, engine.config)
	assert.NotNil(t, engine.proposals)
	assert.NotNil(t, engine.votes)
}

func TestHandleProposal(t *testing.T) {
	config := Config{
		TotalValidators: 4,
		FaultTolerance:  1,
		RoundDuration:   5 * time.Second,
		WaveTimeout:     2 * time.Second,
	}

	engine := NewEngine(config)
	proposal := &Proposal{
		BlockHash: "test-hash",
		Round:     1,
		Sender:    "validator1",
		Timestamp: time.Now(),
	}

	err := engine.HandleProposal(proposal)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(engine.proposals))
}

func TestHandleVote(t *testing.T) {
	config := Config{
		TotalValidators: 4,
		FaultTolerance:  1,
		RoundDuration:   5 * time.Second,
		WaveTimeout:     2 * time.Second,
	}

	engine := NewEngine(config)
	vote := &Vote{
		BlockHash: "test-hash",
		Round:     1,
		Sender:    "validator1",
		Timestamp: time.Now(),
	}

	err := engine.HandleVote(vote)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(engine.votes["test-hash"]))
}

func TestHasQuorum(t *testing.T) {
	config := Config{
		TotalValidators: 4,
		FaultTolerance:  1,
		RoundDuration:   5 * time.Second,
		WaveTimeout:     2 * time.Second,
	}

	engine := NewEngine(config)
	
	// Add 3 votes (quorum is 3 for 4 validators with 1 fault tolerance)
	for i := 0; i < 3; i++ {
		vote := &Vote{
			BlockHash: "test-hash",
			Round:     1,
			Sender:    "validator" + string(rune('1'+i)),
			Timestamp: time.Now(),
		}
		engine.HandleVote(vote)
	}

	hasQuorum := engine.HasQuorum("test-hash", 1)
	assert.True(t, hasQuorum)
} 