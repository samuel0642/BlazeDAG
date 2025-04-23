package consensus

import (
	"fmt"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

var (
	ErrNoActiveWave = fmt.Errorf("no active wave")
)

// WaveManager manages consensus waves
type WaveManager struct {
	config     *Config
	waveNumber types.Wave
	mu         sync.RWMutex
}

// NewWaveManager creates a new wave manager
func NewWaveManager(config *Config) *WaveManager {
	return &WaveManager{
		config:     config,
		waveNumber: 0,
	}
}

// StartNewWave starts a new wave
func (wm *WaveManager) StartNewWave() (*Wave, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.waveNumber++
	wave := &Wave{
		Number:    wm.waveNumber,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(wm.config.WaveTimeout),
		Status:    types.WaveStatusProposing,
		Votes:     make(map[types.Address]bool),
	}

	return wave, nil
}

// GetWaveNumber returns the current wave number
func (wm *WaveManager) GetWaveNumber() types.Wave {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.waveNumber
}

// IsWaveTimedOut checks if the current wave has timed out
func (wm *WaveManager) IsWaveTimedOut(wave *Wave) bool {
	return time.Now().After(wave.EndTime)
}

// ResetWaveNumber resets the wave number to 0
func (wm *WaveManager) ResetWaveNumber() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.waveNumber = 0
}

// HandleWaveTimeout handles wave timeout
func (wm *WaveManager) HandleWaveTimeout() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	// Nothing to do here, the engine handles the wave status
}

// FinalizeWave finalizes the current wave
func (wm *WaveManager) FinalizeWave() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	// Nothing to do here, the engine handles the wave status
} 