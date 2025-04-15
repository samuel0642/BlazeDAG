package dag

import (
	"testing"
	"time"

	"github.com/samuel0642/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNewDAG(t *testing.T) {
	dag := NewDAG()
	assert.NotNil(t, dag)
	assert.NotNil(t, dag.blocks)
	assert.NotNil(t, dag.tips)
	assert.NotNil(t, dag.references)
	assert.NotNil(t, dag.children)
	assert.NotNil(t, dag.committedBlocks)
	assert.NotNil(t, dag.pendingBlocks)
	assert.NotNil(t, dag.causalOrder)
}

func TestAddBlock(t *testing.T) {
	d := NewDAG()
	block := &types.Block{
		Header: types.BlockHeader{
			Version:    0,
			Round:      1,
			Wave:       0,
			Height:     1,
			ParentHash: []byte("parent"),
		},
		References: []types.Reference{},
		Timestamp:  time.Now(),
	}

	err := d.AddBlock(block)
	assert.NoError(t, err)

	// Verify block was added
	addedBlock, exists := d.GetBlock(block.Hash())
	assert.True(t, exists)
	assert.Equal(t, block, addedBlock)
}

func TestGetTips(t *testing.T) {
	d := NewDAG()
	block := &types.Block{
		Header: types.BlockHeader{
			Version:    0,
			Round:      1,
			Wave:       0,
			Height:     1,
			ParentHash: []byte("parent"),
		},
		References: []types.Reference{},
		Timestamp:  time.Now(),
	}

	err := d.AddBlock(block)
	assert.NoError(t, err)

	tips := d.GetTips()
	assert.Len(t, tips, 1)
	assert.Contains(t, tips, block.Hash())
}

func TestCommitBlock(t *testing.T) {
	d := NewDAG()
	block := &types.Block{
		Header: types.BlockHeader{
			Version:    0,
			Round:      1,
			Wave:       0,
			Height:     1,
			ParentHash: []byte("parent"),
		},
		References: []types.Reference{},
		Timestamp:  time.Now(),
	}

	err := d.AddBlock(block)
	assert.NoError(t, err)

	err = d.CommitBlock(block.Hash())
	assert.NoError(t, err)

	assert.True(t, d.IsCommitted(block.Hash()))
} 