package consensus

import (
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// BlockFinalizer handles block finalization
type BlockFinalizer struct {
	dag          *core.DAG
	certificates map[string]*types.Certificate
	mu           sync.RWMutex
}

// NewBlockFinalizer creates a new block finalizer
func NewBlockFinalizer(dag *core.DAG) *BlockFinalizer {
	return &BlockFinalizer{
		dag:          dag,
		certificates: make(map[string]*types.Certificate),
	}
}

// AddCertificate adds a certificate to the finalizer
func (bf *BlockFinalizer) AddCertificate(cert *types.Certificate) error {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	blockHash := string(cert.BlockHash)
	bf.certificates[blockHash] = cert
	return nil
}

// GetCertificate retrieves a certificate for a block
func (bf *BlockFinalizer) GetCertificate(blockHash string) (*types.Certificate, bool) {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	cert, exists := bf.certificates[blockHash]
	return cert, exists
}

// IsFinalized checks if a block is finalized
func (bf *BlockFinalizer) IsFinalized(blockHash string) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	cert, exists := bf.certificates[blockHash]
	if !exists {
		return false
	}

	// A block is finalized if it has a valid certificate with enough signatures
	requiredSignatures := 2*33 + 1 // 2f+1 where f=33
	return cert != nil && len(cert.Signatures) >= requiredSignatures
}

// GetFinalizedBlocks returns all finalized block hashes
func (bf *BlockFinalizer) GetFinalizedBlocks() []string {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	var finalized []string
	requiredSignatures := 2*33 + 1 // 2f+1 where f=33
	for blockHash, cert := range bf.certificates {
		if cert != nil && len(cert.Signatures) >= requiredSignatures {
			finalized = append(finalized, blockHash)
		}
	}
	return finalized
} 