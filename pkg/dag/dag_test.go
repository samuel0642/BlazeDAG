package dag

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to calculate block hash
func calculateBlockHash(block *types.Block) []byte {
	hasher := sha256.New()
	hasher.Write(block.Header.ParentHash)
	for _, tx := range block.Body.Transactions {
		hasher.Write(tx.From)
		hasher.Write(tx.To)
	}
	for _, ref := range block.References {
		hasher.Write(ref.BlockHash)
	}
	return hasher.Sum(nil)
}

func TestDAG_AddBlock(t *testing.T) {
	d := NewDAG()

	// Create a block
	block := &types.Block{
		Header: types.BlockHeader{
			Version:    1,
			Round:      1,
			Wave:       0,
			Height:     1,
			ParentHash: []byte("genesis"),
		},
		Body: types.BlockBody{
			Transactions: []types.Transaction{
				{
					From:  []byte("sender1"),
					To:    []byte("receiver1"),
					Value: 100,
				},
				{
					From:  []byte("sender2"),
					To:    []byte("receiver2"),
					Value: 200,
				},
			},
		},
		Certificate: &types.Certificate{
			BlockHash:    []byte("block1"),
			Signatures:   [][]byte{[]byte("sig1"), []byte("sig2")},
			Round:        1,
			Wave:        0,
			ValidatorSet: [][]byte{[]byte("validator1"), []byte("validator2")},
			Timestamp:   time.Now(),
		},
		References: []types.Reference{
			{
				BlockHash: []byte("genesis"),
				Round:     0,
				Wave:      0,
				Type:      types.ReferenceTypeStandard,
			},
		},
		Signature: []byte("block_sig"),
		Timestamp: time.Now(),
	}

	// Add genesis block first
	genesis := &types.Block{
		Header: types.BlockHeader{
			Version:    1,
			Round:      0,
			Wave:       0,
			Height:     0,
			ParentHash: []byte("genesis"),
		},
		Timestamp: time.Now(),
	}
	err := d.AddBlock(genesis)
	require.NoError(t, err)

	// Add the block
	err = d.AddBlock(block)
	require.NoError(t, err)

	// Calculate block hash
	blockHash := calculateBlockHash(block)

	// Verify the block was added
	addedBlock, exists := d.GetBlock(blockHash)
	assert.True(t, exists)
	assert.Equal(t, block, addedBlock)
}

func TestDAG_GetTips(t *testing.T) {
	d := NewDAG()

	// Add genesis block first
	genesis := &types.Block{
		Header: types.BlockHeader{
			Version:    1,
			Round:      0,
			Wave:       0,
			Height:     0,
			ParentHash: []byte("genesis"),
		},
		Timestamp: time.Now(),
	}
	err := d.AddBlock(genesis)
	require.NoError(t, err)

	// Create and add a block
	block := &types.Block{
		Header: types.BlockHeader{
			Version:    1,
			Round:      1,
			Wave:       0,
			Height:     1,
			ParentHash: []byte("genesis"),
		},
		Body: types.BlockBody{
			Transactions: []types.Transaction{
				{
					From:  []byte("sender1"),
					To:    []byte("receiver1"),
					Value: 100,
				},
			},
		},
		References: []types.Reference{
			{
				BlockHash: []byte("genesis"),
				Round:     0,
				Wave:      0,
				Type:      types.ReferenceTypeStandard,
			},
		},
		Timestamp: time.Now(),
	}

	err = d.AddBlock(block)
	require.NoError(t, err)

	// Calculate block hash
	blockHash := calculateBlockHash(block)

	// Get tips
	tips := d.GetTips()
	require.Len(t, tips, 1)
	assert.Equal(t, blockHash, tips[0])
}

func TestDAG_ComplexStructure(t *testing.T) {
	d := NewDAG()

	// Create two parent blocks with same height
	parent1 := &types.Block{
		Header: types.BlockHeader{
			Version:    1,
			Round:      1,
			Wave:       0,
			Height:     1,
			ParentHash: []byte("genesis"),
		},
		Body: types.BlockBody{
			Transactions: []types.Transaction{
				{
					From:  []byte("sender1"),
					To:    []byte("receiver1"),
					Value: 100,
				},
			},
		},
		References: []types.Reference{
			{
				BlockHash: []byte("genesis"),
				Round:     0,
				Wave:      0,
				Type:      types.ReferenceTypeStandard,
			},
		},
		Timestamp: time.Now(),
	}

	parent2 := &types.Block{
		Header: types.BlockHeader{
			Version:    1,
			Round:      1,
			Wave:       0,
			Height:     1,
			ParentHash: []byte("genesis"),
		},
		Body: types.BlockBody{
			Transactions: []types.Transaction{
				{
					From:  []byte("sender2"),
					To:    []byte("receiver2"),
					Value: 200,
				},
			},
		},
		References: []types.Reference{
			{
				BlockHash: []byte("genesis"),
				Round:     0,
				Wave:      0,
				Type:      types.ReferenceTypeStandard,
			},
		},
		Timestamp: time.Now(),
	}

	// Add parent blocks
	err := d.AddBlock(parent1)
	assert.NoError(t, err)
	err = d.AddBlock(parent2)
	assert.NoError(t, err)

	// Create a child block that references both parents
	child := &types.Block{
		Header: types.BlockHeader{
			Version:    1,
			Round:      2,
			Wave:       0,
			Height:     2,
			ParentHash: parent1.Hash(),
		},
		Body: types.BlockBody{
			Transactions: []types.Transaction{
				{
					From:  []byte("sender3"),
					To:    []byte("receiver3"),
					Value: 300,
				},
			},
		},
		References: []types.Reference{
			{
				BlockHash: parent1.Hash(),
				Round:     1,
				Wave:      0,
				Type:      types.ReferenceTypeStandard,
			},
			{
				BlockHash: parent2.Hash(),
				Round:     1,
				Wave:      0,
				Type:      types.ReferenceTypeStandard,
			},
		},
		Timestamp: time.Now(),
	}

	// Add child block
	err = d.AddBlock(child)
	assert.NoError(t, err)

	// Verify block references
	_, exists := d.GetBlock(parent1.Hash())
	assert.True(t, exists)
	_, exists = d.GetBlock(parent2.Hash())
	assert.True(t, exists)
	_, exists = d.GetBlock(child.Hash())
	assert.True(t, exists)

	// Verify tips (only child should be a tip)
	tips := d.GetTips()
	assert.Len(t, tips, 1)
	assert.Equal(t, child.Hash(), tips[0])
}

func TestDAG(t *testing.T) {
	d := NewDAG()

	// Create test blocks
	block1 := &types.Block{
		Header: types.BlockHeader{
			ParentHash: []byte("block1"),
		},
		References: []types.Reference{
			{
				BlockHash: []byte("genesis"),
			},
		},
	}

	block2 := &types.Block{
		Header: types.BlockHeader{
			ParentHash: []byte("block2"),
		},
		References: []types.Reference{
			{
				BlockHash: []byte("block1"),
			},
		},
	}

	block3 := &types.Block{
		Header: types.BlockHeader{
			ParentHash: []byte("block3"),
		},
		References: []types.Reference{
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
		Header: types.BlockHeader{
			ParentHash: []byte("block1"),
		},
		References: []types.Reference{
			{
				BlockHash: []byte("genesis"),
			},
		},
	}

	block2 := &types.Block{
		Header: types.BlockHeader{
			ParentHash: []byte("block2"),
		},
		References: []types.Reference{
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