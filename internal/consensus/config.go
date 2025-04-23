package consensus

import (
	"fmt"
	"time"
)

// Config stores consensus configuration
type Config struct {
	// WaveTimeout is the maximum duration for a wave
	WaveTimeout time.Duration

	// MinValidators is the minimum number of validators required
	MinValidators int

	// TotalValidators is the total number of validators
	TotalValidators int

	// QuorumSize is the minimum number of votes required for consensus
	QuorumSize int

	// BlockTime is the target time between blocks
	BlockTime time.Duration

	// MaxBlockSize is the maximum size of a block in bytes
	MaxBlockSize uint64

	// MaxTransactionsPerBlock is the maximum number of transactions per block
	MaxTransactionsPerBlock uint64

	// FaultTolerance is the number of faulty nodes that can be tolerated
	FaultTolerance int
}

// DefaultConfig returns the default consensus configuration
func DefaultConfig() *Config {
	return &Config{
		WaveTimeout:            30 * time.Second,
		MinValidators:         4,
		TotalValidators:      10,
		QuorumSize:           7,
		BlockTime:            2 * time.Second,
		MaxBlockSize:         1024 * 1024, // 1MB
		MaxTransactionsPerBlock: 1000,
		FaultTolerance:       3,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.WaveTimeout <= 0 {
		return fmt.Errorf("wave timeout must be positive")
	}
	if c.MinValidators < 1 {
		return fmt.Errorf("minimum validators must be at least 1")
	}
	if c.TotalValidators < c.MinValidators {
		return fmt.Errorf("total validators must be at least minimum validators")
	}
	if c.QuorumSize < 1 || c.QuorumSize > c.TotalValidators {
		return fmt.Errorf("quorum size must be between 1 and total validators")
	}
	if c.BlockTime <= 0 {
		return fmt.Errorf("block time must be positive")
	}
	if c.MaxBlockSize == 0 {
		return fmt.Errorf("max block size must be positive")
	}
	if c.MaxTransactionsPerBlock == 0 {
		return fmt.Errorf("max transactions per block must be positive")
	}
	if c.FaultTolerance < 0 || c.FaultTolerance > c.TotalValidators/3 {
		return fmt.Errorf("fault tolerance must be between 0 and total validators/3")
	}
	return nil
} 