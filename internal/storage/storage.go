package storage

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// Storage represents the persistent storage for chain data
type Storage struct {
	mu sync.RWMutex
	baseDir string
}

// NewStorage creates a new storage instance
func NewStorage(baseDir string) (*Storage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &Storage{
		baseDir: baseDir,
	}, nil
}

// SaveBlock saves a block to storage
func (s *Storage) SaveBlock(block *types.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(block)
	if err != nil {
		return err
	}

	// Encode block hash as hex for filesystem safety
	blockHash := hex.EncodeToString(block.ComputeHash())
	
	// Create validator-specific directory
	validatorDir := filepath.Join(s.baseDir, "blocks", string(block.Header.Validator))
	if err := os.MkdirAll(validatorDir, 0755); err != nil {
		return fmt.Errorf("failed to create validator directory: %v", err)
	}

	// Save block in validator's directory
	blockFile := filepath.Join(validatorDir, blockHash)
	if err := os.WriteFile(blockFile, data, 0644); err != nil {
		return fmt.Errorf("failed to save block: %v", err)
	}

	return nil
}

// LoadBlock loads a block from storage
func (s *Storage) LoadBlock(hash types.Hash) (*types.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Encode block hash as hex for filesystem safety
	blockHash := hex.EncodeToString(hash)
	
	// Search in all validator directories
	blocksDir := filepath.Join(s.baseDir, "blocks")
	validators, err := os.ReadDir(blocksDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read blocks directory: %v", err)
	}

	for _, validator := range validators {
		if !validator.IsDir() {
			continue
		}
		
		blockFile := filepath.Join(blocksDir, validator.Name(), blockHash)
		data, err := os.ReadFile(blockFile)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read block file: %v", err)
		}

		var block types.Block
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block: %v", err)
		}

		return &block, nil
	}

	return nil, fmt.Errorf("block not found")
}

// SaveMempool saves the mempool to storage
func (s *Storage) SaveMempool(txs []*types.Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(txs)
	if err != nil {
		return err
	}

	mempoolFile := filepath.Join(s.baseDir, "mempool.json")
	return os.WriteFile(mempoolFile, data, 0644)
}

// LoadMempool loads the mempool from storage
func (s *Storage) LoadMempool() ([]*types.Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mempoolFile := filepath.Join(s.baseDir, "mempool.json")
	data, err := os.ReadFile(mempoolFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []*types.Transaction{}, nil
		}
		return nil, err
	}

	var txs []*types.Transaction
	if err := json.Unmarshal(data, &txs); err != nil {
		return nil, err
	}

	return txs, nil
}

// SaveState saves the engine state to storage
func (s *Storage) SaveState(state *types.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	stateFile := filepath.Join(s.baseDir, "engine_state.json")
	return os.WriteFile(stateFile, data, 0644)
}

// LoadState loads the engine state from storage
func (s *Storage) LoadState() (*types.State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateFile := filepath.Join(s.baseDir, "engine_state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return types.NewState(), nil
		}
		return nil, err
	}

	var state types.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// GetLatestBlocks returns the n latest blocks from storage
func (s *Storage) GetLatestBlocks(n int) ([]*types.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	blocksDir := filepath.Join(s.baseDir, "blocks")
	validators, err := os.ReadDir(blocksDir)
	if err != nil {
		return nil, err
	}

	// Get all blocks from all validators
	allBlocks := make([]*types.Block, 0)
	for _, validator := range validators {
		if !validator.IsDir() {
			continue
		}

		validatorDir := filepath.Join(blocksDir, validator.Name())
		files, err := os.ReadDir(validatorDir)
		if err != nil {
			continue
		}

		// Get blocks from this validator
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			data, err := os.ReadFile(filepath.Join(validatorDir, file.Name()))
			if err != nil {
				continue
			}

			var block types.Block
			if err := json.Unmarshal(data, &block); err != nil {
				continue
			}

			allBlocks = append(allBlocks, &block)
		}
	}

	// Sort all blocks by height, wave, and round in descending order
	sort.Slice(allBlocks, func(i, j int) bool {
		if allBlocks[i].Header.Height != allBlocks[j].Header.Height {
			return allBlocks[i].Header.Height > allBlocks[j].Header.Height
		}
		if allBlocks[i].Header.Wave != allBlocks[j].Header.Wave {
			return allBlocks[i].Header.Wave > allBlocks[j].Header.Wave
		}
		if allBlocks[i].Header.Round != allBlocks[j].Header.Round {
			return allBlocks[i].Header.Round > allBlocks[j].Header.Round
		}
		// If all else is equal, sort by validator to ensure consistent ordering
		return allBlocks[i].Header.Validator > allBlocks[j].Header.Validator
	})

	// Return the n latest blocks
	if n > len(allBlocks) {
		n = len(allBlocks)
	}
	return allBlocks[:n], nil
} 