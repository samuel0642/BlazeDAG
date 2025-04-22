package types

import "errors"

var (
	ErrInvalidProof = errors.New("invalid proof")
)

// ProofElementType represents the type of proof element
type ProofElementType int

const (
	ProofElementTypeAccount ProofElementType = iota
	ProofElementTypeStorage
	ProofElementTypeCode
)

// ProofElement represents a single element in a state proof
type ProofElement struct {
	Type       ProofElementType
	Address    string
	StorageKey string
	Value      interface{}
}

// StateProof represents a complete state proof
type StateProof struct {
	Elements []*ProofElement
	Root     []byte
} 