package consensus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/samuel0642/BlazeDAG/internal/types"
)

// WaveState represents the current state of a consensus wave
type WaveState struct {
	mu sync.RWMutex

	// Wave information
	currentWave     uint64
	waveStartTime   time.Time
	waveEndTime     time.Time
	waveTimeout     time.Duration
	roundDuration   time.Duration

	// Leader information
	currentLeader   []byte
	validatorSet    [][]byte
	quorumSize      int

	// Proposals and votes
	proposals       map[string]*types.Proposal
	votes          map[string][]*types.Vote
	certificates   map[string]*types.Certificate

	// Wave status
	status         types.WaveStatus
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewWaveState creates a new wave state manager
func NewWaveState(ctx context.Context, config *types.ConsensusConfig) (*WaveState, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	childCtx, cancel := context.WithCancel(ctx)

	// Convert string validators to byte slices
	validatorSet := make([][]byte, len(config.ValidatorSet))
	for i, v := range config.ValidatorSet {
		validatorSet[i] = []byte(v)
	}

	ws := &WaveState{
		currentWave:    0,
		waveTimeout:    config.WaveTimeout,
		roundDuration:  config.RoundDuration,
		validatorSet:   validatorSet,
		quorumSize:     config.QuorumSize,
		proposals:      make(map[string]*types.Proposal),
		votes:         make(map[string][]*types.Vote),
		certificates:  make(map[string]*types.Certificate),
		status:        types.WaveStatusInitializing,
		ctx:           childCtx,
		cancel:        cancel,
	}

	return ws, nil
}

// StartWave begins a new wave
func (ws *WaveState) StartWave() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.status != types.WaveStatusInitializing && ws.status != types.WaveStatusCompleted {
		return fmt.Errorf("cannot start wave in current status: %v", ws.status)
	}

	ws.currentWave++
	ws.waveStartTime = time.Now()
	ws.waveEndTime = ws.waveStartTime.Add(ws.waveTimeout)
	ws.status = types.WaveStatusActive

	// Select leader for this wave
	leader, err := ws.selectLeader()
	if err != nil {
		return fmt.Errorf("failed to select leader: %v", err)
	}
	ws.currentLeader = leader

	// Clear previous wave data
	ws.proposals = make(map[string]*types.Proposal)
	ws.votes = make(map[string][]*types.Vote)
	ws.certificates = make(map[string]*types.Certificate)

	return nil
}

// AddProposal adds a new proposal to the current wave
func (ws *WaveState) AddProposal(proposal *types.Proposal) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.status != types.WaveStatusActive {
		return fmt.Errorf("cannot add proposal in current status: %v", ws.status)
	}

	if proposal.Wave != ws.currentWave {
		return fmt.Errorf("proposal wave mismatch: got %d, want %d", proposal.Wave, ws.currentWave)
	}

	if string(proposal.Proposer) != string(ws.currentLeader) {
		return fmt.Errorf("proposal from non-leader")
	}

	ws.proposals[string(proposal.ID)] = proposal
	return nil
}

// AddVote adds a new vote to the current wave
func (ws *WaveState) AddVote(vote *types.Vote) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.status != types.WaveStatusActive {
		return fmt.Errorf("cannot add vote in current status: %v", ws.status)
	}

	if vote.Wave != ws.currentWave {
		return fmt.Errorf("vote wave mismatch: got %d, want %d", vote.Wave, ws.currentWave)
	}

	// Verify the voter is in the validator set
	validValidator := false
	for _, validator := range ws.validatorSet {
		if string(validator) == string(vote.Validator) {
			validValidator = true
			break
		}
	}
	if !validValidator {
		return fmt.Errorf("vote from invalid validator")
	}

	proposalID := string(vote.ProposalID)
	ws.votes[proposalID] = append(ws.votes[proposalID], vote)

	// Check if we have enough votes for a certificate
	if len(ws.votes[proposalID]) >= ws.quorumSize {
		cert := &types.Certificate{
			BlockHash:    vote.ProposalID,
			Wave:        ws.currentWave,
			Round:       vote.Round,
			ValidatorSet: ws.validatorSet,
			Timestamp:   time.Now(),
		}
		// Collect signatures from votes
		signatures := make([][]byte, 0, len(ws.votes[proposalID]))
		for _, v := range ws.votes[proposalID] {
			signatures = append(signatures, v.Validator) // Using validator ID as signature for now
		}
		cert.Signatures = signatures
		ws.certificates[proposalID] = cert
		ws.status = types.WaveStatusCompleting
	}

	return nil
}

// GetCertificate returns the certificate for a given block hash if available
func (ws *WaveState) GetCertificate(blockHash []byte) (*types.Certificate, bool) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	cert, ok := ws.certificates[string(blockHash)]
	return cert, ok
}

// CompleteWave finalizes the current wave
func (ws *WaveState) CompleteWave() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.status != types.WaveStatusCompleting {
		return fmt.Errorf("cannot complete wave in current status: %v", ws.status)
	}

	ws.status = types.WaveStatusCompleted
	return nil
}

// selectLeader chooses a leader for the current wave
func (ws *WaveState) selectLeader() ([]byte, error) {
	if len(ws.validatorSet) == 0 {
		return nil, fmt.Errorf("empty validator set")
	}

	// Simple round-robin leader selection based on wave number
	leaderIndex := ws.currentWave % uint64(len(ws.validatorSet))
	return ws.validatorSet[leaderIndex], nil
}

// Stop stops the wave state manager
func (ws *WaveState) Stop() {
	ws.cancel()
}

// Status returns the current wave status
func (ws *WaveState) Status() types.WaveStatus {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.status
}

// CurrentWave returns the current wave number
func (ws *WaveState) CurrentWave() uint64 {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.currentWave
}

// CurrentLeader returns the current wave leader
func (ws *WaveState) CurrentLeader() []byte {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.currentLeader
} 