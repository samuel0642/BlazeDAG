package state

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"time"
)

// StateCommitment represents a state commitment
type StateCommitment struct {
	StateRoot    []byte
	BlockNumber  uint64
	Timestamp    time.Time
	Changes      []*StateChange
	Proof        *StateProof
}

// StateCommitter handles state commitment
type StateCommitter struct {
	commitments []*StateCommitment
	state       *State
	mu          sync.RWMutex
}

// NewStateCommitter creates a new state committer
func NewStateCommitter(state *State) *StateCommitter {
	return &StateCommitter{
		commitments: make([]*StateCommitment, 0),
		state:       state,
	}
}

// CommitState commits the current state
func (sc *StateCommitter) CommitState(blockNumber uint64, changes []*StateChange) *StateCommitment {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Calculate state root
	stateRoot := sc.calculateStateRoot()

	// Generate proof
	proof := sc.generateProof(changes)

	commitment := &StateCommitment{
		StateRoot:    stateRoot,
		BlockNumber:  blockNumber,
		Timestamp:    time.Now(),
		Changes:      changes,
		Proof:        proof,
	}

	sc.commitments = append(sc.commitments, commitment)
	return commitment
}

// calculateStateRoot calculates the state root
func (sc *StateCommitter) calculateStateRoot() []byte {
	h := sha256.New()

	// Hash accounts
	for addr, account := range sc.state.Accounts {
		h.Write([]byte(addr))
		h.Write(account.Balance.Bytes())
		binary.Write(h, binary.LittleEndian, account.Nonce)
		h.Write(account.CodeHash)
		h.Write(account.StorageRoot)
	}

	// Hash storage
	for addr, storage := range sc.state.Storage {
		h.Write([]byte(addr))
		for key, value := range storage {
			h.Write([]byte(key))
			h.Write(value)
		}
	}

	return h.Sum(nil)
}

// generateProof generates a proof for the state changes
func (sc *StateCommitter) generateProof(changes []*StateChange) *StateProof {
	proof := &StateProof{
		Elements: make([]*ProofElement, 0, len(changes)),
	}

	for _, change := range changes {
		element := &ProofElement{
			Type:    sc.getProofElementType(change.Type),
			Address: change.Address,
			Value:   change.NewValue,
		}

		if change.Type == StateChangeTypeStorage {
			element.Key = change.Key
		}

		proof.Elements = append(proof.Elements, element)
	}

	return proof
}

// getProofElementType converts a state change type to a proof element type
func (sc *StateCommitter) getProofElementType(changeType StateChangeType) ProofElementType {
	switch changeType {
	case StateChangeTypeAccount:
		return ProofElementTypeAccount
	case StateChangeTypeStorage:
		return ProofElementTypeStorage
	case StateChangeTypeCode:
		return ProofElementTypeCode
	default:
		return ProofElementTypeAccount
	}
}

// GetCommitment returns a commitment by block number
func (sc *StateCommitter) GetCommitment(blockNumber uint64) *StateCommitment {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for _, commitment := range sc.commitments {
		if commitment.BlockNumber == blockNumber {
			return commitment
		}
	}
	return nil
}

// GetLatestCommitment returns the latest commitment
func (sc *StateCommitter) GetLatestCommitment() *StateCommitment {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if len(sc.commitments) == 0 {
		return nil
	}
	return sc.commitments[len(sc.commitments)-1]
}

// VerifyCommitment verifies a state commitment
func (sc *StateCommitter) VerifyCommitment(commitment *StateCommitment) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Verify state root
	calculatedRoot := sc.calculateStateRoot()
	if string(calculatedRoot) != string(commitment.StateRoot) {
		return false
	}

	// Verify changes
	for _, change := range commitment.Changes {
		if !sc.verifyChange(change) {
			return false
		}
	}

	// Verify proof
	verifier := NewStateVerifier(sc.state)
	return verifier.VerifyProof(commitment.Proof) == nil
}

// verifyChange verifies a state change
func (sc *StateCommitter) verifyChange(change *StateChange) bool {
	switch change.Type {
	case StateChangeTypeAccount:
		account := sc.state.GetAccount(change.Address)
		return account.Equal(change.NewValue.(*Account))
	case StateChangeTypeStorage:
		storage := sc.state.GetStorage(change.Address)
		value, exists := storage[change.Key]
		return exists && string(value) == string(change.NewValue.([]byte))
	case StateChangeTypeCode:
		code := sc.state.GetCode(change.Address)
		return string(code) == string(change.NewValue.([]byte))
	case StateChangeTypeBalance:
		balance := sc.state.GetBalance(change.Address)
		return balance == change.NewValue.(uint64)
	case StateChangeTypeNonce:
		nonce := sc.state.GetNonce(change.Address)
		return nonce == change.NewValue.(uint64)
	default:
		return false
	}
} 