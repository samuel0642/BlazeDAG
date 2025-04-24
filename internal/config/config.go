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

	return &config, nil
} 