package network

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/types"
)

var (
	ErrServiceAlreadyRunning = errors.New("service is already running")
	ErrPeerNotFound          = errors.New("peer not found")
)

// Config represents the network service configuration
type Config struct {
	ListenAddr    string
	Seeds         []string
	MaxPeers      int
	MinPeers      int
	MessageBuffer int
	SyncInterval  time.Duration
}

// Service represents the network service
type Service struct {
	config *Config
	layer  *NetworkLayer

	peers    map[string]*Peer
	messages map[string]*types.NetworkMessage

	running bool
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	wg     sync.WaitGroup
}

// NewService creates a new network service
func NewService(config *Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		config:   config,
		layer:    NewNetworkLayer(),
		peers:    make(map[string]*Peer),
		messages: make(map[string]*types.NetworkMessage),
		running:  false,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the network service
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrServiceAlreadyRunning
	}

	// Start network layer
	if err := s.layer.Start(); err != nil {
		return err
	}

	// Connect to seed nodes
	for _, seed := range s.config.Seeds {
		peer := &Peer{
			ID:        seed,
			Addr:      seed,
			Connected: true,
			LastSeen:  time.Now(),
		}
		s.AddPeer(peer)
	}

	s.running = true
	// Start peer discovery
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.discoverPeers(s.ctx)
	}()

	return nil
}

// Stop stops the network service
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	// Cancel the context to stop all goroutines
	s.cancel()

	// Stop the network layer
	if err := s.layer.Stop(); err != nil {
		return err
	}

	// Clear peers and messages
	s.peers = make(map[string]*Peer)
	s.messages = make(map[string]*types.NetworkMessage)
	s.running = false

	// Wait for all goroutines to finish
	s.wg.Wait()

	return nil
}

// AddPeer adds a peer to the network service
func (s *Service) AddPeer(peer *Peer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.peers) >= s.config.MaxPeers {
		return
	}

	peer.LastSeen = time.Now()
	s.peers[peer.ID] = peer
	s.layer.AddPeer(peer)
}

// RemovePeer removes a peer from the network service
func (s *Service) RemovePeer(peerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.peers, peerID)
	s.layer.RemovePeer(peerID)
}

// GetPeers returns all peers in the network service
func (s *Service) GetPeers() []*Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peers := make([]*Peer, 0, len(s.peers))
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	return peers
}

// BroadcastMessage broadcasts a message to all peers
func (s *Service) BroadcastMessage(msg *types.NetworkMessage) error {
	return s.layer.Broadcast(msg)
}

// discoverPeers periodically discovers new peers
func (s *Service) discoverPeers(ctx context.Context) {
	ticker := time.NewTicker(s.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Create a timeout context for this iteration
			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			
			// Process peers in a goroutine
			done := make(chan struct{})
			go func() {
				defer close(done)
				for _, seed := range s.config.Seeds {
					select {
					case <-timeoutCtx.Done():
						return
					default:
						// Try to connect to seed
						peer := &Peer{
							ID:   seed,
							Addr: seed,
						}
						s.AddPeer(peer)
					}
				}
			}()

			// Wait for either completion or timeout
			select {
			case <-done:
				// Peer processing completed
			case <-timeoutCtx.Done():
				// Timeout or context cancellation
			}
			cancel()
		}
	}
}

// SyncState represents the synchronization state
type SyncState struct {
	LastSyncTime time.Time
	SyncHeight   uint64
	Syncing      bool
}

// SendMessage sends a message to a specific peer
func (s *Service) SendMessage(peerID string, msg *types.NetworkMessage) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return errors.New("service is not running")
	}

	peer, exists := s.peers[peerID]
	if !exists {
		return errors.New("peer not found")
	}

	msg.Sender = s.config.ListenAddr
	msg.Recipient = peerID
	msg.Timestamp = time.Now()

	if err := s.layer.SendMessage(peer, msg); err != nil {
		return err
	}

	s.messages[string(msg.Signature)] = msg
	return nil
}

// GetMessage retrieves a message by its signature
func (s *Service) GetMessage(signature []byte) (*types.NetworkMessage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg, exists := s.messages[string(signature)]
	return msg, exists
}

// Message handlers
func (s *Service) handleBlockMessage(msg *types.NetworkMessage) {
	// TODO: Implement block message handling
}

func (s *Service) handleVoteMessage(msg *types.NetworkMessage) {
	// TODO: Implement vote message handling
}

func (s *Service) handleCertificateMessage(msg *types.NetworkMessage) {
	// TODO: Implement certificate message handling
}

func (s *Service) handleComplaintMessage(msg *types.NetworkMessage) {
	// TODO: Implement complaint message handling
}

func (s *Service) handleSyncRequest(msg *types.NetworkMessage) {
	// TODO: Implement sync request handling
}

func (s *Service) handleSyncResponse(msg *types.NetworkMessage) {
	// TODO: Implement sync response handling
}

func (s *Service) handleRecoveryRequest(msg *types.NetworkMessage) {
	// TODO: Implement recovery request handling
}

func (s *Service) handleRecoveryResponse(msg *types.NetworkMessage) {
	// TODO: Implement recovery response handling
}

// PeerScore represents the scoring metrics for a peer
type PeerScore struct {
	Latency       time.Duration
	Uptime        time.Duration
	MessageCount  int
	SuccessRate   float64
	LastUpdated   time.Time
}

// OptimizePeerConnections dynamically optimizes the peer connections based on performance metrics
func (s *Service) OptimizePeerConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	// Initialize peer scores if not exists
	peerScores := make(map[string]*PeerScore)
	for _, peer := range s.peers {
		if _, exists := peerScores[peer.ID]; !exists {
			peerScores[peer.ID] = &PeerScore{
				LastUpdated: time.Now(),
			}
		}
	}

	// Calculate scores for each peer
	for peerID, score := range peerScores {
		peer := s.peers[peerID]
		
		// Update latency (simplified for example)
		score.Latency = time.Since(peer.LastSeen)
		
		// Update uptime
		if peer.Connected {
			score.Uptime += time.Since(score.LastUpdated)
		}
		
		// Update message count and success rate
		// This would be implemented with actual message tracking
		score.MessageCount++
		score.SuccessRate = 0.95 // Example value
		
		score.LastUpdated = time.Now()
	}

	// Sort peers by score
	type scoredPeer struct {
		ID    string
		Score float64
	}
	
	scoredPeers := make([]scoredPeer, 0, len(peerScores))
	for peerID, score := range peerScores {
		// Calculate composite score
		compositeScore := float64(score.MessageCount) * score.SuccessRate * 
			(1.0 - float64(score.Latency)/float64(time.Minute)) * 
			(float64(score.Uptime) / float64(time.Hour))
			
		scoredPeers = append(scoredPeers, scoredPeer{
			ID:    peerID,
			Score: compositeScore,
		})
	}

	// Sort by score in descending order
	sort.Slice(scoredPeers, func(i, j int) bool {
		return scoredPeers[i].Score > scoredPeers[j].Score
	})

	// Maintain optimal number of connections
	optimalPeers := s.config.MaxPeers
	if len(scoredPeers) > optimalPeers {
		// Disconnect from lowest scoring peers
		for i := optimalPeers; i < len(scoredPeers); i++ {
			s.RemovePeer(scoredPeers[i].ID)
		}
	} else if len(scoredPeers) < s.config.MinPeers {
		// Connect to more peers if below minimum
		for _, seed := range s.config.Seeds {
			if len(s.peers) >= s.config.MaxPeers {
				break
			}
			if _, exists := s.peers[seed]; !exists {
				peer := &Peer{
					ID:   seed,
					Addr: seed,
				}
				s.AddPeer(peer)
			}
		}
	}
}