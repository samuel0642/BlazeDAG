package consensus

import (
	"sync"
	"time"
)

// Certificate represents a block certificate
type Certificate struct {
	BlockHash    string
	Signatures   []*Signature
	Round        int
	Wave         int
	ValidatorSet []string
	Timestamp    time.Time
}

// Signature represents a validator's signature
type Signature struct {
	Validator  string
	Signature  []byte
	Timestamp  time.Time
}

// BlockFinalizer handles block finalization
type BlockFinalizer struct {
	certificates map[string]*Certificate
	mu           sync.RWMutex
}

// NewBlockFinalizer creates a new block finalizer
func NewBlockFinalizer() *BlockFinalizer {
	return &BlockFinalizer{
		certificates: make(map[string]*Certificate),
	}
}

// AddSignature adds a signature to a block's certificate
func (bf *BlockFinalizer) AddSignature(blockHash string, signature *Signature) {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	cert, exists := bf.certificates[blockHash]
	if !exists {
		cert = &Certificate{
			BlockHash: blockHash,
			Signatures: make([]*Signature, 0),
		}
		bf.certificates[blockHash] = cert
	}

	cert.Signatures = append(cert.Signatures, signature)
}

// GetCertificate returns the certificate for a block
func (bf *BlockFinalizer) GetCertificate(blockHash string) *Certificate {
	bf.mu.RLock()
	defer bf.mu.RUnlock()
	return bf.certificates[blockHash]
}

// HasQuorum checks if a block has enough signatures to reach quorum
func (bf *BlockFinalizer) HasQuorum(blockHash string, totalValidators int, faultTolerance int) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	cert := bf.certificates[blockHash]
	if cert == nil {
		return false
	}

	// Count unique validators who signed
	validators := make(map[string]bool)
	for _, sig := range cert.Signatures {
		validators[sig.Validator] = true
	}

	// Check if we have enough signatures (2f+1)
	return len(validators) >= (2*faultTolerance + 1)
}

// FinalizeBlock finalizes a block
func (bf *BlockFinalizer) FinalizeBlock(blockHash string, round int, wave int, validatorSet []string) *Certificate {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	cert := bf.certificates[blockHash]
	if cert == nil {
		return nil
	}

	cert.Round = round
	cert.Wave = wave
	cert.ValidatorSet = validatorSet
	cert.Timestamp = time.Now()

	return cert
}

// GetFinalizedBlocks returns all finalized blocks
func (bf *BlockFinalizer) GetFinalizedBlocks() []string {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	blocks := make([]string, 0, len(bf.certificates))
	for blockHash := range bf.certificates {
		blocks = append(blocks, blockHash)
	}
	return blocks
} 