package consensus

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
	"gopkg.in/yaml.v2"
)

// Config represents the consensus configuration
type Config struct {
	NodeID         string
	WaveTimeout    time.Duration
	RoundDuration  time.Duration
	ValidatorSet   []types.Address
	QuorumSize     int
	ListenAddr     types.Address
	Seeds          []types.Address
	TotalValidators int
}

// LoadConfig loads the consensus configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config struct {
		NodeID string `yaml:"node_id"`
		Consensus struct {
			WaveTimeout    string   `yaml:"wave_timeout"`
			RoundDuration  string   `yaml:"round_duration"`
			ValidatorSet   []string `yaml:"validator_set"`
			QuorumSize     int      `yaml:"quorum_size"`
			ListenAddr     string   `yaml:"listen_addr"`
			Seeds          []string `yaml:"seeds"`
			TotalValidators int     `yaml:"total_validators"`
		} `yaml:"consensus"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Parse durations
	waveTimeout, err := time.ParseDuration(config.Consensus.WaveTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid wave timeout: %v", err)
	}

	roundDuration, err := time.ParseDuration(config.Consensus.RoundDuration)
	if err != nil {
		return nil, fmt.Errorf("invalid round duration: %v", err)
	}

	// Convert validator set to Address type
	validatorSet := make([]types.Address, len(config.Consensus.ValidatorSet))
	for i, v := range config.Consensus.ValidatorSet {
		validatorSet[i] = types.Address(v)
	}

	// Convert seeds to Address type
	seeds := make([]types.Address, len(config.Consensus.Seeds))
	for i, s := range config.Consensus.Seeds {
		seeds[i] = types.Address(s)
	}

	// Validate required fields
	if config.NodeID == "" {
		return nil, fmt.Errorf("node_id is required")
	}
	if config.Consensus.ListenAddr == "" {
		return nil, fmt.Errorf("listen_addr is required")
	}

	return &Config{
		NodeID:         config.NodeID,
		WaveTimeout:    waveTimeout,
		RoundDuration:  roundDuration,
		ValidatorSet:   validatorSet,
		QuorumSize:     config.Consensus.QuorumSize,
		ListenAddr:     types.Address(config.Consensus.ListenAddr),
		Seeds:          seeds,
		TotalValidators: config.Consensus.TotalValidators,
	}, nil
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