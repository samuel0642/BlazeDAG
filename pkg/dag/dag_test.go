package dag

import (
	"testing"
	"time"

	"BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestDAG(t *testing.T) {
	d := NewDAG()

	// Create test blocks
	block1 := &types.Block{
		Header: &types.BlockHeader{
			ParentHash: []byte("block1"),
		},
		References: []*types.Reference{
			{
				BlockHash: []byte("genesis"),
			},
		},
	}

	block2 := &types.Block{
		Header: &types.BlockHeader{
			ParentHash: []byte("block2"),
		},
		References: []*types.Reference{
			{
				BlockHash: []byte("block1"),
			},
		},
	}

	block3 := &types.Block{
		Header: &types.BlockHeader{
			ParentHash: []byte("block3"),
		},
		References: []*types.Reference{
			{
				BlockHash: []byte("block1"),
			},
			{
				BlockHash: []byte("block2"),
			},
		},
	}

	// Add blocks
	err := d.AddBlock(block1)
	assert.NoError(t, err)
	err = d.AddBlock(block2)
	assert.NoError(t, err)
	err = d.AddBlock(block3)
	assert.NoError(t, err)

	// Test GetBlock
	b, ok := d.GetBlock([]byte("block1"))
	assert.True(t, ok)
	assert.Equal(t, block1, b)

	// Test GetTips
	tips := d.GetTips()
	assert.Len(t, tips, 1)
	assert.Equal(t, []byte("block3"), tips[0])

	// Test GetReferences
	refs := d.GetReferences([]byte("block3"))
	assert.Len(t, refs, 2)
	assert.Contains(t, refs, []byte("block1"))
	assert.Contains(t, refs, []byte("block2"))

	// Test GetChildren
	children := d.GetChildren([]byte("block1"))
	assert.Len(t, children, 2)
	assert.Contains(t, children, []byte("block2"))
	assert.Contains(t, children, []byte("block3"))

	// Test CommitBlock
	err = d.CommitBlock([]byte("block1"))
	assert.NoError(t, err)
	assert.True(t, d.IsCommitted([]byte("block1")))

	// Test GetCausalOrder
	order, ok := d.GetCausalOrder([]byte("block1"))
	assert.True(t, ok)
	assert.Equal(t, uint64(1), order)

	err = d.CommitBlock([]byte("block2"))
	assert.NoError(t, err)
	order, ok = d.GetCausalOrder([]byte("block2"))
	assert.True(t, ok)
	assert.Equal(t, uint64(2), order)

	err = d.CommitBlock([]byte("block3"))
	assert.NoError(t, err)
	order, ok = d.GetCausalOrder([]byte("block3"))
	assert.True(t, ok)
	assert.Equal(t, uint64(3), order)
}

func TestDAGErrors(t *testing.T) {
	d := NewDAG()

	// Test CommitBlock with non-existent block
	err := d.CommitBlock([]byte("nonexistent"))
	assert.Equal(t, ErrBlockNotFound, err)

	// Test GetBlock with non-existent block
	_, ok := d.GetBlock([]byte("nonexistent"))
	assert.False(t, ok)
}

func TestDAGConcurrency(t *testing.T) {
	d := NewDAG()
	done := make(chan bool)

	// Create test blocks
	block1 := &types.Block{
		Header: &types.BlockHeader{
			ParentHash: []byte("block1"),
		},
		References: []*types.Reference{
			{
				BlockHash: []byte("genesis"),
			},
		},
	}

	block2 := &types.Block{
		Header: &types.BlockHeader{
			ParentHash: []byte("block2"),
		},
		References: []*types.Reference{
			{
				BlockHash: []byte("block1"),
			},
		},
	}

	// Concurrent block addition
	go func() {
		err := d.AddBlock(block1)
		assert.NoError(t, err)
		done <- true
	}()

	go func() {
		err := d.AddBlock(block2)
		assert.NoError(t, err)
		done <- true
	}()

	// Wait for goroutines to complete
	<-done
	<-done

	// Verify blocks were added correctly
	b1, ok := d.GetBlock([]byte("block1"))
	assert.True(t, ok)
	assert.Equal(t, block1, b1)

	b2, ok := d.GetBlock([]byte("block2"))
	assert.True(t, ok)
	assert.Equal(t, block2, b2)
}

func TestDAGComplexStructure(t *testing.T) {
	d := NewDAG()

	// Create a more complex DAG structure
	blocks := make([]*types.Block, 10)
	for i := 0; i < 10; i++ {
		blocks[i] = &types.Block{
			Header: &types.BlockHeader{
				ParentHash: []byte("block" + string(rune(i))),
			},
		}
	}

	// Add references
	blocks[1].References = []*types.Reference{
		{BlockHash: []byte("block0")},
	}
	blocks[2].References = []*types.Reference{
		{BlockHash: []byte("block0")},
		{BlockHash: []byte("block1")},
	}
	blocks[3].References = []*types.Reference{
		{BlockHash: []byte("block1")},
		{BlockHash: []byte("block2")},
	}
	blocks[4].References = []*types.Reference{
		{BlockHash: []byte("block2")},
	}
	blocks[5].References = []*types.Reference{
		{BlockHash: []byte("block3")},
		{BlockHash: []byte("block4")},
	}

	// Add blocks
	for _, block := range blocks {
		err := d.AddBlock(block)
		assert.NoError(t, err)
	}

	// Test tips
	tips := d.GetTips()
	assert.Len(t, tips, 5) // blocks 5-9 should be tips

	// Test references
	refs := d.GetReferences([]byte("block5"))
	assert.Len(t, refs, 2)
	assert.Contains(t, refs, []byte("block3"))
	assert.Contains(t, refs, []byte("block4"))

	// Test children
	children := d.GetChildren([]byte("block2"))
	assert.Len(t, children, 2)
	assert.Contains(t, children, []byte("block3"))
	assert.Contains(t, children, []byte("block4"))

	// Commit blocks and test causal order
	for i := 0; i < 6; i++ {
		err := d.CommitBlock([]byte("block" + string(rune(i))))
		assert.NoError(t, err)
	}

	order, ok := d.GetCausalOrder([]byte("block5"))
	assert.True(t, ok)
	assert.Equal(t, uint64(6), order)
} 