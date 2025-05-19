package consensus

import (
	"bytes"
	"encoding/gob"
	"log"
	"net"
	"strconv"
	"sync"
)

// SyncServer handles incoming block synchronization requests
type SyncServer struct {
	blockSynchronizer *BlockSynchronizer
	listener          net.Listener
	port              int
	logger            *log.Logger
	stopChan          chan struct{}
	mu                sync.RWMutex
	running           bool
}

// NewSyncServer creates a new sync server
func NewSyncServer(bs *BlockSynchronizer, port int) *SyncServer {
	return &SyncServer{
		blockSynchronizer: bs,
		port:              port,
		logger:            log.New(log.Writer(), "[SyncServer] ", log.LstdFlags),
		stopChan:          make(chan struct{}),
	}
}

// Start starts the sync server
func (ss *SyncServer) Start() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.running {
		return nil
	}

	addr := ":" + strconv.Itoa(ss.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	ss.listener = listener
	ss.running = true

	go ss.acceptConnections()

	ss.logger.Printf("Sync server started on port %d", ss.port)
	return nil
}

// Stop stops the sync server
func (ss *SyncServer) Stop() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !ss.running {
		return
	}

	close(ss.stopChan)
	if ss.listener != nil {
		ss.listener.Close()
	}
	ss.running = false

	ss.logger.Printf("Sync server stopped")
}

// acceptConnections accepts incoming connections
func (ss *SyncServer) acceptConnections() {
	for {
		select {
		case <-ss.stopChan:
			return
		default:
			conn, err := ss.listener.Accept()
			if err != nil {
				if ss.running {
					ss.logger.Printf("Error accepting connection: %v", err)
				}
				continue
			}

			go ss.handleConnection(conn)
		}
	}
}

// handleConnection handles an incoming connection
func (ss *SyncServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read the sync request
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		ss.logger.Printf("Error reading sync request: %v", err)
		return
	}

	// Decode the request
	var req SyncRequest
	decoder := gob.NewDecoder(bytes.NewReader(buf[:n]))
	if err := decoder.Decode(&req); err != nil {
		ss.logger.Printf("Error decoding sync request: %v", err)
		return
	}

	// Handle the request
	ss.blockSynchronizer.HandleSyncRequest(&req, conn)
}
