package state

import (
	"errors"
)

// ProofElementType represents the type of proof element
type ProofElementType int

const (
	ProofElementTypeAccount ProofElementType = iota
	ProofElementTypeStorage
	ProofElementTypeCode
)

// ProofElement represents a proof element
type ProofElement struct {
	Type       ProofElementType
	Address    string
	StorageKey string
	Value      interface{}
}

// StateProof represents a state proof
type StateProof struct {
	Elements []*ProofElement
}

// Errors
var (
	ErrInvalidProof = errors.New("invalid proof")
) 