package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"
)

// LeaderSelector handles leader selection for waves
type LeaderSelector struct {
	validators     []string
	validatorSet   map[string]bool
	mu             sync.RWMutex
}

// NewLeaderSelector creates a new leader selector
func NewLeaderSelector(validators []string) *LeaderSelector {
	validatorSet := make(map[string]bool)
	for _, v := range validators {
		validatorSet[v] = true
	}

	return &LeaderSelector{
		validators:   validators,
		validatorSet: validatorSet,
	}
}

// SelectLeader selects a leader for the given wave number
func (ls *LeaderSelector) SelectLeader(waveNumber int, committedBlocks []string) string {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	if len(ls.validators) == 0 {
		return ""
	}

	// Use committed blocks as salt for leader selection
	salt := make([]byte, 0)
	for _, block := range committedBlocks {
		salt = append(salt, []byte(block)...)
	}

	// Create a deterministic hash based on wave number and salt
	h := sha256.New()
	binary.Write(h, binary.LittleEndian, uint64(waveNumber))
	h.Write(salt)
	hash := h.Sum(nil)

	// Select leader based on hash
	index := binary.BigEndian.Uint64(hash) % uint64(len(ls.validators))
	return ls.validators[index]
}

// UpdateValidators updates the validator set
func (ls *LeaderSelector) UpdateValidators(validators []string) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	ls.validators = validators
	ls.validatorSet = make(map[string]bool)
	for _, v := range validators {
		ls.validatorSet[v] = true
	}
}

// IsValidator checks if an address is a validator
func (ls *LeaderSelector) IsValidator(address string) bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.validatorSet[address]
}

// GetValidators returns the current validator set
func (ls *LeaderSelector) GetValidators() []string {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.validators
} 