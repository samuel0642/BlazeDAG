package consensus

import (
	"sync"
	"time"
)

// WaveState represents the current state of a wave
type WaveState struct {
	WaveNumber    int
	StartTime     time.Time
	EndTime       time.Time
	Leader        string
	Proposals     map[string]*Proposal
	Votes         map[string][]*Vote
	Status        WaveStatus
	Finalized     bool
	mu            sync.RWMutex
}

// WaveStatus represents the status of a wave
type WaveStatus int

const (
	WaveStatusInitializing WaveStatus = iota
	WaveStatusActive
	WaveStatusCompleting
	WaveStatusCompleted
	WaveStatusFailed
)

// WaveManager manages the wave-based consensus
type WaveManager struct {
	currentWave *WaveState
	config      Config
	mu          sync.RWMutex
}

// NewWaveManager creates a new wave manager
func NewWaveManager(config Config) *WaveManager {
	return &WaveManager{
		config: config,
	}
}

// StartNewWave starts a new wave
func (wm *WaveManager) StartNewWave() *WaveState {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	waveNumber := 0
	if wm.currentWave != nil {
		waveNumber = wm.currentWave.WaveNumber + 1
	}

	wave := &WaveState{
		WaveNumber: waveNumber,
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(wm.config.WaveTimeout),
		Proposals:  make(map[string]*Proposal),
		Votes:      make(map[string][]*Vote),
		Status:     WaveStatusInitializing,
	}

	wm.currentWave = wave
	return wave
}

// GetCurrentWave returns the current wave
func (wm *WaveManager) GetCurrentWave() *WaveState {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.currentWave
}

// UpdateWaveStatus updates the status of the current wave
func (wm *WaveManager) UpdateWaveStatus(status WaveStatus) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	if wm.currentWave != nil {
		wm.currentWave.Status = status
	}
}

// FinalizeWave finalizes the current wave
func (wm *WaveManager) FinalizeWave() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	if wm.currentWave != nil {
		wm.currentWave.Finalized = true
		wm.currentWave.Status = WaveStatusCompleted
	}
}

// HandleWaveTimeout handles wave timeout
func (wm *WaveManager) HandleWaveTimeout() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	if wm.currentWave != nil && !wm.currentWave.Finalized {
		wm.currentWave.Status = WaveStatusFailed
	}
}

// GetWaveProgress returns the progress of the current wave
func (wm *WaveManager) GetWaveProgress() float64 {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	if wm.currentWave == nil {
		return 0
	}

	elapsed := time.Since(wm.currentWave.StartTime)
	total := wm.currentWave.EndTime.Sub(wm.currentWave.StartTime)
	return float64(elapsed) / float64(total)
} 