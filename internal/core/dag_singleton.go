package core

import (
	"sync"
)

var (
	// Global DAG instance
	globalDAG *DAG
	// Mutex to protect singleton initialization
	dagMu sync.Mutex
)

// GetDAG returns the singleton instance of the DAG
func GetDAG() *DAG {
	dagMu.Lock()
	defer dagMu.Unlock()

	if globalDAG == nil {
		globalDAG = NewDAG()
	}

	return globalDAG
}

// SetGlobalDAG explicitly sets the global DAG instance
// This is useful for testing or when initialization needs to be controlled
func SetGlobalDAG(dag *DAG) {
	dagMu.Lock()
	defer dagMu.Unlock()
	globalDAG = dag
}
