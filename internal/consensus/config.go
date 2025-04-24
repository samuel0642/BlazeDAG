package consensus

import (
	"fmt"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// Config represents the consensus configuration
type Config struct {
	// WaveTimeout is the maximum duration for a wave
	WaveTimeout time.Duration

	// RoundDuration is the duration of a round
	RoundDuration time.Duration

	// ValidatorSet is the set of validators
	ValidatorSet []types.Address

	// QuorumSize is the minimum number of votes required for consensus
	QuorumSize int

	// ListenAddr is the address to listen for incoming connections
	ListenAddr types.Address

	// Seeds are the addresses of seed nodes
	Seeds []types.Address

	// TotalValidators is the total number of validators
	TotalValidators int
}

// NewConfig creates a new consensus configuration with default values
func NewConfig() *Config {
	return &Config{
		WaveTimeout:     30 * time.Second,
		RoundDuration:   2 * time.Second,
		ValidatorSet:    make([]types.Address, 0),
		QuorumSize:      7,
		ListenAddr:      types.Address(""),
		Seeds:           make([]types.Address, 0),
		TotalValidators: 10,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.WaveTimeout <= 0 {
		return fmt.Errorf("wave timeout must be positive")
	}
	if c.RoundDuration <= 0 {
		return fmt.Errorf("round duration must be positive")
	}
	if len(c.ValidatorSet) == 0 {
		return fmt.Errorf("validator set cannot be empty")
	}
	if c.QuorumSize <= 0 || c.QuorumSize > len(c.ValidatorSet) {
		return fmt.Errorf("invalid quorum size")
	}
	if c.ListenAddr == "" {
		return fmt.Errorf("listen address cannot be empty")
	}
	if c.TotalValidators <= 0 {
		return fmt.Errorf("total validators must be positive")
	}
	return nil
} 