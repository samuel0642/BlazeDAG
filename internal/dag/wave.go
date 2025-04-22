package dag

import (
	"sync"
	"time"
)

// WaveState represents the state of a wave
type WaveState struct {
	WaveNumber    int
	StartTime     time.Time
	EndTime       time.Time
	Leader        string
	Blocks        map[string]*Block
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

// WaveController controls wave progression
type WaveController struct {
	dag           *DAG
	currentWave   *WaveState
	waveTimeout   time.Duration
	mu            sync.RWMutex
}

// NewWaveController creates a new wave controller
func NewWaveController(dag *DAG, waveTimeout time.Duration) *WaveController {
	return &WaveController{
		dag:         dag,
		waveTimeout: waveTimeout,
	}
}

// StartNewWave starts a new wave
func (wc *WaveController) StartNewWave(leader string) *WaveState {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	waveNumber := 0
	if wc.currentWave != nil {
		waveNumber = wc.currentWave.WaveNumber + 1
	}

	wave := &WaveState{
		WaveNumber: waveNumber,
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(wc.waveTimeout),
		Leader:     leader,
		Blocks:     make(map[string]*Block),
		Status:     WaveStatusInitializing,
	}

	wc.currentWave = wave
	return wave
}

// GetCurrentWave returns the current wave
func (wc *WaveController) GetCurrentWave() *WaveState {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	return wc.currentWave
}

// AddBlock adds a block to the current wave
func (wc *WaveController) AddBlock(block *Block) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if wc.currentWave != nil {
		wc.currentWave.Blocks[block.Hash] = block
		wc.dag.AddBlock(block)
	}
}

// UpdateWaveStatus updates the status of the current wave
func (wc *WaveController) UpdateWaveStatus(status WaveStatus) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if wc.currentWave != nil {
		wc.currentWave.Status = status
	}
}

// FinalizeWave finalizes the current wave
func (wc *WaveController) FinalizeWave() {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if wc.currentWave != nil {
		wc.currentWave.Finalized = true
		wc.currentWave.Status = WaveStatusCompleted
	}
}

// HandleWaveTimeout handles wave timeout
func (wc *WaveController) HandleWaveTimeout() {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if wc.currentWave != nil && !wc.currentWave.Finalized {
		wc.currentWave.Status = WaveStatusFailed
	}
}

// GetWaveProgress returns the progress of the current wave
func (wc *WaveController) GetWaveProgress() float64 {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if wc.currentWave == nil {
		return 0
	}

	elapsed := time.Since(wc.currentWave.StartTime)
	total := wc.currentWave.EndTime.Sub(wc.currentWave.StartTime)
	return float64(elapsed) / float64(total)
}

// GetWaveBlocks returns all blocks in the current wave
func (wc *WaveController) GetWaveBlocks() []*Block {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if wc.currentWave == nil {
		return nil
	}

	blocks := make([]*Block, 0, len(wc.currentWave.Blocks))
	for _, block := range wc.currentWave.Blocks {
		blocks = append(blocks, block)
	}
	return blocks
}

// IsWaveComplete checks if the current wave is complete
func (wc *WaveController) IsWaveComplete() bool {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if wc.currentWave == nil {
		return false
	}

	return wc.currentWave.Status == WaveStatusCompleted || wc.currentWave.Status == WaveStatusFailed
}

// GetWaveLeader returns the leader of the current wave
func (wc *WaveController) GetWaveLeader() string {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if wc.currentWave == nil {
		return ""
	}

	return wc.currentWave.Leader
} 