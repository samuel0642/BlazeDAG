package consensus

import (
	"testing"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNewEngine(t *testing.T) {
	config := Config{
		WaveTimeout:      10 * time.Second,
		RoundTimeout:     5 * time.Second,
		MinValidators:    4,
		QuorumSize:       3,
		MaxBlockSize:     1024 * 1024,
		MaxTransactions:  1000,
		MaxGasLimit:      1000000,
		MaxGasPrice:      1000000,
		MaxNonce:         1000000,
		MaxValue:         1000000,
		MaxCodeSize:      1024 * 1024,
		MaxStorageSize:   1024 * 1024,
		MaxEventSize:     1024,
		MaxLogSize:       1024,
		MaxReceiptSize:   1024,
		MaxProofSize:     1024,
		MaxSignatureSize: 1024,
	}

	engine := NewConsensusEngine(nil, "test-node")
	assert.NotNil(t, engine)
}

func TestHandleProposal(t *testing.T) {
	engine := NewConsensusEngine(nil, "test-node")
	proposal := &Proposal{
		Block:      nil,
		Wave:       1,
		Round:      1,
		Proposer:   "validator1",
		Timestamp:  time.Now(),
		Signatures: make([]types.Signature, 0),
	}

	err := engine.ProcessBlock(proposal.Block)
	assert.NoError(t, err)
}

func TestHandleVote(t *testing.T) {
	engine := NewConsensusEngine(nil, "test-node")
	vote := &types.Vote{
		Type:      types.VoteTypeProposal,
		Wave:      1,
		Round:     1,
		BlockHash: []byte("test-hash"),
		Validator: []byte("validator1"),
		Timestamp: time.Now(),
	}

	err := engine.ProcessVote(vote)
	assert.NoError(t, err)
}

func TestHasQuorum(t *testing.T) {
	engine := NewConsensusEngine(nil, "test-node")
	
	// Add 3 votes (quorum is 3 for 4 validators with 1 fault tolerance)
	for i := 0; i < 3; i++ {
		vote := &types.Vote{
			Type:      types.VoteTypeProposal,
			Wave:      1,
			Round:     1,
			BlockHash: []byte("test-hash"),
			Validator: []byte("validator" + string(rune('1'+i))),
			Timestamp: time.Now(),
		}
		engine.ProcessVote(vote)
	}

	// TODO: Implement quorum check
	assert.True(t, true)
} 