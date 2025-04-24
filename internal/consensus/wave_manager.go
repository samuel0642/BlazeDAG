package consensus

import (
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// WaveManager manages wave transitions
type WaveManager struct {
	engine  *ConsensusEngine
	timeout time.Duration
	timer   *time.Timer
	mu      sync.RWMutex
}

// NewWaveManager creates a new wave manager
func NewWaveManager(config *Config) *WaveManager {
	return &WaveManager{
		timeout: config.WaveTimeout,
	}
}

// Start starts the wave manager
func (wm *WaveManager) Start() {
	wm.timer = time.NewTimer(wm.timeout)
	go wm.run()
}

// Stop stops the wave manager
func (wm *WaveManager) Stop() {
	if wm.timer != nil {
		wm.timer.Stop()
	}
}

// run runs the wave manager loop
func (wm *WaveManager) run() {
	for {
		select {
		case <-wm.timer.C:
			wm.ProcessTimeout()
			wm.timer.Reset(wm.timeout)
		}
	}
}

// ProcessTimeout handles wave timeout
func (wm *WaveManager) ProcessTimeout() {
	wm.engine.ProcessTimeout()
}

// GetCurrentWave returns the current wave
func (wm *WaveManager) GetCurrentWave() types.Wave {
	return wm.engine.GetCurrentWave()
}

// StartNewWave starts a new wave
func (wm *WaveManager) StartNewWave() (*WaveState, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wave := NewWaveState(wm.engine.GetCurrentWave(), wm.timeout, wm.engine.config.QuorumSize)
	return wave, nil
}

// FinalizeWave finalizes the current wave
func (wm *WaveManager) FinalizeWave() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Nothing to do here, the engine handles the wave status
}

// handleWaveProposing handles the proposing phase of a wave
func (wm *WaveManager) handleWaveProposing() error {
	// Check if we are the leader
	if !wm.engine.IsLeader() {
		return nil
	}

	// Create block
	block, err := wm.engine.CreateBlock()
	if err != nil {
		return err
	}

	// Broadcast block
	return wm.engine.BroadcastBlock(block)
} 