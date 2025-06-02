package config

import (
	"fmt"
	"os"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	NodeID    types.Address `yaml:"node_id"`
	Consensus ConsensusConfig `yaml:"consensus"`
	Block     BlockConfig `yaml:"block"`
}

// ConsensusConfig represents the consensus configuration
type ConsensusConfig struct {
	WaveTimeout   time.Duration `yaml:"wave_timeout"`
	RoundDuration time.Duration `yaml:"round_duration"`
	ValidatorSet  []types.Address `yaml:"validator_set"`
	QuorumSize    int `yaml:"quorum_size"`
	ListenAddr    types.Address `yaml:"listen_addr"`
	Seeds         []types.Address `yaml:"seeds"`
}

// BlockConfig represents the block configuration
type BlockConfig struct {
	MaxTransactionsPerBlock uint64 `yaml:"max_transactions_per_block"`
	MaxBlockSize            uint64 `yaml:"max_block_size"`              // in bytes
	TransactionTimeoutMs    uint64 `yaml:"transaction_timeout_ms"`      // timeout for transaction processing
	MemPoolSize             uint64 `yaml:"mempool_size"`               // maximum transactions in mempool
	BatchSize               uint64 `yaml:"batch_size"`                 // batch size for parallel processing
	WorkerCount             int    `yaml:"worker_count"`               // number of worker threads
	EnableOptimizations     bool   `yaml:"enable_optimizations"`       // enable performance optimizations
}

// DefaultBlockConfig returns default block configuration optimized for 40K transactions
func DefaultBlockConfig() BlockConfig {
	return BlockConfig{
		MaxTransactionsPerBlock: 40000,
		MaxBlockSize:            100 * 1024 * 1024, // 100MB
		TransactionTimeoutMs:    5000,               // 5 seconds
		MemPoolSize:             200000,             // 200K transactions
		BatchSize:               1000,               // process 1000 transactions per batch
		WorkerCount:             16,                 // 16 worker threads
		EnableOptimizations:     true,
	}
}

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Set default block config if not provided
	if config.Block.MaxTransactionsPerBlock == 0 {
		config.Block = DefaultBlockConfig()
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Block.MaxTransactionsPerBlock == 0 {
		return fmt.Errorf("max_transactions_per_block must be greater than 0")
	}
	if c.Block.MaxBlockSize == 0 {
		return fmt.Errorf("max_block_size must be greater than 0")
	}
	if c.Block.WorkerCount <= 0 {
		return fmt.Errorf("worker_count must be greater than 0")
	}
	return nil
} 