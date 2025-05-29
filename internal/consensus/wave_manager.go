package consensus

import (
	"sync"
	"time"

	// "fmt"
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
func NewWaveManager(engine *ConsensusEngine) *WaveManager {
	return &WaveManager{
		engine:  engine,
		timeout: engine.config.WaveTimeout,
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
	lastBlockCreationWave := types.Wave(0)

	for {
		select {
		case <-wm.timer.C:
			wm.ProcessTimeout()
			wm.timer.Reset(wm.timeout)
		default:
			// Only create blocks when we are the leader and haven't created one for this wave
			if wm.engine.IsLeader() {
				currentWave := wm.engine.GetCurrentWave()
				if currentWave > lastBlockCreationWave {
					wm.engine.logger.Printf("🏗️ Wave manager triggering block creation for wave %d (leader: %s)",
						currentWave, wm.engine.GetNodeID())

					if err := wm.handleWaveProposing(); err != nil {
						wm.engine.logger.Printf("❌ Error handling wave proposing: %v", err)
					} else {
						lastBlockCreationWave = currentWave
						wm.engine.logger.Printf("✅ Wave manager completed block creation for wave %d", currentWave)
					}
				}
			}
			time.Sleep(1000 * time.Millisecond) // Check every second instead of 10 seconds
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
	wm.engine.logger.Printf("🏗️ Wave manager creating block...")
	block, err := wm.engine.CreateBlock()
	if err != nil {
		wm.engine.logger.Printf("❌ Wave manager failed to create block: %v", err)
		return err
	}

	if block == nil {
		wm.engine.logger.Printf("⚠️ Wave manager: CreateBlock returned nil, skipping broadcast")
		return nil
	}

	wm.engine.logger.Printf("🏗️ Wave manager created block %x, now broadcasting...", block.ComputeHash())

	// Broadcast block
	return wm.engine.BroadcastBlock(block)
}
