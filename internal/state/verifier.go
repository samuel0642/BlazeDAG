package state

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sync"
)

var (
	ErrInvalidStateRoot = errors.New("invalid state root")
	ErrInvalidProof     = errors.New("invalid proof")
	ErrInconsistentState = errors.New("inconsistent state")
)

// StateVerifier handles state verification
type StateVerifier struct {
	state *State
	mu    sync.RWMutex
}

// NewStateVerifier creates a new state verifier
func NewStateVerifier(state *State) *StateVerifier {
	return &StateVerifier{
		state: state,
	}
}

// VerifyStateRoot verifies the state root
func (sv *StateVerifier) VerifyStateRoot(root []byte) error {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	calculatedRoot := sv.calculateStateRoot()
	if string(calculatedRoot) != string(root) {
		return ErrInvalidStateRoot
	}
	return nil
}

// VerifyProof verifies a state proof
func (sv *StateVerifier) VerifyProof(proof *StateProof) error {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	// Verify each proof element
	for _, element := range proof.Elements {
		if !sv.verifyProofElement(element) {
			return ErrInvalidProof
		}
	}

	return nil
}

// VerifyConsistency verifies state consistency
func (sv *StateVerifier) VerifyConsistency() error {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	// Verify account consistency
	for addr, account := range sv.state.Accounts {
		// Verify storage root
		storageRoot := sv.calculateStorageRoot(addr)
		if string(storageRoot) != string(account.StorageRoot) {
			return ErrInconsistentState
		}

		// Verify code hash
		codeHash := sv.calculateCodeHash(addr)
		if string(codeHash) != string(account.CodeHash) {
			return ErrInconsistentState
		}
	}

	return nil
}

// calculateStateRoot calculates the state root
func (sv *StateVerifier) calculateStateRoot() []byte {
	h := sha256.New()

	// Hash accounts
	for addr, account := range sv.state.Accounts {
		h.Write([]byte(addr))
		h.Write(account.Balance.Bytes())
		binary.Write(h, binary.LittleEndian, account.Nonce)
		h.Write(account.CodeHash)
		h.Write(account.StorageRoot)
	}

	// Hash storage
	for addr, storage := range sv.state.Storage {
		h.Write([]byte(addr))
		for key, value := range storage {
			h.Write([]byte(key))
			h.Write(value)
		}
	}

	return h.Sum(nil)
}

// calculateStorageRoot calculates the storage root for an account
func (sv *StateVerifier) calculateStorageRoot(addr string) []byte {
	h := sha256.New()
	storage := sv.state.Storage[addr]

	for key, value := range storage {
		h.Write([]byte(key))
		h.Write(value)
	}

	return h.Sum(nil)
}

// calculateCodeHash calculates the code hash for an account
func (sv *StateVerifier) calculateCodeHash(addr string) []byte {
	code := sv.state.Code[addr]
	h := sha256.New()
	h.Write(code)
	return h.Sum(nil)
}

// verifyProofElement verifies a proof element
func (sv *StateVerifier) verifyProofElement(element *ProofElement) bool {
	switch element.Type {
	case ProofElementTypeAccount:
		return sv.verifyAccountProof(element)
	case ProofElementTypeStorage:
		return sv.verifyStorageProof(element)
	case ProofElementTypeCode:
		return sv.verifyCodeProof(element)
	default:
		return false
	}
}

// verifyAccountProof verifies an account proof
func (sv *StateVerifier) verifyAccountProof(element *ProofElement) bool {
	account := sv.state.Accounts[element.Address]
	if account == nil {
		return false
	}

	// Verify account data matches proof
	return account.Equal(element.Value.(*Account))
}

// verifyStorageProof verifies a storage proof
func (sv *StateVerifier) verifyStorageProof(element *ProofElement) bool {
	storage := sv.state.Storage[element.Address]
	if storage == nil {
		return false
	}

	// Verify storage value matches proof
	value, exists := storage[element.Key]
	if !exists {
		return false
	}

	return string(value) == string(element.Value.([]byte))
}

// verifyCodeProof verifies a code proof
func (sv *StateVerifier) verifyCodeProof(element *ProofElement) bool {
	code := sv.state.Code[element.Address]
	if code == nil {
		return false
	}

	// Verify code matches proof
	return string(code) == string(element.Value.([]byte))
}

// StateProof represents a state proof
type StateProof struct {
	Elements []*ProofElement
}

// ProofElement represents a proof element
type ProofElement struct {
	Type    ProofElementType
	Address string
	Key     string
	Value   interface{}
}

// ProofElementType represents the type of proof element
type ProofElementType int

const (
	ProofElementTypeAccount ProofElementType = iota
	ProofElementTypeStorage
	ProofElementTypeCode
) 