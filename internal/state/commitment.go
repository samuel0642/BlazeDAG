package state

import (
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// StateCommitment represents a state commitment
type StateCommitment struct {
	StateRoot    []byte
	BlockNumber  uint64
	Timestamp    time.Time
	Changes      []*StateChange
	Proof        *types.StateProof
}

// StateCommitter handles state commitments
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

// CommitState commits a state change
func (sc *StateCommitter) CommitState(blockNumber uint64, changes []*StateChange) *StateCommitment {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Calculate state root
	stateRoot := sc.calculateStateRoot()

	// Generate proof
	proof := sc.generateProof(changes)

	// Create commitment
	commitment := &StateCommitment{
		StateRoot:    stateRoot,
		BlockNumber:  blockNumber,
		Timestamp:    time.Now(),
		Changes:      changes,
		Proof:        proof,
	}

	// Add to commitments
	sc.commitments = append(sc.commitments, commitment)

	return commitment
}

// calculateStateRoot calculates the state root
func (sc *StateCommitter) calculateStateRoot() []byte {
	verifier := NewStateVerifier(sc.state)
	return verifier.calculateStateRoot()
}

// generateProof generates a state proof
func (sc *StateCommitter) generateProof(changes []*StateChange) *types.StateProof {
	elements := make([]*types.ProofElement, 0)

	for _, change := range changes {
		element := &types.ProofElement{
			Type:       sc.getProofElementType(change.Type),
			Address:    change.Address,
			StorageKey: change.StorageKey,
			Value:      change.NewValue,
		}
		elements = append(elements, element)
	}

	return &types.StateProof{
		Elements: elements,
		Root:     sc.calculateStateRoot(),
	}
}

// getProofElementType converts a state change type to a proof element type
func (sc *StateCommitter) getProofElementType(changeType StateChangeType) types.ProofElementType {
	switch changeType {
	case StateChangeTypeAccount:
		return types.ProofElementTypeAccount
	case StateChangeTypeStorage:
		return types.ProofElementTypeStorage
	case StateChangeTypeCode:
		return types.ProofElementTypeCode
	default:
		return types.ProofElementTypeAccount
	}
}

// GetCommitment gets a commitment by block number
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

// GetLatestCommitment gets the latest commitment
func (sc *StateCommitter) GetLatestCommitment() *StateCommitment {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if len(sc.commitments) == 0 {
		return nil
	}

	return sc.commitments[len(sc.commitments)-1]
}

// VerifyCommitment verifies a commitment
func (sc *StateCommitter) VerifyCommitment(commitment *StateCommitment) bool {
	if commitment == nil {
		return false
	}

	// Verify proof
	verifier := NewStateVerifier(sc.state)
	return verifier.VerifyProof(commitment.Proof) == nil
}

// verifyChange verifies a state change
func (sc *StateCommitter) verifyChange(change *StateChange) bool {
	switch change.Type {
	case StateChangeTypeAccount:
		account, err := sc.state.GetAccount(change.Address)
		if err != nil {
			return false
		}
		return account.Balance == change.NewValue.(uint64)
	case StateChangeTypeStorage:
		value, err := sc.state.GetStorage([]byte(change.Address), []byte(change.StorageKey))
		if err != nil {
			return false
		}
		return string(value) == string(change.NewValue.([]byte))
	case StateChangeTypeCode:
		code, err := sc.state.GetCode([]byte(change.Address))
		if err != nil {
			return false
		}
		return string(code) == string(change.NewValue.([]byte))
	case StateChangeTypeBalance:
		account, err := sc.state.GetAccount(change.Address)
		if err != nil {
			return false
		}
		return account.Balance == change.NewValue.(uint64)
	case StateChangeTypeNonce:
		account, err := sc.state.GetAccount(change.Address)
		if err != nil {
			return false
		}
		return account.Nonce == change.NewValue.(uint64)
	default:
		return false
	}
} 