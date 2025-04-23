package consensus

import (
	"fmt"
	"sync"
	"time"

	// "github.com/CrossDAG/BlazeDAG/internal/types"
)

// WaveController manages the wave lifecycle
type WaveController struct {
	engine      *ConsensusEngine
	running     bool
	waveTimeout time.Duration
	mu          sync.RWMutex
}

// NewWaveController creates a new wave controller
func NewWaveController(engine *ConsensusEngine) *WaveController {
	return &WaveController{
		engine:      engine,
		waveTimeout: engine.config.WaveTimeout,
	}
}

// Start starts the wave controller
func (wc *WaveController) Start() error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if wc.running {
		return nil
	}

	wc.running = true
	go wc.run()
	return nil
}

// Stop stops the wave controller
func (wc *WaveController) Stop() error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if !wc.running {
		return nil
	}

	wc.running = false
	return nil
}

// run is the main loop of the wave controller
func (wc *WaveController) run() {
	ticker := time.NewTicker(wc.waveTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !wc.running {
				return
			}

			// Start new wave
			if err := wc.engine.StartNewWave(); err != nil {
				fmt.Printf("Failed to start new wave: %v\n", err)
				continue
			}

			// Select leader
			wc.engine.SelectLeader()

			// If we are the leader, create a block
			if wc.engine.IsLeader() {
				wave := wc.engine.GetCurrentWave()
				if wave == nil {
					continue
				}

				block, err := wc.engine.CreateBlock(wave.Number, 0)
				if err != nil {
					fmt.Printf("Failed to create block: %v\n", err)
					continue
				}

				if err := wc.engine.HandleBlock(block); err != nil {
					fmt.Printf("Failed to handle block: %v\n", err)
					continue
				}
			}
		}
	}
}

// GetCurrentWave returns the current wave
func (wc *WaveController) GetCurrentWave() *Wave {
	return wc.engine.GetCurrentWave()
}

// IsLeader checks if this node is the current wave leader
func (wc *WaveController) IsLeader() bool {
	return wc.engine.IsLeader()
}

// HandleTimeout handles wave timeout
func (wc *WaveController) HandleTimeout() {
	wc.engine.HandleWaveTimeout()
} 