package dag

import (
	"sync"
	"time"
)

// WaveMetrics represents metrics for a wave
type WaveMetrics struct {
	WaveNumber      int
	BlockCount      int
	AverageLatency  time.Duration
	Throughput      float64
	ResourceUsage   float64
	ConflictCount   int
	LastUpdate      time.Time
}

// WaveOptimizer optimizes wave performance
type WaveOptimizer struct {
	metrics     map[int]*WaveMetrics
	mu          sync.RWMutex
}

// NewWaveOptimizer creates a new wave optimizer
func NewWaveOptimizer() *WaveOptimizer {
	return &WaveOptimizer{
		metrics: make(map[int]*WaveMetrics),
	}
}

// UpdateMetrics updates metrics for a wave
func (wo *WaveOptimizer) UpdateMetrics(waveNumber int, blockCount int, latency time.Duration, resourceUsage float64, conflictCount int) {
	wo.mu.Lock()
	defer wo.mu.Unlock()

	metrics, exists := wo.metrics[waveNumber]
	if !exists {
		metrics = &WaveMetrics{
			WaveNumber: waveNumber,
		}
		wo.metrics[waveNumber] = metrics
	}

	metrics.BlockCount = blockCount
	metrics.AverageLatency = latency
	metrics.ResourceUsage = resourceUsage
	metrics.ConflictCount = conflictCount
	metrics.LastUpdate = time.Now()

	// Calculate throughput (blocks per second)
	if latency > 0 {
		metrics.Throughput = float64(blockCount) / latency.Seconds()
	}
}

// GetMetrics returns metrics for a wave
func (wo *WaveOptimizer) GetMetrics(waveNumber int) *WaveMetrics {
	wo.mu.RLock()
	defer wo.mu.RUnlock()
	return wo.metrics[waveNumber]
}

// GetLatestMetrics returns metrics for the latest wave
func (wo *WaveOptimizer) GetLatestMetrics() *WaveMetrics {
	wo.mu.RLock()
	defer wo.mu.RUnlock()

	var latestWave int
	var latestTime time.Time

	for waveNumber, metrics := range wo.metrics {
		if metrics.LastUpdate.After(latestTime) {
			latestWave = waveNumber
			latestTime = metrics.LastUpdate
		}
	}

	return wo.metrics[latestWave]
}

// OptimizeWave optimizes a wave based on metrics
func (wo *WaveOptimizer) OptimizeWave(waveNumber int) *WaveOptimization {
	wo.mu.RLock()
	defer wo.mu.RUnlock()

	metrics := wo.metrics[waveNumber]
	if metrics == nil {
		return nil
	}

	optimization := &WaveOptimization{
		WaveNumber: waveNumber,
	}

	// Optimize based on latency
	if metrics.AverageLatency > 100*time.Millisecond {
		optimization.SuggestedLatency = metrics.AverageLatency / 2
	}

	// Optimize based on throughput
	if metrics.Throughput < 1000 {
		optimization.SuggestedThroughput = metrics.Throughput * 1.5
	}

	// Optimize based on resource usage
	if metrics.ResourceUsage > 0.8 {
		optimization.SuggestedResourceUsage = 0.6
	}

	// Optimize based on conflicts
	if metrics.ConflictCount > 10 {
		optimization.SuggestedConflictResolution = true
	}

	return optimization
}

// WaveOptimization represents optimization suggestions for a wave
type WaveOptimization struct {
	WaveNumber              int
	SuggestedLatency        time.Duration
	SuggestedThroughput     float64
	SuggestedResourceUsage  float64
	SuggestedConflictResolution bool
}

// ApplyOptimization applies optimization suggestions
func (wo *WaveOptimizer) ApplyOptimization(optimization *WaveOptimization) {
	wo.mu.Lock()
	defer wo.mu.Unlock()

	metrics := wo.metrics[optimization.WaveNumber]
	if metrics == nil {
		return
	}

	// Apply latency optimization
	if optimization.SuggestedLatency > 0 {
		metrics.AverageLatency = optimization.SuggestedLatency
	}

	// Apply throughput optimization
	if optimization.SuggestedThroughput > 0 {
		metrics.Throughput = optimization.SuggestedThroughput
	}

	// Apply resource usage optimization
	if optimization.SuggestedResourceUsage > 0 {
		metrics.ResourceUsage = optimization.SuggestedResourceUsage
	}

	// Apply conflict resolution optimization
	if optimization.SuggestedConflictResolution {
		metrics.ConflictCount = 0
	}

	metrics.LastUpdate = time.Now()
}

// GetOptimizationHistory returns the optimization history
func (wo *WaveOptimizer) GetOptimizationHistory() []*WaveOptimization {
	wo.mu.RLock()
	defer wo.mu.RUnlock()

	history := make([]*WaveOptimization, 0, len(wo.metrics))
	for waveNumber := range wo.metrics {
		optimization := wo.OptimizeWave(waveNumber)
		if optimization != nil {
			history = append(history, optimization)
		}
	}
	return history
} 