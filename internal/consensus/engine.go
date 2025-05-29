package consensus

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"

	// "strings"
	"sync"
	"time"

	"encoding/gob"

	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

var (
	ErrNoActiveWave = errors.New("no active wave")
)

// Proposal represents a block proposal in the consensus process
type Proposal struct {
	BlockHash string
	Round     uint64
	Sender    string
	Timestamp time.Time
}

// Vote represents a vote on a proposal
type Vote struct {
	BlockHash string
	Round     uint64
	Wave      uint64
	Validator []byte
	Timestamp time.Time
}

// Wave represents a consensus wave
type Wave struct {
	Number    types.Wave
	StartTime time.Time
	EndTime   time.Time
	Status    types.WaveStatus
	Leader    types.Address
	Votes     map[types.Address]bool
}

// Engine represents the consensus engine
type Engine struct {
	config      *Config
	proposals   map[string]*types.Proposal
	votes       map[string][]*types.Vote
	mu          sync.RWMutex
	logger      *log.Logger
	waveManager *WaveManager
	dag         *core.DAG
}

// NewEngine creates a new consensus engine with the given configuration
func NewEngine(config *Config) *Engine {
	return &Engine{
		config:      config,
		proposals:   make(map[string]*types.Proposal),
		votes:       make(map[string][]*types.Vote),
		logger:      log.New(log.Writer(), "[Consensus] ", log.LstdFlags),
		waveManager: NewWaveManager(nil),
		dag:         core.GetDAG(), // Use singleton DAG
	}
}

// HasQuorum checks if a proposal has received enough votes to reach quorum
func (e *Engine) HasQuorum() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Count unique validators who voted
	validators := make(map[types.Address]bool)
	for _, votes := range e.votes {
		for _, vote := range votes {
			validators[vote.Validator] = true
		}
	}

	// Check if we have enough votes (2/3 + 1)
	hasQuorum := len(validators) >= (2*e.config.TotalValidators/3 + 1)
	if hasQuorum {
		e.logger.Printf("Quorum reached with %d validators", len(validators))
	}
	return hasQuorum
}

// ConsensusEngine represents the consensus engine
type ConsensusEngine struct {
	mu                sync.RWMutex
	running           bool
	nodeID            types.Address
	dag               *core.DAG
	stateManager      *state.StateManager
	waveManager       *WaveManager
	currentWave       *WaveState
	currentRound      types.Round
	currentHeight     types.BlockNumber
	currentLeader     types.Address
	config            *Config
	proposals         map[string]*types.Proposal
	validators        []types.Address
	waves             map[types.Wave]*WaveState
	logger            *log.Logger
	blockProcessor    *core.BlockProcessor
	networkServer     *NetworkServer
	votes             map[string][]*types.Vote
	savedLeaderBlock  *types.Block
	blockSynchronizer *BlockSynchronizer
	syncServer        *SyncServer
}

// NewConsensusEngine creates a new consensus engine
func NewConsensusEngine(config *Config, stateManager *state.StateManager, blockProcessor *core.BlockProcessor) *ConsensusEngine {
	engine := &ConsensusEngine{
		config:         config,
		stateManager:   stateManager,
		blockProcessor: blockProcessor,
		proposals:      make(map[string]*types.Proposal),
		waves:          make(map[types.Wave]*WaveState),
		logger:         log.New(log.Writer(), "[Consensus] ", log.LstdFlags),
		votes:          make(map[string][]*types.Vote),
		dag:            core.GetDAG(), // Use singleton DAG
	}

	// Set node ID from config
	engine.nodeID = types.Address(config.NodeID)
	if engine.nodeID == "" {
		engine.logger.Printf("Warning: Node ID is empty")
	}

	// Set listen address from config
	if config.ListenAddr == "" {
		engine.logger.Printf("Warning: Listen address is empty")
	}

	// Initialize network server
	engine.networkServer = NewNetworkServer(engine)

	// Initialize wave manager
	engine.waveManager = NewWaveManager(engine)

	// Create block synchronizer
	syncInterval := 5 * time.Second // Sync every 5 seconds
	blockSynchronizer := NewBlockSynchronizer(engine, syncInterval)

	// Get sync port based on validator ID
	var syncPort int
	switch string(engine.nodeID) {
	case "validator1":
		syncPort = 4000
	case "validator2":
		syncPort = 4000
	default:
		syncPort = 4000
	}

	// Initialize sync server
	syncServer := NewSyncServer(blockSynchronizer, syncPort)

	// Store references to synchronization components
	engine.blockSynchronizer = blockSynchronizer
	engine.syncServer = syncServer

	return engine
}

// Start starts the consensus engine
func (ce *ConsensusEngine) Start() error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if ce.running {
		return nil
	}

	// Initialize validator set
	ce.validators = ce.config.ValidatorSet
	ce.logger.Printf("Initialized validator set with %d validators", len(ce.validators))

	// Get current state from block processor
	state := ce.blockProcessor.GetState()
	if state != nil {
		// Start from the last wave and round
		ce.currentWave = NewWaveState(types.Wave(state.CurrentWave), ce.config.WaveTimeout, ce.config.QuorumSize)
		ce.currentRound = types.Round(state.CurrentRound)

		// Log initial wave
		fmt.Printf("\n====================================\n")
		fmt.Printf("======== STARTING WITH WAVE %d ========\n", ce.currentWave.GetWaveNumber())
		fmt.Printf("====================================\n\n")

		// Only set height if we have a latest block
		if state.LatestBlock != nil {
			ce.currentHeight = state.LatestBlock.Header.Height

			// Load all blocks into DAG
			blocks, err := ce.stateManager.GetAllBlocks()
			if err != nil {
				ce.logger.Printf("Warning: Failed to load blocks from state: %v", err)
			} else {
				for _, block := range blocks {
					if err := ce.dag.AddBlock(block); err != nil {
						if err.Error() == "block already exists" {
							ce.logger.Printf("Block %s already exists in DAG, skipping", block.ComputeHash())
						} else {
							ce.logger.Printf("Warning: Failed to add block %s to DAG: %v", block.ComputeHash(), err)
						}
					}
				}
				ce.logger.Printf("Loaded %d blocks into DAG", len(blocks))
			}
		} else {
			ce.currentHeight = 0
		}

		ce.logger.Printf("Resuming from wave %d, round %d, height %d",
			ce.currentWave.GetWaveNumber(), ce.currentRound, ce.currentHeight)
	} else {
		// Start with wave 1 if no state exists
		ce.currentWave = NewWaveState(1, ce.config.WaveTimeout, ce.config.QuorumSize)
		ce.currentRound = 1
		ce.currentHeight = 0
		ce.logger.Printf("Starting new chain from wave 1")
	}

	// Select initial leader
	ce.selectLeader()
	ce.logger.Printf("Selected initial leader: %s", ce.currentLeader)

	// Register types for gob encoding/decoding
	RegisterSyncTypes()

	// Start network server
	if err := ce.networkServer.Start(); err != nil {
		return fmt.Errorf("failed to start network server: %v", err)
	}

	// Start sync server
	if err := ce.syncServer.Start(); err != nil {
		ce.logger.Printf("Warning: Failed to start sync server: %v", err)
	}

	// Start block synchronizer
	ce.blockSynchronizer.Start()

	// Start wave manager
	ce.waveManager.Start()

	ce.running = true
	ce.logger.Printf("Consensus engine started")
	return nil
}

// waveTimer handles wave timeouts
func (ce *ConsensusEngine) waveTimer() {
	lastWaveTime := time.Now()
	lastWave := types.Wave(1) // Start from wave 1

	for ce.running {
		time.Sleep(ce.config.WaveTimeout)
		ce.mu.Lock()
		if ce.currentWave != nil {
			// Only increment wave if the wave timeout has elapsed and we're on the expected wave
			if time.Since(lastWaveTime) >= ce.config.WaveTimeout && ce.currentWave.GetWaveNumber() == lastWave {
				oldWave := ce.currentWave.GetWaveNumber()
				ce.currentWave = NewWaveState(ce.currentWave.GetWaveNumber()+1, ce.config.WaveTimeout, ce.config.QuorumSize)
				ce.selectLeader()

				// Add prominent wave transition logging
				fmt.Printf("\n====================================\n")
				fmt.Printf("======== WAVE %d ENDED ========\n", oldWave)
				fmt.Printf("======== WAVE %d STARTED ========\n", ce.currentWave.GetWaveNumber())
				fmt.Printf("====================================\n\n")

				ce.logger.Printf("Wave forwarded from %d to %d", oldWave, ce.currentWave.GetWaveNumber())
				lastWaveTime = time.Now()
				lastWave = ce.currentWave.GetWaveNumber()
			}
		}
		ce.mu.Unlock()
	}
}

// selectLeader selects a leader for the current wave
func (ce *ConsensusEngine) selectLeader() {
	if len(ce.validators) == 0 {
		ce.logger.Printf("No validators available for leader selection")
		return
	}

	// Select leader based on wave number
	leaderIndex := int(ce.currentWave.GetWaveNumber()) % len(ce.validators)
	ce.currentLeader = ce.validators[leaderIndex]

	// Print leader selection
	// fmt.Printf("\n======================== Wave %d Leader Selection =========================\n", ce.currentWave.GetWaveNumber())
	fmt.Printf("***Leader selected*** %s\n", ce.currentLeader)
	// fmt.Printf("======================================================================\n\n")
}

// IsLeader checks if the current node is the leader
func (ce *ConsensusEngine) IsLeader() bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	if !ce.running || ce.currentWave == nil {
		return false
	}

	isLeader := ce.currentLeader == ce.nodeID
	if isLeader {
		ce.logger.Printf("Node %s is leader for wave %d",
			ce.nodeID, ce.currentWave.GetWaveNumber())
	}
	return isLeader
}

// GetCurrentWave returns the current wave
func (ce *ConsensusEngine) GetCurrentWave() types.Wave {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	if ce.currentWave == nil {
		return 0
	}
	return ce.currentWave.GetWaveNumber()
}

// HandleBlock handles a block
func (ce *ConsensusEngine) HandleBlock(block *types.Block) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return nil
	}

	ce.logger.Printf("Processing block %s in wave %d, round %d",
		block.ComputeHash(), block.Header.Wave, block.Header.Round)

	// Create proposal
	proposal := &types.Proposal{
		ID:        block.ComputeHash(),
		BlockHash: block.ComputeHash(),
		Block:     block,
		Proposer:  ce.nodeID,
		Timestamp: time.Now(),
	}

	// Add proposal to wave state
	if err := ce.currentWave.AddProposal(proposal); err != nil {
		return fmt.Errorf("failed to add proposal: %v", err)
	}

	return nil
}

// HandleVote handles a vote
func (ce *ConsensusEngine) HandleVote(vote *types.Vote) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running || ce.currentWave == nil {
		return ErrNoActiveWave
	}

	fmt.Printf(";;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;Vote received - Block Hash: %x, Validator: %s, Wave: %d, Round: %d, Proposal ID: %x\n",
		vote.BlockHash, vote.Validator, vote.Wave, vote.Round, vote.ProposalID)

	// Validate vote
	if vote == nil {
		return fmt.Errorf("vote is nil")
	}

	// Try to get block from DAG
	_, err := ce.dag.GetBlock(vote.BlockHash)
	if err != nil {
		// If block not found and we have block data in vote, save it to DAG
		if vote.Block != nil {
			ce.logger.Printf("Block not found in DAG, saving block from vote data")
			if err := ce.dag.AddBlock(vote.Block); err != nil {
				return fmt.Errorf("failed to save block from vote: %v", err)
			}
		} else {
			return fmt.Errorf("failed to get block from DAG and no block data in vote: %v", err)
		}
	}

	// Add vote to our vote map
	ce.votes[string(vote.BlockHash)] = append(ce.votes[string(vote.BlockHash)], vote)

	// Count unique validators who voted for this block
	validatorVotes := make(map[types.Address]bool)
	for _, v := range ce.votes[string(vote.BlockHash)] {
		validatorVotes[v.Validator] = true
	}

	ce.logger.Printf("Updated vote count for block %x to %d", vote.BlockHash, len(validatorVotes))

	// Add vote to wave state
	if err := ce.currentWave.AddProposalVote(vote); err != nil {
		ce.logger.Printf("Warning: Failed to add vote to wave state: %v", err)
		// Continue processing even if wave state update fails
	}

	// Check for quorum
	if ce.currentWave.HasQuorum(vote.ProposalID) {
		ce.logger.Printf("Quorum reached for block %s", vote.BlockHash)
		ce.currentWave.SetStatus(types.WaveStatusFinalized)
	}

	return nil
}

// HasQuorum checks if the current wave has reached quorum
func (ce *ConsensusEngine) HasQuorum() bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	if !ce.running || ce.currentWave == nil {
		return false
	}

	// Get all proposals in the current wave
	for _, proposal := range ce.currentWave.GetProposals() {
		if ce.currentWave.HasQuorum(proposal.ID) {
			return true
		}
	}

	return false
}

// GetValidators returns the list of validators
func (ce *ConsensusEngine) GetValidators() []types.Address {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.validators
}

// ProcessTimeout handles wave timeout
func (ce *ConsensusEngine) ProcessTimeout() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running || ce.currentWave == nil {
		return
	}

	if ce.currentWave.IsTimedOut() {
		// ce.logger.Printf("Wave %d timed out", ce.currentWave.GetWaveNumber())
		ce.currentWave.SetStatus(types.WaveStatusFailed)
		ce.currentWave = NewWaveState(ce.currentWave.GetWaveNumber()+1, ce.config.WaveTimeout, ce.config.QuorumSize)
		ce.selectLeader()
	}
}

// HandleCertificate handles a certificate
func (ce *ConsensusEngine) HandleCertificate(cert *types.Certificate) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return nil
	}

	ce.logger.Printf("Processing certificate for block %s in wave %d, round %d",
		cert.BlockHash, cert.Wave, cert.Round)

	// Validate certificate
	if err := ce.validateCertificate(cert); err != nil {
		ce.logger.Printf("Certificate validation failed: %v", err)
		return err
	}

	// Get block
	block, err := ce.dag.GetBlock(cert.BlockHash)
	if err != nil {
		ce.logger.Printf("Failed to get block for certificate: %v", err)
		return err
	}

	// Add certificate to block
	block.Certificate = cert

	// Update state
	if err := ce.stateManager.CommitBlock(block); err != nil {
		ce.logger.Printf("Failed to commit block: %v", err)
		return err
	}

	ce.logger.Printf("Block %s committed with certificate", cert.BlockHash)
	return nil
}

// validateBlock validates a block
func (ce *ConsensusEngine) validateBlock(block *types.Block) error {
	// Check block structure
	if block == nil || block.Header == nil || block.Body == nil {
		return types.ErrInvalidBlock
	}

	// Check block references
	if err := ce.validateReferences(block); err != nil {
		return err
	}

	// Check transactions
	if err := ce.validateTransactions(block); err != nil {
		return err
	}

	return nil
}

// validateVote validates a vote
func (ce *ConsensusEngine) validateVote(vote *types.Vote) error {
	// Check vote structure
	if vote == nil || len(vote.Validator) == 0 {
		return types.ErrInvalidVote
	}

	// Check if block exists
	_, err := ce.dag.GetBlock(vote.BlockHash)
	if err != nil {
		return err
	}

	return nil
}

// validateCertificate validates a certificate
func (ce *ConsensusEngine) validateCertificate(cert *types.Certificate) error {
	// Check certificate structure
	if cert == nil || len(cert.BlockHash) == 0 {
		return types.ErrInvalidCertificate
	}

	// Check if block exists
	_, err := ce.dag.GetBlock(cert.BlockHash)
	if err != nil {
		return err
	}

	return nil
}

// validateReferences validates block references
func (ce *ConsensusEngine) validateReferences(block *types.Block) error {
	// Check parent exists
	if len(block.Header.ParentHash) > 0 {
		_, err := ce.dag.GetBlock(block.Header.ParentHash)
		if err != nil {
			return err
		}
	}

	// Check other references
	for _, ref := range block.Header.References {
		_, err := ce.dag.GetBlock(ref.BlockHash)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateTransactions validates block transactions
func (ce *ConsensusEngine) validateTransactions(block *types.Block) error {
	for _, tx := range block.Body.Transactions {
		// Check transaction structure
		if len(tx.From) == 0 || len(tx.To) == 0 {
			return types.ErrInvalidTransaction
		}

		// Check sender balance
		sender, err := ce.stateManager.GetAccount(string(tx.From))
		if err != nil {
			return err
		}

		if sender.Balance < tx.Value {
			return types.ErrInsufficientBalance
		}

		// Check nonce
		if sender.Nonce != tx.Nonce {
			return types.ErrInvalidNonce
		}
	}

	return nil
}

// StartNewWave starts a new consensus wave
func (ce *ConsensusEngine) StartNewWave() error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	wave, err := ce.waveManager.StartNewWave()
	if err != nil {
		ce.logger.Printf("Failed to start new wave: %v", err)
		return err
	}

	leaderIndex := uint64(wave.Number) % uint64(len(ce.validators))
	wave.Leader = ce.validators[leaderIndex]
	ce.currentWave = wave

	// Add prominent wave start logging
	fmt.Printf("\n====================================\n")
	fmt.Printf("======== NEW WAVE %d STARTED ========\n", wave.Number)
	fmt.Printf("======== LEADER: %s ========\n", wave.Leader)
	fmt.Printf("====================================\n\n")

	ce.logger.Printf("Started new wave %d with leader %s",
		wave.Number, wave.Leader)
	return nil
}

// CreateBlock creates a new block
func (ce *ConsensusEngine) CreateBlock() (*types.Block, error) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return nil, errors.New("engine is not running")
	}

	// Check if we already created a block in this wave
	currentWave := ce.currentWave.GetWaveNumber()
	recentBlocks := ce.dag.GetRecentBlocks(10)
	for _, block := range recentBlocks {
		if block.Header.Wave == currentWave && block.Header.Validator == ce.nodeID {
			ce.logger.Printf("Already created block in wave %d, skipping", currentWave)
			return nil, nil
		}
	}

	fmt.Printf("\n====================================\n")
	fmt.Printf("======== CREATING BLOCK IN WAVE %d ========\n", currentWave)
	fmt.Printf("====================================\n\n")

	ce.logger.Printf("Creating new block in wave %d, round %d",
		currentWave, ce.currentRound)

	// Use BlockProcessor to create block, passing the current wave
	block, err := ce.blockProcessor.CreateBlock(ce.currentRound, currentWave)
	if err != nil {
		ce.logger.Printf("Failed to create block: %v", err)
		return nil, err
	}

	// Block is already added to DAG by BlockProcessor, no need to add again

	ce.logger.Printf("Created block %s in wave %d, round %d",
		block.ComputeHash(), currentWave, ce.currentRound)
	return block, nil
}

// ProcessVote processes a vote from a validator
func (ce *ConsensusEngine) ProcessVote(validator types.Address, vote bool) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if ce.currentWave == nil {
		return fmt.Errorf("no active wave")
	}

	ce.logger.Printf("Processing vote from validator %s in wave %d",
		validator, ce.currentWave.GetWaveNumber())

	// Record vote
	ce.currentWave.AddVote(validator)
	ce.processVote(vote)

	return nil
}

// verifyVote verifies a vote
func (ce *ConsensusEngine) verifyVote(vote *types.Vote) error {
	// TODO: Implement vote verification
	return nil
}

// getNextHeight returns the next block height
func (ce *ConsensusEngine) getNextHeight() types.BlockNumber {
	return ce.currentHeight + 1
}

// getParentHash returns the parent block hash
func (ce *ConsensusEngine) getParentHash() types.Hash {
	if ce.currentHeight == 0 {
		return types.Hash{}
	}
	block, err := ce.dag.GetBlockByHeight(ce.currentHeight)
	if err != nil {
		return types.Hash{}
	}
	return block.ComputeHash()
}

// selectReferences selects references for the new block
func (ce *ConsensusEngine) selectReferences() []*types.Reference {
	// TODO: Implement reference selection logic
	return make([]*types.Reference, 0)
}

// calculateStateRoot calculates the state root
func (ce *ConsensusEngine) calculateStateRoot() types.Hash {
	// TODO: Implement state root calculation
	return types.Hash{}
}

// signBlock signs a block
func (ce *ConsensusEngine) signBlock(block *types.Block) error {
	// TODO: Implement proper block signing
	// For now, just create a dummy signature
	block.Header.Signature = types.Signature{
		Validator: ce.nodeID,
		Signature: []byte("dummy_signature"),
		Timestamp: time.Now(),
	}
	return nil
}

// BroadcastBlock broadcasts a block to all validators
func (ce *ConsensusEngine) BroadcastBlock(block *types.Block) error {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	if !ce.running {
		return errors.New("engine is not running")
	}

	// Check if block is nil
	if block == nil {
		ce.logger.Printf("❌ BROADCAST FAILED: Cannot broadcast nil block")
		return nil
	}

	// Sign the block
	if err := ce.signBlock(block); err != nil {
		return fmt.Errorf("failed to sign block: %v", err)
	}

	// Compute block hash once
	blockHash := block.ComputeHash()

	// Log detailed information about the block being broadcast
	ce.logger.Printf("📤📤📤 STARTING BLOCK BROADCAST 📤📤📤")
	ce.logger.Printf("📤 Block Hash: %x", blockHash)
	ce.logger.Printf("📤 Block Height: %d, Wave: %d, Round: %d",
		block.Header.Height, block.Header.Wave, block.Header.Round)
	ce.logger.Printf("📤 Block Validator: %s", block.Header.Validator)
	ce.logger.Printf("📤 Block References: %d", len(block.Header.References))
	ce.logger.Printf("📤 Block Transactions: %d", len(block.Body.Transactions))
	ce.logger.Printf("📤 All validators: %v", ce.validators)
	ce.logger.Printf("📤 Our node ID: %s", ce.nodeID)

	// Create proposal with the full block
	proposal := &types.Proposal{
		ID:        blockHash,
		BlockHash: blockHash,
		Block:     block,
		Proposer:  ce.nodeID,
		Timestamp: time.Now(),
		Wave:      block.Header.Wave,
		Round:     block.Header.Round,
	}

	// Store the proposal
	ce.proposals[string(blockHash)] = proposal

	// Broadcast to all validators
	successCount := 0
	totalTargets := 0
	for _, validator := range ce.validators {
		if validator != ce.nodeID { // Don't send to self
			totalTargets++
			validatorAddr := ce.getValidatorAddress(validator)
			if validatorAddr == "" {
				ce.logger.Printf("⚠️ WARNING: No address found for validator %s", validator)
				continue
			}

			ce.logger.Printf("📤 SENDING to validator %s at %s...", validator, validatorAddr)

			// Create connection to validator
			conn, err := net.Dial("tcp", validatorAddr)
			if err != nil {
				ce.logger.Printf("❌ FAILED to connect to validator %s at %s: %v", validator, validatorAddr, err)
				continue
			}
			defer conn.Close()

			// Set write deadline
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			// Create a buffer to hold the encoded proposal
			var buf bytes.Buffer
			encoder := gob.NewEncoder(&buf)
			if err := encoder.Encode(proposal); err != nil {
				ce.logger.Printf("❌ FAILED to encode proposal for validator %s: %v", validator, err)
				continue
			}

			ce.logger.Printf("📤 Sending %d bytes to validator %s", buf.Len(), validator)

			// Write the encoded proposal to the connection
			if _, err := conn.Write(buf.Bytes()); err != nil {
				ce.logger.Printf("❌ FAILED to send proposal to validator %s: %v", validator, err)
				continue
			}

			ce.logger.Printf("📤 Waiting for ACK from validator %s...", validator)

			// Wait for acknowledgment
			ackBuf := make([]byte, 3) // "ACK" is 3 bytes
			conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			if _, err := conn.Read(ackBuf); err != nil {
				ce.logger.Printf("❌ FAILED to receive ACK from validator %s: %v", validator, err)
				continue
			}

			if !bytes.Equal(ackBuf, []byte("ACK")) {
				ce.logger.Printf("❌ Invalid ACK from validator %s: got %s", validator, string(ackBuf))
				continue
			}

			ce.logger.Printf("✅ SUCCESS: Block sent to validator %s", validator)
			successCount++
		}
	}

	ce.logger.Printf("📤📤📤 BROADCAST COMPLETE: %d/%d validators received block %x 📤📤📤",
		successCount, totalTargets, blockHash)

	if successCount == 0 && totalTargets > 0 {
		ce.logger.Printf("❌❌❌ BROADCAST FAILED: No validators received the block! ❌❌❌")
	}

	return nil
}

// getValidatorAddress returns the network address for a validator
func (ce *ConsensusEngine) getValidatorAddress(validator types.Address) string {
	// Get validator's listen address from config
	for _, v := range ce.config.ValidatorSet {
		if v == validator {
			// Map validators to their actual IP addresses
			switch string(validator) {
			case "validator1":
				return "54.183.204.244:3000"
			case "validator2":
				return "52.53.192.236:3000"
			default:
				return ""
			}
		}
	}
	return ""
}

// HandleProposal handles incoming block proposals from other validators
func (ce *ConsensusEngine) HandleProposal(proposal *types.Proposal) error {
	// Check if engine is running first (without mutex)
	ce.mu.RLock()
	running := ce.running
	ce.mu.RUnlock()

	if !running {
		return errors.New("engine is not running")
	}

	// Enhanced logging for proposal handling
	ce.logger.Printf("Received proposal - BlockHash: %x, Wave: %d, Round: %d, Proposer: %s",
		proposal.BlockHash, proposal.Wave, proposal.Round, proposal.Proposer)

	// Check if we already have this proposal (with mutex)
	ce.mu.RLock()
	_, exists := ce.proposals[string(proposal.BlockHash)]
	ce.mu.RUnlock()

	if exists {
		ce.logger.Printf("Proposal already exists, skipping duplicate")
		return nil
	}

	// Verify proposal (this doesn't need mutex for most operations)
	if err := ce.verifyProposal(proposal); err != nil {
		ce.logger.Printf("Proposal verification failed: %v", err)
		return fmt.Errorf("invalid proposal: %v", err)
	}

	// Check if proposer is the current wave leader (with mutex)
	ce.mu.Lock()
	if ce.isWaveLeader(types.Address(proposal.Proposer)) {
		// Save the block from wave leader
		ce.savedLeaderBlock = proposal.Block
		ce.logger.Printf("Saved block from wave leader %s", string(proposal.Proposer))
	}

	// Store proposal
	ce.proposals[string(proposal.BlockHash)] = proposal
	ce.mu.Unlock()

	// Process block (this is the slow operation, do it without mutex)
	if err := ce.processBlock(proposal.Block); err != nil {
		ce.logger.Printf("Failed to process block: %v", err)
		return fmt.Errorf("failed to process block: %v", err)
	}

	ce.logger.Printf("Successfully processed proposal for block %x from validator %s",
		proposal.BlockHash, string(proposal.Proposer))

	return nil
}

// verifyProposal verifies a block proposal
func (ce *ConsensusEngine) verifyProposal(proposal *types.Proposal) error {
	// Verify proposer is a valid validator
	isValidValidator := false
	for _, v := range ce.validators {
		if v == proposal.Proposer {
			isValidValidator = true
			break
		}
	}
	if !isValidValidator {
		return fmt.Errorf("invalid proposer: %s", string(proposal.Proposer))
	}

	// Verify block exists
	if proposal.Block == nil {
		return fmt.Errorf("proposal has nil block")
	}

	// Verify block hash matches
	computedHash := proposal.Block.ComputeHash()
	if !bytes.Equal(proposal.BlockHash, computedHash) {
		ce.logger.Printf("Block hash mismatch: expected %x, got %x", computedHash, proposal.BlockHash)
		return fmt.Errorf("block hash mismatch")
	}

	// Log detailed information about the block being verified
	ce.logger.Printf("Verifying block - Hash: %x, Height: %d, Wave: %d, Round: %d, Validator: %s, References: %d",
		computedHash, proposal.Block.Header.Height, proposal.Block.Header.Wave,
		proposal.Block.Header.Round, proposal.Block.Header.Validator,
		len(proposal.Block.Header.References))

	// Add block to DAG, but handle "already exists" gracefully
	if err := ce.dag.AddBlock(proposal.Block); err != nil {
		if err.Error() == "block already exists" {
			ce.logger.Printf("Block %x already exists in DAG, continuing", computedHash)
		} else {
			return fmt.Errorf("failed to add block to DAG: %v", err)
		}
	} else {
		ce.logger.Printf("Successfully added block %x to DAG", computedHash)
	}

	// Verify block signature
	if err := ce.verifyBlockSignature(proposal.Block); err != nil {
		return fmt.Errorf("invalid block signature: %v", err)
	}

	ce.logger.Printf("------------------------------------PROPOSAL VERIFIED------------------------------------")

	return nil
}

func (ce *ConsensusEngine) createProposal() *types.Proposal {
	return &types.Proposal{
		Timestamp: time.Now(),
		Round:     ce.currentRound,
		Wave:      ce.currentWave.GetWaveNumber(),
		BlockHash: ce.getParentHash(),
		Proposer:  ce.nodeID,
	}
}

func (ce *ConsensusEngine) processVote(vote bool) {
	if vote {
		ce.currentWave.SetStatus(types.WaveStatusProposing)
	} else {
		ce.currentWave.SetStatus(types.WaveStatusFailed)
	}

	// Check if we have enough votes (2/3 + 1)
	validators := ce.currentWave.GetVotes()
	if len(validators) >= (2*ce.config.TotalValidators/3 + 1) {
		ce.currentWave.SetStatus(types.WaveStatusFinalized)
	}
}

func (ce *ConsensusEngine) advanceWave() {
	if ce.currentWave.Status != types.WaveStatusFinalized {
		oldWave := ce.currentWave.Number
		ce.currentWave.Number++
		ce.currentWave.Status = types.WaveStatusProposing

		// Add prominent wave advancement logging
		fmt.Printf("\n====================================\n")
		fmt.Printf("======== WAVE %d ADVANCED TO %d ========\n", oldWave, ce.currentWave.Number)
		fmt.Printf("====================================\n\n")

		ce.logger.Printf("Wave advanced from %d to %d", oldWave, ce.currentWave.Number)
	}
}

func (ce *ConsensusEngine) resetWave() {
	if ce.currentWave.Status != types.WaveStatusFinalized {
		ce.currentWave.Number = 0
		ce.currentWave.Status = types.WaveStatusProposing
	}
}

// GetRecentBlocks returns the most recent blocks
func (ce *ConsensusEngine) GetRecentBlocks(count int) []*types.Block {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.dag.GetRecentBlocks(count)
}

// GetBlock returns a block by its hash
func (ce *ConsensusEngine) GetBlock(hash types.Hash) (*types.Block, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.dag.GetBlock(hash)
}

// NetworkServer handles incoming connections and messages
type NetworkServer struct {
	listener        net.Listener
	consensusEngine *ConsensusEngine
	running         bool
	mu              sync.RWMutex
	port            string
}

// initGobTypes initializes the types for gob encoding/decoding
func initGobTypes() {
	gob.Register(&types.Proposal{})
	gob.Register(&types.Block{})
	gob.Register(&types.Transaction{})
	gob.Register(&types.Vote{})
	gob.Register(&types.Certificate{})
}

// NewNetworkServer creates a new network server
func NewNetworkServer(engine *ConsensusEngine) *NetworkServer {
	// Initialize gob types
	initGobTypes()

	// Extract port from listen address
	addr := string(engine.config.ListenAddr)
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		engine.logger.Printf("Warning: Invalid listen address: %v", err)
		port = "3000" // Default port
	}

	return &NetworkServer{
		consensusEngine: engine,
		port:            port,
	}
}

// Start starts the network server
func (ns *NetworkServer) Start() error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if ns.running {
		return nil
	}

	// Start listening for connections
	listener, err := net.Listen("tcp", ":"+ns.port)
	if err != nil {
		return fmt.Errorf("failed to start network server: %v", err)
	}

	ns.listener = listener
	ns.running = true

	// Start accepting connections
	go ns.acceptConnections()

	ns.consensusEngine.logger.Printf("Network server started on port %s", ns.port)
	return nil
}

// Stop stops the network server
func (ns *NetworkServer) Stop() {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if !ns.running {
		return
	}

	ns.running = false
	if ns.listener != nil {
		ns.listener.Close()
	}

	ns.consensusEngine.logger.Printf("Network server stopped on port %s", ns.port)
}

// acceptConnections accepts incoming connections
func (ns *NetworkServer) acceptConnections() {
	for ns.running {
		conn, err := ns.listener.Accept()
		if err != nil {
			if ns.running {
				ns.consensusEngine.logger.Printf("Error accepting connection: %v", err)
			}
			continue
		}

		// Handle connection in a new goroutine
		go ns.handleConnection(conn)
	}
}

// handleConnection handles an incoming connection
func (ns *NetworkServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	ns.consensusEngine.logger.Printf("📥 NEW CONNECTION from %s", remoteAddr)

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Read all data from the connection
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		ns.consensusEngine.logger.Printf("❌ ERROR reading from %s: %v", remoteAddr, err)
		return
	}

	ns.consensusEngine.logger.Printf("📥 RECEIVED %d bytes from %s", n, remoteAddr)

	// Create a buffer with the received data
	reader := bytes.NewReader(buf[:n])

	// Decode message
	decoder := gob.NewDecoder(reader)
	var proposal types.Proposal
	if err := decoder.Decode(&proposal); err != nil {
		ns.consensusEngine.logger.Printf("❌ ERROR decoding proposal from %s: %v", remoteAddr, err)
		return
	}

	// Basic validation before passing to consensus engine
	if proposal.Block == nil {
		ns.consensusEngine.logger.Printf("❌ ERROR: Received proposal with nil block from %s", remoteAddr)
		return
	}

	// Log the received proposal details
	blockHash := proposal.BlockHash
	ns.consensusEngine.logger.Printf("📥📥📥 RECEIVED BLOCK PROPOSAL 📥📥📥")
	ns.consensusEngine.logger.Printf("📥 From: %s", remoteAddr)
	ns.consensusEngine.logger.Printf("📥 Block Hash: %x", blockHash)
	ns.consensusEngine.logger.Printf("📥 Proposer: %s", proposal.Proposer)
	ns.consensusEngine.logger.Printf("📥 Block Validator: %s", proposal.Block.Header.Validator)
	ns.consensusEngine.logger.Printf("📥 Wave: %d, Round: %d", proposal.Wave, proposal.Round)
	ns.consensusEngine.logger.Printf("📥 Block Height: %d", proposal.Block.Header.Height)
	ns.consensusEngine.logger.Printf("📥 Block References: %d", len(proposal.Block.Header.References))
	ns.consensusEngine.logger.Printf("📥 Block Transactions: %d", len(proposal.Block.Body.Transactions))

	// Verify the proposal came from a valid validator
	isValidValidator := false
	for _, v := range ns.consensusEngine.validators {
		if v == proposal.Proposer {
			isValidValidator = true
			break
		}
	}
	if !isValidValidator {
		ns.consensusEngine.logger.Printf("❌ REJECTED: Invalid proposer %s", string(proposal.Proposer))
		return
	}

	ns.consensusEngine.logger.Printf("📥 PROCESSING proposal from valid validator %s...", proposal.Proposer)

	// Use HandleProposal to process the received proposal
	if err := ns.consensusEngine.HandleProposal(&proposal); err != nil {
		ns.consensusEngine.logger.Printf("❌ ERROR handling proposal from %s: %v", remoteAddr, err)
		return
	}

	ns.consensusEngine.logger.Printf("📥 SENDING ACK to %s", remoteAddr)

	// Send acknowledgment
	ack := []byte("ACK")
	if _, err := conn.Write(ack); err != nil {
		ns.consensusEngine.logger.Printf("❌ ERROR sending ACK to %s: %v", remoteAddr, err)
		return
	}

	ns.consensusEngine.logger.Printf("✅ SUCCESSFULLY processed proposal from %s and sent ACK", remoteAddr)
}

// Stop stops the consensus engine
func (ce *ConsensusEngine) Stop() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return
	}

	// Stop network server
	ce.networkServer.Stop()

	// Stop sync server
	ce.syncServer.Stop()

	// Stop block synchronizer
	ce.blockSynchronizer.Stop()

	// Stop wave manager
	ce.waveManager.Stop()

	ce.running = false
	ce.logger.Printf("Consensus engine stopped")
}

// processBlock processes a received block
func (ce *ConsensusEngine) processBlock(block *types.Block) error {
	// Verify block
	if err := ce.verifyBlock(block); err != nil {
		return fmt.Errorf("block verification failed: %v", err)
	}

	// Log block details
	ce.logger.Printf("Processing block - Hash=%x, Height=%d, Wave=%d, Round=%d, Validator=%s, References=%d",
		block.ComputeHash(), block.Header.Height, block.Header.Wave, block.Header.Round,
		block.Header.Validator, len(block.Header.References))

	// Log transaction details
	for i, tx := range block.Body.Transactions {
		ce.logger.Printf("Transaction %d: From=%s, To=%s, Value=%d, Nonce=%d",
			i, string(tx.From), string(tx.To), tx.Value, tx.Nonce)
	}

	// Add block to DAG (if not already added in verification)
	// This is a safeguard to ensure the block is in the DAG
	blockHash := block.ComputeHash()
	_, err := ce.dag.GetBlock(blockHash)
	if err != nil {
		// Block not found in DAG, add it
		if err := ce.dag.AddBlock(block); err != nil {
			if err.Error() == "block already exists" {
				ce.logger.Printf("Block already exists in DAG during processing, continuing")
			} else {
				ce.logger.Printf("Failed to add block to DAG during processing: %v", err)
				return err
			}
		} else {
			ce.logger.Printf("Added block %x to DAG during processing", blockHash)
		}
	} else {
		ce.logger.Printf("Block %x already exists in DAG, skipping addition", blockHash)
	}

	// Handle block from other validators
	if block.Header.Validator != ce.nodeID {
		ce.logger.Printf("Block is from another validator: %s, our ID: %s",
			block.Header.Validator, ce.nodeID)
	}

	existingTxs := ce.blockProcessor.GetMempoolTransactions()
	existingTxMap := make(map[string]*types.Transaction)
	for _, tx := range existingTxs {
		existingTxMap[string(tx.GetHash())] = tx
	}

	// Save transactions from block back to mempool
	for _, tx := range block.Body.Transactions {
		txHash := string(tx.GetHash())
		if existingTx, exists := existingTxMap[txHash]; !exists {
			// Set transaction state to Pending
			tx.State = types.TransactionStatePending
			ce.blockProcessor.AddTransaction(tx)
			ce.logger.Printf("Added transaction from block back to mempool - Hash: %x", tx.GetHash())
		} else {
			// Update existing transaction state if needed
			if existingTx.State != types.TransactionStatePending {
				existingTx.State = types.TransactionStatePending
				ce.logger.Printf("Updated transaction state in mempool - Hash: %x", tx.GetHash())
			}
		}
	}

	// Update state
	if err := ce.stateManager.CommitBlock(block); err != nil {
		return fmt.Errorf("failed to commit block: %v", err)
	}

	ce.logger.Printf("Successfully processed block %x at height %d from validator %s",
		blockHash, block.Header.Height, block.Header.Validator)
	return nil
}

// verifyBlock verifies a block
func (ce *ConsensusEngine) verifyBlock(block *types.Block) error {
	// Check block structure
	if block == nil || block.Header == nil {
		return fmt.Errorf("invalid block structure")
	}

	// Verify block signature
	if err := ce.verifyBlockSignature(block); err != nil {
		return fmt.Errorf("invalid block signature: %v", err)
	}

	// Verify block references
	if err := ce.verifyBlockReferences(block); err != nil {
		return fmt.Errorf("invalid block references: %v", err)
	}

	return nil
}

// verifyBlockSignature verifies a block's signature
func (ce *ConsensusEngine) verifyBlockSignature(block *types.Block) error {
	// TODO: Implement proper signature verification
	// For now, just check if signature is valid
	if block.Header.Signature.Validator == "" || len(block.Header.Signature.Signature) == 0 {
		return fmt.Errorf("block has invalid signature")
	}

	// Verify validator is in the validator set
	isValidValidator := false
	for _, v := range ce.validators {
		if v == block.Header.Signature.Validator {
			isValidValidator = true
			break
		}
	}
	if !isValidValidator {
		return fmt.Errorf("block signed by invalid validator: %s", string(block.Header.Signature.Validator))
	}

	return nil
}

// verifyBlockReferences verifies block references
func (ce *ConsensusEngine) verifyBlockReferences(block *types.Block) error {
	// Check if parent exists
	if len(block.Header.ParentHash) > 0 {
		parent, err := ce.dag.GetBlock(block.Header.ParentHash)
		if err != nil {
			return fmt.Errorf("parent block not found: %v", err)
		}
		if parent == nil {
			return fmt.Errorf("parent block is nil")
		}
	}

	// Check other references
	for _, ref := range block.Header.References {
		refBlock, err := ce.dag.GetBlock(ref.BlockHash)
		if err != nil {
			return fmt.Errorf("reference block not found: %v", err)
		}
		if refBlock == nil {
			return fmt.Errorf("reference block is nil")
		}
	}

	return nil
}

// IsBlockApproved checks if a block has been approved by the consensus
func (ce *ConsensusEngine) IsBlockApproved(blockHash types.Hash) bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	// Check if we have enough votes for this block
	votes, exists := ce.votes[string(blockHash)]
	if !exists {
		return false
	}

	// Count unique validators who voted for this block
	validatorVotes := make(map[types.Address]bool)
	for _, vote := range votes {
		validatorVotes[vote.Validator] = true
	}

	// Check if we have enough unique validator votes
	return len(validatorVotes) >= ce.config.QuorumSize
}

// GetBlockVotes returns the number of votes received for a block
func (ce *ConsensusEngine) GetBlockVotes(blockHash types.Hash) int {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	votes, exists := ce.votes[string(blockHash)]
	if !exists {
		return 0
	}

	// Count unique validators who voted for this block
	validatorVotes := make(map[types.Address]bool)
	for _, vote := range votes {
		validatorVotes[vote.Validator] = true
	}

	return len(validatorVotes)
}

func (ce *ConsensusEngine) isWaveLeader(validator types.Address) bool {
	return ce.currentLeader == validator
}

// GetSavedLeaderBlock returns the saved leader block
func (ce *ConsensusEngine) GetSavedLeaderBlock() *types.Block {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.savedLeaderBlock
}

// GetNodeID returns the node ID
func (ce *ConsensusEngine) GetNodeID() string {
	return string(ce.nodeID)
}

// GetWaveStatus returns the current wave status
func (ce *ConsensusEngine) GetWaveStatus() string {
	if ce.currentWave == nil {
		return "unknown"
	}

	switch ce.currentWave.Status {
	case 0:
		return "initializing"
	case 1:
		return "proposing"
	case 2:
		return "finalized"
	case 3:
		return "failed"
	default:
		return "unknown"
	}
}
