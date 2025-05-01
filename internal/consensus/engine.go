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

	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
	"encoding/gob"
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
	config    *Config
	proposals map[string]*types.Proposal
	votes     map[string][]*types.Vote
	mu        sync.RWMutex
	logger    *log.Logger
	waveManager *WaveManager
}

// NewEngine creates a new consensus engine with the given configuration
func NewEngine(config *Config) *Engine {
	return &Engine{
		config:    config,
		proposals: make(map[string]*types.Proposal),
		votes:     make(map[string][]*types.Vote),
		logger:    log.New(log.Writer(), "[Consensus] ", log.LstdFlags),
		waveManager: NewWaveManager(nil),
	}
}

// HandleProposal handles a new block proposal
func (e *Engine) HandleProposal(proposal *types.Proposal) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Printf("Received proposal for block %s in wave %d, round %d", 
		proposal.BlockHash, proposal.Wave, proposal.Round)
	e.proposals[string(proposal.BlockHash)] = proposal
	return nil
}

// HandleVote handles a new vote on a proposal
func (e *Engine) HandleVote(vote *types.Vote) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Printf("Received vote for block %s from validator %s in wave %d, round %d",
		vote.BlockHash, vote.Validator, vote.Wave, vote.Round)
	e.votes[string(vote.BlockHash)] = append(e.votes[string(vote.BlockHash)], vote)
	return nil
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
	mu            sync.RWMutex
	running       bool
	nodeID        types.Address
	dag           *core.DAG
	stateManager  *state.StateManager
	waveManager   *WaveManager
	currentWave   *WaveState
	currentRound  types.Round
	currentHeight types.BlockNumber
	currentLeader types.Address
	config        *Config
	proposals     map[string]*types.Proposal
	validators    []types.Address
	waves         map[types.Wave]*WaveState
	logger        *log.Logger
	blockProcessor *core.BlockProcessor
	networkServer  *NetworkServer
	votes         map[string][]*types.Vote
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
		
		// Only set height if we have a latest block
		if state.LatestBlock != nil {
			ce.currentHeight = state.LatestBlock.Header.Height
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

	// Initialize network server
	ce.networkServer = NewNetworkServer(ce)

	// Start network server
	if err := ce.networkServer.Start(); err != nil {
		return fmt.Errorf("failed to start network server: %v", err)
	}

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
	// ce.logger.Printf("Selected leader %s for wave %d", 
	// 	ce.currentLeader, ce.currentWave.GetWaveNumber())
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

	ce.logger.Printf("Processing vote for block %s from validator %s in wave %d, round %d",
		vote.BlockHash, vote.Validator, vote.Wave, vote.Round)

	// Add vote to wave state
	if err := ce.currentWave.AddProposalVote(vote); err != nil {
		return fmt.Errorf("failed to add vote: %v", err)
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

	ce.logger.Printf("Creating new block in wave %d, round %d", 
		ce.currentWave.GetWaveNumber(), ce.currentRound)

	// Use BlockProcessor to create block
	fmt.Print("111111111")
	block, err := ce.blockProcessor.CreateBlock(ce.currentRound)
	if err != nil {
		ce.logger.Printf("Failed to create block: %v", err)
		return nil, err
	}

	ce.logger.Printf("Created block %s in wave %d, round %d", 
		block.ComputeHash(), ce.currentWave.GetWaveNumber(), ce.currentRound)
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
		Validator:  ce.nodeID,
		Signature:  []byte("dummy_signature"),
		Timestamp:  time.Now(),
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

	// Sign the block
	if err := ce.signBlock(block); err != nil {
		return fmt.Errorf("failed to sign block: %v", err)
	}

	// Compute block hash once
	blockHash := block.ComputeHash()

	// Create proposal
	proposal := &types.Proposal{
		ID:        blockHash,
		BlockHash: blockHash,
		Block:     block,
		Proposer:  ce.nodeID,
		Timestamp: time.Now(),
	}

	// Store the proposal
	ce.proposals[string(blockHash)] = proposal

	// Broadcast to all validators
	for _, validator := range ce.validators {
		if validator != ce.nodeID { // Don't send to self
			// Get validator's address from config
			validatorAddr := ce.getValidatorAddress(validator)
			if validatorAddr == "" {
				ce.logger.Printf("Warning: No address found for validator %s", validator)
				continue
			}

			ce.logger.Printf("Broadcasting block %x to validator %s at %s", 
				blockHash, validator, validatorAddr)

			// Create connection to validator
			conn, err := net.Dial("tcp", validatorAddr)
			if err != nil {
				ce.logger.Printf("Error connecting to validator %s at %s: %v", validator, validatorAddr, err)
				continue
			}
			defer conn.Close()

			// Set write deadline
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			// Create a buffer to hold the encoded proposal
			var buf bytes.Buffer
			encoder := gob.NewEncoder(&buf)
			if err := encoder.Encode(proposal); err != nil {
				ce.logger.Printf("Error encoding proposal for validator %s: %v", validator, err)
				continue
			}

			// Write the encoded proposal to the connection
			if _, err := conn.Write(buf.Bytes()); err != nil {
				ce.logger.Printf("Error sending proposal to validator %s: %v", validator, err)
				continue
			}

			// Wait for acknowledgment
			ackBuf := make([]byte, 3) // "ACK" is 3 bytes
			conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			if _, err := conn.Read(ackBuf); err != nil {
				ce.logger.Printf("Error receiving acknowledgment from validator %s: %v", validator, err)
				continue
			}

			if !bytes.Equal(ackBuf, []byte("ACK")) {
				ce.logger.Printf("Invalid acknowledgment from validator %s", validator)
				continue
			}

			ce.logger.Printf("Successfully broadcasted block %x to validator %s", 
				blockHash, validator)
		}
	}

	return nil
}

// getValidatorAddress returns the network address for a validator
func (ce *ConsensusEngine) getValidatorAddress(validator types.Address) string {
	// Get validator's listen address from config
	for _, v := range ce.config.ValidatorSet {
		if v == validator {
			// Determine port based on validator
			var port string
			switch string(validator) {
			case "validator1":
				port = "3000"
			case "validator2":
				port = "3001"
			case "validator3":
				port = "3002"
			default:
				return ""
			}

			// For local testing, use localhost
			return fmt.Sprintf("localhost:%s", port)
		}
	}
	return ""
}

// HandleProposal handles incoming block proposals from other validators
func (ce *ConsensusEngine) HandleProposal(proposal *types.Proposal) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if !ce.running {
		return errors.New("engine is not running")
	}

	// Verify proposal
	if err := ce.verifyProposal(proposal); err != nil {
		return fmt.Errorf("invalid proposal: %v", err)
	}

	// Store proposal
	ce.proposals[string(proposal.BlockHash)] = proposal

	// Process block
	if err := ce.processBlock(proposal.Block); err != nil {
		return fmt.Errorf("failed to process block: %v", err)
	}

	ce.logger.Printf("Processed proposal for block %s from validator %s", 
		string(proposal.BlockHash), string(proposal.Proposer))

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

	// Verify block hash matches
	computedHash := proposal.Block.ComputeHash()
	if !bytes.Equal(proposal.BlockHash, computedHash) {
		// ce.logger.Printf("Block hash mismatch: expected %x, got %x", computedHash, proposal.BlockHash)
		return fmt.Errorf("block hash mismatch")
	}

	// Verify block signature
	if err := ce.verifyBlockSignature(proposal.Block); err != nil {
		return fmt.Errorf("invalid block signature: %v", err)
	}

	return nil
}

func (ce *ConsensusEngine) createProposal() *types.Proposal {
	return &types.Proposal{
		Timestamp:  time.Now(),
		Round:      ce.currentRound,
		Wave:       ce.currentWave.GetWaveNumber(),
		BlockHash:  ce.getParentHash(),
		Proposer:   ce.nodeID,
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
		ce.currentWave.Number++
		ce.currentWave.Status = types.WaveStatusProposing
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
	listener net.Listener
	engine   *ConsensusEngine
	running  bool
	mu       sync.RWMutex
	port     string
}

// NewNetworkServer creates a new network server
func NewNetworkServer(engine *ConsensusEngine) *NetworkServer {
	// Extract port from listen address
	addr := string(engine.config.ListenAddr)
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		engine.logger.Printf("Warning: Invalid listen address: %v", err)
		port = "3000" // Default port
	}

	return &NetworkServer{
		engine: engine,
		port:   port,
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

	ns.engine.logger.Printf("Network server started on port %s", ns.port)
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
	
	ns.engine.logger.Printf("Network server stopped on port %s", ns.port)
}

// acceptConnections accepts incoming connections
func (ns *NetworkServer) acceptConnections() {
	for ns.running {
		conn, err := ns.listener.Accept()
		if err != nil {
			if ns.running {
				ns.engine.logger.Printf("Error accepting connection: %v", err)
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
	ns.engine.logger.Printf("New connection from %s", remoteAddr)

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Read all data from the connection
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		ns.engine.logger.Printf("Error reading from %s: %v", remoteAddr, err)
		return
	}

	// Create a buffer with the received data
	reader := bytes.NewReader(buf[:n])

	// Decode message
	decoder := gob.NewDecoder(reader)
	var proposal types.Proposal
	if err := decoder.Decode(&proposal); err != nil {
		ns.engine.logger.Printf("Error decoding message from %s: %v", remoteAddr, err)
		return
	}

	// Verify the proposal came from a valid validator
	isValidValidator := false
	for _, v := range ns.engine.validators {
		if v == proposal.Proposer {
			isValidValidator = true
			break
		}
	}
	if !isValidValidator {
		ns.engine.logger.Printf("Invalid proposer: %s", string(proposal.Proposer))
		return
	}

	ns.engine.logger.Printf("Received proposal for block %x from %s", 
		proposal.BlockHash, remoteAddr)

	// Handle proposal
	if err := ns.engine.HandleProposal(&proposal); err != nil {
		// ns.engine.logger.Printf("Error handling proposal from %s: %v", remoteAddr, err)
		return
	}

	// Send acknowledgment
	ack := []byte("ACK")
	if _, err := conn.Write(ack); err != nil {
		ns.engine.logger.Printf("Error sending acknowledgment to %s: %v", remoteAddr, err)
		return
	}

	ns.engine.logger.Printf("Successfully processed proposal from %s", remoteAddr)
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

	// Add block to DAG
	if err := ce.dag.AddBlock(block); err != nil {
		return fmt.Errorf("failed to add block to DAG: %v", err)
	}

	// Update state
	if err := ce.stateManager.CommitBlock(block); err != nil {
		return fmt.Errorf("failed to commit block: %v", err)
	}

	ce.logger.Printf("Processed block %s at height %d", 
		string(block.ComputeHash()), block.Header.Height)
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