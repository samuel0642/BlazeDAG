package storage

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/state"
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
	blockFile := filepath.Join(s.baseDir, "blocks", blockHash)
	if err := os.MkdirAll(filepath.Dir(blockFile), 0755); err != nil {
		return err
	}

	return os.WriteFile(blockFile, data, 0644)
}

// LoadBlock loads a block from storage
func (s *Storage) LoadBlock(hash types.Hash) (*types.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Encode block hash as hex for filesystem safety
	blockHash := hex.EncodeToString(hash)
	blockFile := filepath.Join(s.baseDir, "blocks", blockHash)
	data, err := os.ReadFile(blockFile)
	if err != nil {
		return nil, err
	}

	var block types.Block
	if err := json.Unmarshal(data, &block); err != nil {
		return nil, err
	}

	return &block, nil
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
func (s *Storage) SaveState(state *core.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a map to store the state data
	stateData := map[string]interface{}{
		"latest_block":     state.LatestBlock,
		"pending_blocks":   state.PendingBlocks,
		"finalized_blocks": state.FinalizedBlocks,
		"current_wave":     state.CurrentWave,
		"current_round":    state.CurrentRound,
		"active_proposals": state.ActiveProposals,
		"votes":           state.Votes,
		"connected_peers": state.ConnectedPeers,
	}

	// Ensure current_round is always set
	if state.CurrentRound == 0 {
		state.CurrentRound = 1
	}

	data, err := json.Marshal(stateData)
	if err != nil {
		return err
	}

	stateFile := filepath.Join(s.baseDir, "engine_state.json")
	return os.WriteFile(stateFile, data, 0644)
}

// LoadState loads the engine state from storage
func (s *Storage) LoadState() (*core.State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateFile := filepath.Join(s.baseDir, "engine_state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize with default values
			state := core.NewState()
			state.CurrentWave = 1
			state.CurrentRound = 1
			return state, nil
		}
		return nil, err
	}

	var stateData map[string]interface{}
	if err := json.Unmarshal(data, &stateData); err != nil {
		return nil, err
	}

	state := core.NewState()

	// Restore state data
	if latestBlock, ok := stateData["latest_block"].(map[string]interface{}); ok {
		blockData, err := json.Marshal(latestBlock)
		if err == nil {
			json.Unmarshal(blockData, &state.LatestBlock)
		}
	}

	if pendingBlocks, ok := stateData["pending_blocks"].(map[string]interface{}); ok {
		state.PendingBlocks = make(map[string]*types.Block)
		for k, v := range pendingBlocks {
			if blockData, ok := v.(map[string]interface{}); ok {
				blockBytes, err := json.Marshal(blockData)
				if err == nil {
					var block types.Block
					if err := json.Unmarshal(blockBytes, &block); err == nil {
						state.PendingBlocks[k] = &block
					}
				}
			}
		}
	}

	if finalizedBlocks, ok := stateData["finalized_blocks"].(map[string]interface{}); ok {
		state.FinalizedBlocks = make(map[string]*types.Block)
		for k, v := range finalizedBlocks {
			if blockData, ok := v.(map[string]interface{}); ok {
				blockBytes, err := json.Marshal(blockData)
				if err == nil {
					var block types.Block
					if err := json.Unmarshal(blockBytes, &block); err == nil {
						state.FinalizedBlocks[k] = &block
					}
				}
			}
		}
	}

	if currentWave, ok := stateData["current_wave"].(float64); ok {
		state.CurrentWave = uint64(currentWave)
	}

	if currentRound, ok := stateData["current_round"].(float64); ok {
		state.CurrentRound = uint64(currentRound)
	} else {
		// If current_round is not found, set it to 1
		state.CurrentRound = 1
	}

	if activeProposals, ok := stateData["active_proposals"].(map[string]interface{}); ok {
		state.ActiveProposals = make(map[string]*types.Proposal)
		for k, v := range activeProposals {
			if proposalData, ok := v.(map[string]interface{}); ok {
				proposalBytes, err := json.Marshal(proposalData)
				if err == nil {
					var proposal types.Proposal
					if err := json.Unmarshal(proposalBytes, &proposal); err == nil {
						state.ActiveProposals[k] = &proposal
					}
				}
			}
		}
	}

	if votes, ok := stateData["votes"].(map[string]interface{}); ok {
		state.Votes = make(map[string][]*types.Vote)
		for k, v := range votes {
			if voteList, ok := v.([]interface{}); ok {
				state.Votes[k] = make([]*types.Vote, 0, len(voteList))
				for _, voteData := range voteList {
					if voteMap, ok := voteData.(map[string]interface{}); ok {
						voteBytes, err := json.Marshal(voteMap)
						if err == nil {
							var vote types.Vote
							if err := json.Unmarshal(voteBytes, &vote); err == nil {
								state.Votes[k] = append(state.Votes[k], &vote)
							}
						}
					}
				}
			}
		}
	}

	if connectedPeers, ok := stateData["connected_peers"].(map[string]interface{}); ok {
		state.ConnectedPeers = make(map[types.Address]*types.Peer)
		for k, v := range connectedPeers {
			if peerData, ok := v.(map[string]interface{}); ok {
				peerBytes, err := json.Marshal(peerData)
				if err == nil {
					var peer types.Peer
					if err := json.Unmarshal(peerBytes, &peer); err == nil {
						state.ConnectedPeers[types.Address(k)] = &peer
					}
				}
			}
		}
	}

	return state, nil
}

// SaveAccountState saves the account state to storage
func (s *Storage) SaveAccountState(state *state.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	stateFile := filepath.Join(s.baseDir, "account_state.json")
	return os.WriteFile(stateFile, data, 0644)
}

// LoadAccountState loads the account state from storage
func (s *Storage) LoadAccountState() (*state.State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateFile := filepath.Join(s.baseDir, "account_state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return state.NewState(), nil
		}
		return nil, err
	}

	var state state.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
} 