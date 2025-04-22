package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

var (
	ErrInvalidStateRoot = errors.New("invalid state root")
	ErrInconsistentState = errors.New("inconsistent state")
)

// StateVerifier handles state verification
type StateVerifier struct {
	state *State
}

// NewStateVerifier creates a new state verifier
func NewStateVerifier(state *State) *StateVerifier {
	return &StateVerifier{
		state: state,
	}
}

// VerifyStateRoot verifies the state root
func (sv *StateVerifier) VerifyStateRoot(root []byte) error {
	calculatedRoot := sv.calculateStateRoot()
	if string(calculatedRoot) != string(root) {
		return ErrInvalidStateRoot
	}
	return nil
}

// VerifyProof verifies a state proof
func (sv *StateVerifier) VerifyProof(proof *types.StateProof) error {
	if proof == nil {
		return types.ErrInvalidProof
	}

	for _, element := range proof.Elements {
		switch element.Type {
		case types.ProofElementTypeAccount:
			if err := sv.verifyAccountProof(element); err != nil {
				return err
			}
		case types.ProofElementTypeStorage:
			if err := sv.verifyStorageProof(element); err != nil {
				return err
			}
		case types.ProofElementTypeCode:
			if err := sv.verifyCodeProof(element); err != nil {
				return err
			}
		default:
			return types.ErrInvalidProof
		}
	}

	return nil
}

// VerifyConsistency verifies state consistency
func (sv *StateVerifier) VerifyConsistency() error {
	// Verify account consistency
	for addr, account := range sv.state.accounts {
		// Verify storage root
		storageRoot := sv.calculateStorageRoot(addr)
		if string(storageRoot) != string(account.StorageRoot) {
			return ErrInconsistentState
		}

		// Verify code hash
		codeHash := sv.calculateCodeHash(addr)
		if !bytes.Equal(account.CodeHash, codeHash[:]) {
			return ErrInconsistentState
		}
	}

	return nil
}

// calculateStateRoot calculates the state root
func (sv *StateVerifier) calculateStateRoot() []byte {
	h := sha256.New()

	// Hash accounts
	for addr, account := range sv.state.accounts {
		h.Write([]byte(addr))
		binary.Write(h, binary.LittleEndian, account.Balance)
		binary.Write(h, binary.LittleEndian, account.Nonce)
		h.Write(account.CodeHash)
		h.Write(account.StorageRoot)
	}

	// Hash storage
	for addr, storage := range sv.state.storage {
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
	storage := sv.state.storage[addr]

	for key, value := range storage {
		h.Write([]byte(key))
		h.Write(value)
	}

	return h.Sum(nil)
}

// calculateCodeHash calculates the code hash for an account
func (sv *StateVerifier) calculateCodeHash(addr string) [32]byte {
	code := sv.state.storage[addr]["code"]
	if len(code) == 0 {
		return [32]byte{}
	}
	return sha256.Sum256(code)
}

// verifyAccountProof verifies an account proof
func (sv *StateVerifier) verifyAccountProof(element *types.ProofElement) error {
	account, ok := element.Value.(*types.Account)
	if !ok {
		return types.ErrInvalidProof
	}

	// Verify account data
	stateAccount := sv.state.accounts[element.Address]
	if account.Balance != stateAccount.Balance {
		return types.ErrInvalidProof
	}

	if account.Nonce != stateAccount.Nonce {
		return types.ErrInvalidProof
	}

	codeHash := sv.calculateCodeHash(element.Address)
	if !bytes.Equal(account.CodeHash, codeHash[:]) {
		return types.ErrInvalidProof
	}

	return nil
}

// verifyStorageProof verifies a storage proof
func (sv *StateVerifier) verifyStorageProof(element *types.ProofElement) error {
	value, ok := element.Value.([]byte)
	if !ok {
		return types.ErrInvalidProof
	}

	// Verify storage value
	storageValue := sv.state.storage[element.Address][element.StorageKey]
	if !bytes.Equal(value, storageValue) {
		return types.ErrInvalidProof
	}

	return nil
}

// verifyCodeProof verifies a code proof
func (sv *StateVerifier) verifyCodeProof(element *types.ProofElement) error {
	code, ok := element.Value.([]byte)
	if !ok {
		return types.ErrInvalidProof
	}

	// Verify code
	stateCode := sv.state.storage[element.Address]["code"]
	if !bytes.Equal(code, stateCode) {
		return types.ErrInvalidProof
	}

	return nil
} 