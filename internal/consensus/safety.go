package consensus

import (
	"sync"
	"time"
)

// Complaint represents a validator's complaint
type Complaint struct {
	BlockHash  string
	Validator  string
	Reason     string
	Timestamp  time.Time
	Signature  []byte
}

// SafetySystem handles safety mechanisms
type SafetySystem struct {
	complaints    map[string][]*Complaint
	timeouts      map[string]time.Time
	recoveryState map[string]bool
	mu            sync.RWMutex
}

// NewSafetySystem creates a new safety system
func NewSafetySystem() *SafetySystem {
	return &SafetySystem{
		complaints:    make(map[string][]*Complaint),
		timeouts:      make(map[string]time.Time),
		recoveryState: make(map[string]bool),
	}
}

// AddComplaint adds a new complaint
func (ss *SafetySystem) AddComplaint(complaint *Complaint) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.complaints[complaint.BlockHash] = append(ss.complaints[complaint.BlockHash], complaint)
}

// GetComplaints returns all complaints for a block
func (ss *SafetySystem) GetComplaints(blockHash string) []*Complaint {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.complaints[blockHash]
}

// SetTimeout sets a timeout for a block
func (ss *SafetySystem) SetTimeout(blockHash string, timeout time.Time) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.timeouts[blockHash] = timeout
}

// CheckTimeout checks if a block has timed out
func (ss *SafetySystem) CheckTimeout(blockHash string) bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	timeout, exists := ss.timeouts[blockHash]
	if !exists {
		return false
	}

	return time.Now().After(timeout)
}

// StartRecovery starts recovery for a block
func (ss *SafetySystem) StartRecovery(blockHash string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.recoveryState[blockHash] = true
}

// IsInRecovery checks if a block is in recovery
func (ss *SafetySystem) IsInRecovery(blockHash string) bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.recoveryState[blockHash]
}

// CompleteRecovery completes recovery for a block
func (ss *SafetySystem) CompleteRecovery(blockHash string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.recoveryState, blockHash)
}

// GetRecoveryBlocks returns all blocks in recovery
func (ss *SafetySystem) GetRecoveryBlocks() []string {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	blocks := make([]string, 0, len(ss.recoveryState))
	for blockHash := range ss.recoveryState {
		blocks = append(blocks, blockHash)
	}
	return blocks
}

// HasFaultyBehavior checks if a validator has exhibited faulty behavior
func (ss *SafetySystem) HasFaultyBehavior(validator string, faultTolerance int) bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	complaintCount := 0
	for _, complaints := range ss.complaints {
		for _, complaint := range complaints {
			if complaint.Validator == validator {
				complaintCount++
			}
		}
	}

	return complaintCount > faultTolerance
} 