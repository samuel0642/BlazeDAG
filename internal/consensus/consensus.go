package consensus

import (
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/storage"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// Consensus represents the consensus engine
type Consensus struct {
	mu sync.RWMutex

	config  *Config
	state   *types.State
	storage *storage.Storage

	blockCreator *core.BlockCreator
	validatorSet *ValidatorSet

	// Channels for communication
	proposalChan chan *types.Block
	voteChan     chan *types.Vote
	commitChan   chan *types.Block

	// State
	currentWave  types.Wave
	currentRound types.Round
	height       types.BlockNumber
}

// ValidatorSet represents a set of validators
type ValidatorSet struct {
	validators []types.Address
}

// NewValidatorSet creates a new validator set
func NewValidatorSet(validators []types.Address) *ValidatorSet {
	return &ValidatorSet{
		validators: validators,
	}
}

// GetLeader returns the leader for a given wave and round
func (vs *ValidatorSet) GetLeader(wave types.Wave, round types.Round) types.Address {
	// Simple round-robin leader selection
	index := (uint64(wave) + uint64(round)) % uint64(len(vs.validators))
	return vs.validators[index]
}

// QuorumSize returns the required quorum size
func (vs *ValidatorSet) QuorumSize() int {
	// For testing, require 2/3 of validators
	return (len(vs.validators) * 2) / 3
}

// NewConsensus creates a new consensus engine
func NewConsensus(config *Config, storage *storage.Storage) (*Consensus, error) {
	// Load state from storage
	state, err := storage.LoadState()
	if err != nil {
		// Initialize new state if none exists
		state = types.NewState()
	}

	// Create validator set
	validatorSet := NewValidatorSet(config.ValidatorSet)

	// Create block creator
	blockCreator := core.NewBlockCreator(&core.Config{
		NodeID: types.Address(config.NodeID),
	}, state, storage)

	return &Consensus{
		config:       config,
		state:        state,
		storage:      storage,
		blockCreator: blockCreator,
		validatorSet: validatorSet,
		proposalChan: make(chan *types.Block, 100),
		voteChan:     make(chan *types.Vote, 100),
		commitChan:   make(chan *types.Block, 100),
		currentWave:  types.Wave(state.CurrentWave),
		currentRound: types.Round(state.CurrentRound),
		height:       types.BlockNumber(state.Height),
	}, nil
}

// Start starts the consensus engine
func (c *Consensus) Start() error {
	// Start goroutines for handling proposals, votes, and commits
	go c.handleProposals()
	go c.handleVotes()
	go c.handleCommits()

	// Start the consensus loop
	go c.consensusLoop()

	return nil
}

// consensusLoop runs the main consensus loop
func (c *Consensus) consensusLoop() {
	for {
		// Check if we are the leader for this round
		if c.isLeader() {
			// Create and propose a new block
			block, err := c.blockCreator.CreateBlock()
			if err != nil {
				continue
			}

			// Broadcast the block proposal
			c.broadcastProposal(block)
		}

		// Wait for the round duration
		time.Sleep(c.config.RoundDuration)
	}
}

// handleProposals handles incoming block proposals
func (c *Consensus) handleProposals() {
	for block := range c.proposalChan {
		// Validate the block
		if !c.validateBlock(block) {
			continue
		}

		// Save the block to storage
		if err := c.storage.SaveBlock(block); err != nil {
			continue
		}

		// Create and broadcast a vote for the block
		vote := &types.Vote{
			BlockHash: block.ComputeHash(),
			Round:     block.Header.Round,
			Wave:      block.Header.Wave,
			Validator: types.Address(c.config.NodeID),
		}
		c.broadcastVote(vote)
	}
}

// handleVotes handles incoming votes
func (c *Consensus) handleVotes() {
	votes := make(map[string][]*types.Vote)

	for vote := range c.voteChan {
		// Add vote to the vote map
		votes[string(vote.BlockHash)] = append(votes[string(vote.BlockHash)], vote)

		// Check if we have enough votes to commit the block
		if len(votes[string(vote.BlockHash)]) >= c.validatorSet.QuorumSize() {
			// Load the block from storage
			block, err := c.storage.LoadBlock(vote.BlockHash)
			if err != nil {
				continue
			}

			// Commit the block
			c.commitChan <- block
		}
	}
}

// handleCommits handles block commits
func (c *Consensus) handleCommits() {
	for block := range c.commitChan {
		// Update state
		c.mu.Lock()
		c.state.CurrentWave = uint64(block.Header.Wave)
		c.state.CurrentRound = uint64(block.Header.Round)
		c.state.Height = uint64(block.Header.Height)
		c.state.LatestBlock = block
		c.mu.Unlock()

		// Save state to storage
		if err := c.storage.SaveState(c.state); err != nil {
			continue
		}

		// Process the block
		c.processBlock(block)
	}
}

// isLeader checks if the current node is the leader for this round
func (c *Consensus) isLeader() bool {
	return c.validatorSet.GetLeader(c.currentWave, c.currentRound) == types.Address(c.config.NodeID)
}

// validateBlock validates a block
func (c *Consensus) validateBlock(block *types.Block) bool {
	// TODO: Implement proper block validation
	return true
}

// broadcastProposal broadcasts a block proposal
func (c *Consensus) broadcastProposal(block *types.Block) {
	// TODO: Implement proper block broadcasting
}

// broadcastVote broadcasts a vote
func (c *Consensus) broadcastVote(vote *types.Vote) {
	// TODO: Implement proper vote broadcasting
}

// processBlock processes a committed block
func (c *Consensus) processBlock(block *types.Block) {
	// TODO: Implement block processing
}