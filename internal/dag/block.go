package dag

import (
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// Block represents a block in the DAG
type Block struct {
	Hash       string
	ParentHash string
	References []string
	Timestamp  time.Time
	Data       []byte
	Signature  []byte
}

// NewBlock creates a new block
func NewBlock(parentHash string, references []string, data []byte) *Block {
	return &Block{
		ParentHash: parentHash,
		References: references,
		Data:       data,
		Timestamp:  time.Now(),
	}
}

// ToTypesBlock converts a Block to types.Block
func (b *Block) ToTypesBlock() *types.Block {
	refs := make([]types.Reference, len(b.References))
	for i, ref := range b.References {
		refs[i] = types.Reference{
			BlockHash: []byte(ref),
			Type:      types.ReferenceTypeStandard,
		}
	}

	return &types.Block{
		Header: types.BlockHeader{
			Version:        1,
			Round:          0,
			Wave:           0,
			Height:         0,
			ParentHash:     []byte(b.ParentHash),
			StateRoot:      []byte{},
			TransactionRoot: []byte{},
			ReceiptRoot:    []byte{},
			Validator:      []byte{},
		},
		Body: types.BlockBody{
			Transactions: []types.Transaction{},
			Receipts:     []types.Receipt{},
			Events:       []types.Event{},
		},
		References: refs,
		Signature:  b.Signature,
		Timestamp:  b.Timestamp,
	}
}

// FromTypesBlock creates a Block from types.Block
func FromTypesBlock(block *types.Block) *Block {
	refs := make([]string, len(block.References))
	for i, ref := range block.References {
		refs[i] = string(ref.BlockHash)
	}

	return &Block{
		Hash:       string(block.Hash()),
		ParentHash: string(block.Header.ParentHash),
		References: refs,
		Timestamp:  block.Timestamp,
		Data:       []byte{},
		Signature:  block.Signature,
	}
} 