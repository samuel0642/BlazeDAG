package consensus

import (
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// WaveController manages wave transitions
type WaveController struct {
	engine  *ConsensusEngine
	timeout time.Duration
	timer   *time.Timer
	mu      sync.RWMutex
}

// NewWaveController creates a new wave controller
func NewWaveController(engine *ConsensusEngine, timeout time.Duration) *WaveController {
	return &WaveController{
		engine:  engine,
		timeout: timeout,
	}
}

// Start starts the wave controller
func (wc *WaveController) Start() {
	wc.timer = time.NewTimer(wc.timeout)
	go wc.run()
}

// Stop stops the wave controller
func (wc *WaveController) Stop() {
	if wc.timer != nil {
		wc.timer.Stop()
	}
}

// run runs the wave controller loop
func (wc *WaveController) run() {
	for {
		select {
		case <-wc.timer.C:
			wc.ProcessTimeout()
			wc.timer.Reset(wc.timeout)
		}
	}
}

// ProcessTimeout handles wave timeout
func (wc *WaveController) ProcessTimeout() {
	wc.engine.ProcessTimeout()
}

// GetCurrentWave returns the current wave
func (wc *WaveController) GetCurrentWave() types.Wave {
	return wc.engine.GetCurrentWave()
}

// handleWaveProposing handles the proposing phase of a wave
func (wc *WaveController) handleWaveProposing() error {
	// Check if we are the leader
	if !wc.engine.IsLeader() {
		return nil
	}

	// Create block
	block, err := wc.engine.CreateBlock()
	if err != nil {
		return err
	}

	// Broadcast block
	return wc.engine.BroadcastBlock(block)
} 