package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/api"
	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/storage"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

// CLI represents the command line interface
type CLI struct {
	config          *consensus.Config
	stateManager    *state.StateManager
	blockProcessor  *core.BlockProcessor
	consensusEngine *consensus.ConsensusEngine
	// consensus       *consensus.Consensus
	logger           *log.Logger
	scanner          *bufio.Scanner
	stopChan         chan struct{}
	currentRound     int
	approvedBlocks   map[string]bool // Changed from types.Hash to string
	savedLeaderBlock *types.Block    // Add this field to store the wave leader's block
	apiServer        *api.Server
}

// NewCLI creates a new CLI instance
func NewCLI(config *consensus.Config) *CLI {
	return &CLI{
		config:         config,
		scanner:        bufio.NewScanner(os.Stdin),
		logger:         log.New(os.Stdout, "", log.LstdFlags),
		stopChan:       make(chan struct{}),
		approvedBlocks: make(map[string]bool),
	}
}

// Start starts the CLI
func (c *CLI) Start() error {
	// Initialize components
	if err := c.initialize(); err != nil {
		return fmt.Errorf("failed to initialize: %v", err)
	}

	// Start consensus engine (no block creation here)
	if err := c.consensusEngine.Start(); err != nil {
		return fmt.Errorf("failed to start consensus engine: %v", err)
	}

	// Create and start API server
	apiPort := 8080
	c.apiServer = api.NewServer(c.consensusEngine, apiPort)
	if err := c.apiServer.Start(); err != nil {
		return fmt.Errorf("failed to start API server: %v", err)
	}
	c.logger.Printf("API server started on port %d", apiPort)

	go c.runChain()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		c.Stop()
		os.Exit(0)
	}()

	// Start CLI loop
	c.logger.Printf("BlazeDAG CLI - Type 'help' for available commands")
	return c.run()
}

// runChain runs the chain with round and wave forwarding
func (c *CLI) runChain() {
	round := 1
	height := types.BlockNumber(0)
	lastWave := types.Wave(1) // Start from wave 1
	blockCreatedInWave := false

	// Initial delay to allow synchronization to start
	log.Printf("Waiting for initial synchronization...")
	time.Sleep(10 * time.Second)

	// Start a goroutine to listen for incoming votes
	go func() {
		// Convert ListenAddr to string and extract port
		listenAddr := string(c.config.ListenAddr)
		_, port, err := net.SplitHostPort(listenAddr)
		if err != nil {
			log.Printf("Error parsing listen address: %v", err)
			return
		}

		// Convert port to number, add offset, and convert back to string
		portNum, err := strconv.Atoi(port)
		if err != nil {
			log.Printf("Error converting port to number: %v", err)
			return
		}
		votePort := strconv.Itoa(portNum + 1000) // Add 1000 to the main port for vote listening

		voteListener, err := net.Listen("tcp", ":"+votePort)
		if err != nil {
			log.Printf("Error starting vote listener: %v", err)
			return
		}
		defer voteListener.Close()

		log.Printf("Vote listener started on port %s", votePort)

		// Give other validators time to start
		time.Sleep(2 * time.Second)

		for {
			conn, err := voteListener.Accept()
			if err != nil {
				log.Printf("Error accepting vote connection: %v", err)
				continue
			}

			go func(conn net.Conn) {
				defer conn.Close()

				// Set a longer read deadline for the initial connection
				conn.SetReadDeadline(time.Now().Add(30 * time.Second))

				// Read the vote message with a buffer
				buf := make([]byte, 4096) // 4KB buffer
				n, err := conn.Read(buf)
				if err != nil {
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						log.Printf("Timeout reading vote message: %v", err)
					} else {
						log.Printf("Error reading vote message: %v", err)
					}
					return
				}

				// Create a buffer with just the data we read
				voteBuf := bytes.NewBuffer(buf[:n])

				// Decode the vote
				var vote types.Vote
				decoder := gob.NewDecoder(voteBuf)
				if err := decoder.Decode(&vote); err != nil {
					log.Printf("Error decoding vote: %v", err)
					return
				}

				fmt.Println("++++++++++++++++++++++++++++")
				fmt.Println(vote)
				fmt.Println("++++++++++++++++++++++++++++")

				// Handle the vote
				if err := c.consensusEngine.HandleVote(&vote); err != nil {
					log.Printf("Error handling vote: %v", err)
					return
				}

				// Set a new deadline for sending acknowledgment
				conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

				// Send acknowledgment
				ack := []byte("ACK")
				if _, err := conn.Write(ack); err != nil {
					log.Printf("Error sending vote acknowledgment: %v", err)
					return
				}

				log.Printf("Successfully processed vote from validator %s for block %x",
					vote.Validator, vote.BlockHash)
			}(conn)
		}
	}()

	// Give other validators time to start
	time.Sleep(2 * time.Second)

	for {
		select {
		case <-c.stopChan:
			return
		default:
			// Check DAG state every 10 rounds
			if round%10 == 0 {
				dag := core.GetDAG() // Get the singleton DAG
				blocks := dag.GetRecentBlocks(20)

				// Count blocks by validator
				validatorBlocks := make(map[string]int)
				for _, block := range blocks {
					validator := string(block.Header.Validator)
					validatorBlocks[validator]++
				}

				// Log blocks by validator
				log.Printf("\n=== Current DAG State ===")
				log.Printf("Total blocks: %d", len(blocks))
				for validator, count := range validatorBlocks {
					log.Printf("  Validator %s: %d blocks", validator, count)
				}
				log.Printf("=== End DAG State ===\n")
			}

			// Get current wave from consensus engine
			currentWave := c.consensusEngine.GetCurrentWave()
			// Only create block if we're in a new wave and haven't created a block yet
			if currentWave != lastWave {
				blockCreatedInWave = false
				lastWave = currentWave
				c.savedLeaderBlock = nil // Clear saved block on wave change
			}

			if !blockCreatedInWave {
				// Create a block with current round number using consensus engine
				block, err := c.consensusEngine.CreateBlock()
				if err != nil {
					log.Printf("Error creating block: %v", err)
					continue
				}
				// Broadcast the block using consensus engine
				if err := c.consensusEngine.BroadcastBlock(block); err != nil {
					log.Printf("Error broadcasting block: %v", err)
					continue
				}

				// Only proceed with voting if we have a saved leader block
				if c.consensusEngine.GetSavedLeaderBlock() != nil {
					fmt.Println("jjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjj")
					// Create and broadcast vote for this block
					vote := &types.Vote{
						BlockHash: c.consensusEngine.GetSavedLeaderBlock().ComputeHash(),
						Round:     types.Round(round),
						Block:     c.consensusEngine.GetSavedLeaderBlock(),
						Wave:      currentWave,
						Validator: types.Address(c.config.NodeID),
						Timestamp: time.Now(),
					}

					// Broadcast vote to all validators
					for _, validator := range c.consensusEngine.GetValidators() {
						if string(validator) != c.config.NodeID { // Don't send to self
							// Get validator's address from config
							var validatorAddr string
							switch string(validator) {
							case "validator1":
								validatorAddr = "localhost:3000"
							case "validator2":
								validatorAddr = "localhost:3001"
							case "validator3":
								validatorAddr = "localhost:3002"
							default:
								log.Printf("Warning: Unknown validator %s", validator)
								continue
							}

							// Extract port and add 1000 for vote port
							_, port, err := net.SplitHostPort(validatorAddr)
							if err != nil {
								log.Printf("Error parsing validator address: %v", err)
								continue
							}
							portNum, err := strconv.Atoi(port)
							if err != nil {
								log.Printf("Error converting port to number: %v", err)
								continue
							}
							votePort := strconv.Itoa(portNum + 1000)
							voteAddr := "localhost:" + votePort

							log.Printf("Broadcasting vote for block %x to validator %s at %s",
								vote.BlockHash, validator, voteAddr)

							// Try to connect with retries
							var conn net.Conn
							maxRetries := 3
							for i := 0; i < maxRetries; i++ {
								conn, err = net.Dial("tcp", voteAddr)
								if err == nil {
									break
								}
								log.Printf("Retry %d: Error connecting to validator %s at %s: %v",
									i+1, validator, voteAddr, err)
								time.Sleep(time.Second)
							}
							if err != nil {
								log.Printf("Failed to connect to validator %s after %d retries",
									validator, maxRetries)
								continue
							}
							defer conn.Close()

							// Set write deadline
							conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

							// Create a buffer to hold the encoded vote
							var buf bytes.Buffer
							encoder := gob.NewEncoder(&buf)
							if err := encoder.Encode(vote); err != nil {
								log.Printf("Error encoding vote for validator %s: %v", validator, err)
								continue
							}

							// Write the encoded vote to the connection
							if _, err := conn.Write(buf.Bytes()); err != nil {
								log.Printf("Error sending vote to validator %s: %v", validator, err)
								continue
							}

							// Wait for acknowledgment with a longer timeout
							ackBuf := make([]byte, 3) // "ACK" is 3 bytes
							conn.SetReadDeadline(time.Now().Add(10 * time.Second))
							if _, err := conn.Read(ackBuf); err != nil {
								if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
									log.Printf("Timeout waiting for acknowledgment from validator %s", validator)
								} else {
									log.Printf("Error receiving acknowledgment from validator %s: %v", validator, err)
								}
								continue
							}

							if !bytes.Equal(ackBuf, []byte("ACK")) {
								log.Printf("Invalid acknowledgment from validator %s", validator)
								continue
							}

							log.Printf("Successfully sent vote to validator %s", validator)
						}
					}

					// Also handle the vote locally
					if err := c.consensusEngine.HandleVote(vote); err != nil {
						log.Printf("Error handling local vote: %v", err)
					}

					// Show leader selection when wave changes
					log.Printf("\n======================== Wave %d Leader Selection =========================", currentWave)
					// Update current round and mark block as created
					c.currentRound = round
					blockCreatedInWave = true
					// Increment round and height
					round++
					height++
				}
			}

			// Sleep for the configured round duration
			time.Sleep(c.config.RoundDuration)
		}
	}
}

// Stop stops the CLI
func (c *CLI) Stop() {
	close(c.stopChan)
	if c.consensusEngine != nil {
		c.consensusEngine.Stop()
	}

	// Stop API server if running
	if c.apiServer != nil {
		c.apiServer.Stop()
	}
}

// initialize initializes the CLI components
func (c *CLI) initialize() error {
	// Initialize state manager
	c.stateManager = state.NewStateManager()

	// Create block processor config
	blockConfig := &core.Config{
		BlockInterval:    1 * time.Second,
		ConsensusTimeout: 5 * time.Second,
		IsValidator:      true,
		NodeID:           types.Address(c.config.NodeID),
	}

	// Use singleton DAG
	dag := core.GetDAG()

	// Create initial state
	initialState := types.NewState()

	// Create storage
	storage, err := storage.NewStorage("data")
	if err != nil {
		return fmt.Errorf("failed to create storage: %v", err)
	}

	// Create state manager
	stateManager := core.NewStateManager(initialState, storage)

	// Initialize block processor
	c.blockProcessor = core.NewBlockProcessor(blockConfig, stateManager, dag)

	// Initialize consensus engine
	c.consensusEngine = consensus.NewConsensusEngine(c.config, c.stateManager, c.blockProcessor)

	// Initialize consensus
	// c.consensus = consensus.NewConsensus(c.config, initialState, storage)

	return nil
}

// run runs the CLI loop
func (c *CLI) run() error {
	for {
		// Read command
		var cmd string
		fmt.Print("> ")
		if !c.scanner.Scan() {
			continue
		}

		input := c.scanner.Text()
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		cmd = parts[0]
		args := parts[1:]

		// Handle command
		switch cmd {
		case "help":
			c.printHelp()
		case "status":
			c.handleStatus()
		case "propose":
			c.handlePropose()
		case "vote":
			c.handleVote()
		case "blocks":
			c.handleBlocks()
		case "block":
			c.handleBlock()
		case "send":
			if err := c.handleSend(args); err != nil {
				c.logger.Printf("Error: %v", err)
			}
		case "showblocks":
			c.showBlocks()
		case "exit":
			c.Stop()
			return nil
		default:
			c.logger.Printf("Unknown command: %s", cmd)
		}
	}
}

// printHelp prints the help message
func (c *CLI) printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help    - Show this help message")
	fmt.Println("  status  - Show current status")
	fmt.Println("  propose - Propose a new block")
	fmt.Println("  vote    - Vote on a proposal")
	fmt.Println("  blocks  - List recent blocks")
	fmt.Println("  block   - Show block details")
	fmt.Println("  send    - Send a transaction (send <from> <to> <amount>)")
	fmt.Println("  showblocks - Show all blocks in the chain")
	fmt.Println("  exit    - Exit the CLI")
	fmt.Println("\nAPI server running on port 8080 with the following endpoints:")
	fmt.Println("  GET /status         - Show node status")
	fmt.Println("  GET /blocks         - List recent blocks")
	fmt.Println("  GET /blocks/{hash}  - Show block details")
	fmt.Println("  GET /dag/stats      - Show DAG statistics")
	fmt.Println("  GET /consensus/wave - Show current wave information")
	fmt.Println("  GET /transactions   - List recent transactions")
	fmt.Println("  GET /validators     - List validators")
}

// handleStatus shows the current status
func (c *CLI) handleStatus() error {
	fmt.Printf("Current wave: %d\n", c.consensusEngine.GetCurrentWave())
	fmt.Printf("Current round: %d\n", c.currentRound)
	fmt.Printf("Is leader: %v\n", c.consensusEngine.IsLeader())
	fmt.Printf("Node ID: %s\n", c.consensusEngine.GetNodeID())
	fmt.Printf("API server: running on port 8080\n")
	return nil
}

// handlePropose handles the propose command
func (c *CLI) handlePropose() error {
	if !c.consensusEngine.IsLeader() {
		return fmt.Errorf("only the leader can propose blocks")
	}

	block, err := c.consensusEngine.CreateBlock()
	if err != nil {
		return fmt.Errorf("failed to create block: %v", err)
	}

	if err := c.consensusEngine.BroadcastBlock(block); err != nil {
		return fmt.Errorf("failed to broadcast block: %v", err)
	}

	fmt.Println("Block proposed successfully")
	return nil
}

// handleVote handles the vote command
func (c *CLI) handleVote() error {
	if !c.consensusEngine.IsLeader() {
		return fmt.Errorf("only the leader can vote")
	}

	proposalID := c.config.NodeID
	vote := &types.Vote{
		ProposalID: types.Hash(proposalID),
		Validator:  types.Address(c.config.NodeID),
		Timestamp:  time.Now(),
	}

	if err := c.consensusEngine.HandleVote(vote); err != nil {
		return fmt.Errorf("failed to handle vote: %v", err)
	}

	fmt.Println("Vote submitted successfully")
	return nil
}

// handleBlocks handles the blocks command
func (c *CLI) handleBlocks() error {
	count := 10 // Default to showing 10 most recent blocks
	blocks := c.consensusEngine.GetRecentBlocks(count)
	if len(blocks) == 0 {
		fmt.Println("No blocks found")
		return nil
	}

	// Use a map to track unique block hashes
	seenBlocks := make(map[string]bool)
	uniqueBlocks := make([]*types.Block, 0)

	for _, block := range blocks {
		blockHash := fmt.Sprintf("%x", block.ComputeHash())
		if !seenBlocks[blockHash] {
			seenBlocks[blockHash] = true
			uniqueBlocks = append(uniqueBlocks, block)
		}
	}

	// Group blocks by validator for display
	validatorBlocks := make(map[string][]*types.Block)
	for _, block := range uniqueBlocks {
		validator := string(block.Header.Validator)
		validatorBlocks[validator] = append(validatorBlocks[validator], block)
	}

	// Print summary of blocks by validator
	fmt.Println("\nBlocks summary by validator:")
	for validator, blocks := range validatorBlocks {
		fmt.Printf("  Validator %s: %d blocks\n", validator, len(blocks))
	}

	fmt.Printf("\nRecent blocks (showing %d):\n", len(uniqueBlocks))
	for _, block := range uniqueBlocks {
		blockHash := block.ComputeHash()
		isApproved := c.approvedBlocks[string(blockHash)]
		votes := c.consensusEngine.GetBlockVotes(blockHash)

		// Format the block hash as a hex string
		hashStr := fmt.Sprintf("%x", blockHash)

		fmt.Printf("Height: %d, Wave: %d, Round: %d, Hash: %s, Validator: %s, Approved: %v, Votes: %d/%d\n",
			block.Header.Height,
			block.Header.Wave,
			block.Header.Round,
			hashStr,
			block.Header.Validator,
			isApproved,
			votes,
			c.config.QuorumSize)
	}
	return nil
}

// handleBlock handles the block command
func (c *CLI) handleBlock() error {
	if !c.consensusEngine.IsLeader() {
		return fmt.Errorf("only the leader can show block details")
	}

	block, err := c.consensusEngine.GetBlock(types.Hash(string(c.config.NodeID)))
	if err != nil {
		return fmt.Errorf("failed to get block: %v", err)
	}

	fmt.Printf("Block Details:\n")
	fmt.Printf("  Height: %d\n", block.Header.Height)
	fmt.Printf("  Hash: %s\n", block.ComputeHash())
	fmt.Printf("  Validator: %s\n", block.Header.Validator)
	fmt.Printf("  Timestamp: %s\n", block.Header.Timestamp)
	fmt.Printf("  Parent Hash: %s\n", block.Header.ParentHash)
	fmt.Printf("  Transactions: %d\n", len(block.Body.Transactions))
	fmt.Printf("  References: %d\n", len(block.Header.References))
	return nil
}

// handleSend handles the send command
func (c *CLI) handleSend(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: send <from> <to> <amount>")
	}

	from := types.Address(args[0])
	to := types.Address(args[1])
	amount, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %v", err)
	}

	tx := &types.Transaction{
		From:      from,
		To:        to,
		Value:     types.Value(amount),
		Nonce:     0, // TODO: Get actual nonce from state
		GasLimit:  21000,
		GasPrice:  1,
		Data:      nil,
		Timestamp: time.Now(),
	}

	c.blockProcessor.AddTransaction(tx)

	// Print detailed transaction information
	fmt.Printf("\nTransaction Details:\n")
	fmt.Printf("  Hash: %x\n", tx.GetHash())
	fmt.Printf("  From: %s\n", tx.From)
	fmt.Printf("  To: %s\n", tx.To)
	fmt.Printf("  Value: %d\n", tx.Value)
	fmt.Printf("  Nonce: %d\n", tx.Nonce)
	fmt.Printf("  Gas Limit: %d\n", tx.GasLimit)
	fmt.Printf("  Gas Price: %d\n", tx.GasPrice)
	fmt.Printf("  Timestamp: %s\n", tx.Timestamp.Format(time.RFC3339))
	fmt.Printf("  Data: %v\n", tx.Data)
	fmt.Printf("\nTransaction added to mempool successfully!\n")
	return nil
}

// showBlocks shows all blocks in the chain
func (c *CLI) showBlocks() error {
	blocks, err := c.stateManager.GetAllBlocks()
	if err != nil {
		return fmt.Errorf("failed to get blocks: %v", err)
	}

	if len(blocks) == 0 {
		fmt.Println("No blocks found in the chain")
		return nil
	}

	fmt.Printf("Found %d blocks in the chain:\n\n", len(blocks))
	for _, block := range blocks {
		fmt.Printf("Block %s:\n", block.ComputeHash())
		fmt.Printf("  Height: %d\n", block.Header.Height)
		fmt.Printf("  Validator: %s\n", block.Header.Validator)
		fmt.Printf("  Timestamp: %s\n", block.Header.Timestamp)
		fmt.Printf("  Parent Hash: %s\n", block.Header.ParentHash)
		fmt.Printf("  Transactions: %d\n", len(block.Body.Transactions))
		fmt.Println()
	}

	return nil
}
