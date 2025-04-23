package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/network"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
)

const (
	dataDir = "chaindata"
	accountsFile = "accounts.json"
	blocksFile = "blocks.json"
	stateFile = "state.json"
	logFile = "chain.log"
)

type CLI struct {
	node            *network.P2PNode
	msgHandler      *network.MessageHandler
	dag             *core.DAG
	stateManager    *state.StateManager
	evm             *core.EVMExecutor
	consensusEngine *consensus.ConsensusEngine
	waveController  *consensus.WaveController
	accounts        map[string]*types.Account
	currentWave     uint64
	isValidator     bool
	dataDir         string
	logger          *log.Logger
	mempool         *core.Mempool
	nodeID          types.Address
}

func NewCLI(isValidator bool, nodeID types.Address) *CLI {
	return &CLI{
		accounts:    make(map[string]*types.Account),
		currentWave: 0,
		dataDir:     dataDir,
		mempool:     core.NewMempool(),
		isValidator: isValidator,
		nodeID:      nodeID,
	}
}

func (cli *CLI) Start(port int) error {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(cli.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Setup logging
	logPath := filepath.Join(cli.dataDir, logFile)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	cli.logger = log.New(logFile, "", log.LstdFlags)
	cli.logger.Printf("Starting BlazeDAG node on port %d (Validator: %v)", port, cli.isValidator)

	// Load saved state
	if err := cli.loadState(); err != nil {
		cli.logger.Printf("Warning: Failed to load saved state: %v", err)
	}

	// Create P2P node
	nodeID := network.PeerID(fmt.Sprintf("node-%d", port))
	node := network.NewP2PNode(nodeID)
	node.Port = port  // Set the port
	if err := node.Start(); err != nil {
		return fmt.Errorf("failed to start P2P node: %v", err)
	}
	cli.node = node
	cli.logger.Printf("P2P node started with ID: %s", nodeID)

	// Create message handler
	msgHandler := network.NewMessageHandler(node)
	if err := msgHandler.Start(); err != nil {
		return fmt.Errorf("failed to start message handler: %v", err)
	}
	cli.msgHandler = msgHandler
	cli.logger.Printf("Message handler started")

	// Create DAG
	dag := core.NewDAG()
	cli.dag = dag
	cli.logger.Printf("DAG initialized")

	// Create state manager
	stateManager := state.NewStateManager()
	cli.stateManager = stateManager
	cli.logger.Printf("State manager initialized")

	// Create EVM executor
	evm := core.NewEVMExecutor(stateManager)
	cli.evm = evm
	cli.logger.Printf("EVM executor initialized")

	// Create consensus engine
	cli.consensusEngine = consensus.NewConsensusEngine(
		cli.dag,
		cli.stateManager,
		cli.evm,
		cli.isValidator,
		cli.nodeID,
	)
	if err := cli.consensusEngine.Start(); err != nil {
		return fmt.Errorf("failed to start consensus engine: %v", err)
	}
	cli.logger.Printf("Consensus engine started")

	// Create wave controller
	waveController := consensus.NewWaveController(cli.consensusEngine)
	if err := waveController.Start(); err != nil {
		return fmt.Errorf("failed to start wave controller: %v", err)
	}
	cli.waveController = waveController
	cli.logger.Printf("Wave controller started")

	// Register message handlers
	cli.registerMessageHandlers()
	cli.logger.Printf("Message handlers registered")

	return nil
}

func (cli *CLI) loadState() error {
	// Load accounts
	accountsPath := filepath.Join(cli.dataDir, accountsFile)
	if data, err := os.ReadFile(accountsPath); err == nil {
		if err := json.Unmarshal(data, &cli.accounts); err != nil {
			return fmt.Errorf("failed to unmarshal accounts: %v", err)
		}
	}

	// Load blocks
	blocksPath := filepath.Join(cli.dataDir, blocksFile)
	if data, err := os.ReadFile(blocksPath); err == nil {
		var blocks []*types.Block
		if err := json.Unmarshal(data, &blocks); err != nil {
			return fmt.Errorf("failed to unmarshal blocks: %v", err)
		}
		for _, block := range blocks {
			if err := cli.dag.AddBlock(block); err != nil {
				return fmt.Errorf("failed to add block: %v", err)
			}
		}
	}

	// Load state
	statePath := filepath.Join(cli.dataDir, stateFile)
	if data, err := os.ReadFile(statePath); err == nil {
		var state struct {
			CurrentWave uint64 `json:"currentWave"`
		}
		if err := json.Unmarshal(data, &state); err != nil {
			return fmt.Errorf("failed to unmarshal state: %v", err)
		}
		cli.currentWave = state.CurrentWave
	}

	return nil
}

func (cli *CLI) saveState() error {
	// Save accounts
	accountsPath := filepath.Join(cli.dataDir, accountsFile)
	if data, err := json.Marshal(cli.accounts); err != nil {
		return fmt.Errorf("failed to marshal accounts: %v", err)
	} else if err := os.WriteFile(accountsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write accounts: %v", err)
	}

	// Save blocks
	blocksPath := filepath.Join(cli.dataDir, blocksFile)
	blocks := cli.dag.GetBlocks()
	if data, err := json.Marshal(blocks); err != nil {
		return fmt.Errorf("failed to marshal blocks: %v", err)
	} else if err := os.WriteFile(blocksPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write blocks: %v", err)
	}

	// Save state
	statePath := filepath.Join(cli.dataDir, stateFile)
	state := struct {
		CurrentWave uint64 `json:"currentWave"`
	}{
		CurrentWave: cli.currentWave,
	}
	if data, err := json.Marshal(state); err != nil {
		return fmt.Errorf("failed to marshal state: %v", err)
	} else if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state: %v", err)
	}

	return nil
}

func (cli *CLI) registerMessageHandlers() {
	cli.msgHandler.RegisterHandler(network.MessageTypeBlock, func(msg *network.Message) error {
		var block types.Block
		if err := msg.UnmarshalPayload(&block); err != nil {
			cli.logger.Printf("Error: Failed to unmarshal block: %v", err)
			return fmt.Errorf("failed to unmarshal block: %v", err)
		}
		cli.logger.Printf("Received block: %s (Height: %d, Wave: %d)", 
			block.ComputeHash(), block.Header.Height, block.Header.Wave)
		return cli.consensusEngine.HandleBlock(&block)
	})

	cli.msgHandler.RegisterHandler(network.MessageTypeVote, func(msg *network.Message) error {
		var vote types.Vote
		if err := msg.UnmarshalPayload(&vote); err != nil {
			cli.logger.Printf("Error: Failed to unmarshal vote: %v", err)
			return fmt.Errorf("failed to unmarshal vote: %v", err)
		}
		cli.logger.Printf("Received vote for block: %s from validator: %s", 
			hex.EncodeToString(vote.BlockHash), string(vote.Validator))
		return cli.consensusEngine.HandleVote(&vote)
	})

	cli.msgHandler.RegisterHandler(network.MessageTypeCertificate, func(msg *network.Message) error {
		var cert types.Certificate
		if err := msg.UnmarshalPayload(&cert); err != nil {
			cli.logger.Printf("Error: Failed to unmarshal certificate: %v", err)
			return fmt.Errorf("failed to unmarshal certificate: %v", err)
		}
		cli.logger.Printf("Received certificate for block: %s with %d signatures", 
			hex.EncodeToString(cert.BlockHash), len(cert.Signatures))
		return cli.consensusEngine.HandleCertificate(&cert)
	})
}

func (cli *CLI) Stop() {
	cli.logger.Printf("Shutting down node...")

	// Save state before stopping
	if err := cli.saveState(); err != nil {
		cli.logger.Printf("Warning: Failed to save state: %v", err)
	}

	if cli.waveController != nil {
		cli.waveController.Stop()
		cli.logger.Printf("Wave controller stopped")
	}
	if cli.consensusEngine != nil {
		cli.consensusEngine.Stop()
		cli.logger.Printf("Consensus engine stopped")
	}
	if cli.msgHandler != nil {
		cli.msgHandler.Stop()
		cli.logger.Printf("Message handler stopped")
	}
	if cli.node != nil {
		cli.node.Stop()
		cli.logger.Printf("P2P node stopped")
	}

	cli.logger.Printf("Node shutdown complete")
}

func (cli *CLI) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("BlazeDAG CLI - Type 'help' for available commands")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "help":
			cli.printHelp()
		case "status":
			cli.printStatus()
		case "peers":
			cli.printPeers()
		case "blocks":
			cli.printBlocks()
		case "block":
			if len(args) < 1 {
				fmt.Println("Usage: block <hash>")
				continue
			}
			cli.printBlockDetails(args[0])
		case "newaccount":
			cli.createNewAccount()
		case "accounts":
			cli.listAccounts()
		case "balance":
			if len(args) < 1 {
				fmt.Println("Usage: balance <address>")
				continue
			}
			cli.checkBalance(args[0])
		case "send":
			if len(args) < 3 {
				fmt.Println("Usage: send <from> <to> <amount>")
				continue
			}
			cli.sendTransaction(args[0], args[1], args[2])
		case "createblock":
			cli.createBlock()
		case "exit":
			return
		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}
}

func (cli *CLI) printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help        - Show this help message")
	fmt.Println("  status      - Show node status")
	fmt.Println("  peers       - List connected peers")
	fmt.Println("  blocks      - List recent blocks")
	fmt.Println("  block       - Show block details")
	fmt.Println("  newaccount  - Create a new account")
	fmt.Println("  accounts    - List all accounts")
	fmt.Println("  balance     - Check account balance")
	fmt.Println("  send        - Send a transaction")
	fmt.Println("  createblock - Create a new block")
	fmt.Println("  exit        - Exit the CLI")
}

func (cli *CLI) printStatus() {
	fmt.Printf("Node ID: %s\n", cli.node.GetID())
	fmt.Printf("Connected Peers: %d\n", len(cli.node.GetActivePeers()))
	fmt.Printf("Current Wave: %d\n", cli.currentWave)
	fmt.Printf("Is Validator: %v\n", cli.isValidator)
	fmt.Printf("Total Accounts: %d\n", len(cli.accounts))
}

func (cli *CLI) printPeers() {
	peers := cli.node.GetActivePeers()
	fmt.Printf("Connected Peers (%d):\n", len(peers))
	for _, peer := range peers {
		fmt.Printf("  - %s\n", peer)
	}
}

func (cli *CLI) printBlocks() {
	// Get the last 10 blocks from the DAG
	blocks := cli.dag.GetRecentBlocks(10)

	fmt.Printf("Recent Blocks (%d):\n", len(blocks))
	for _, block := range blocks {
		fmt.Printf("  - Hash: %s\n", block.ComputeHash())
		fmt.Printf("    Height: %d\n", block.Header.Height)
		fmt.Printf("    Wave: %d\n", block.Header.Wave)
		fmt.Printf("    Timestamp: %s\n", block.Timestamp.Format(time.RFC3339))
		fmt.Printf("    Transactions: %d\n", len(block.Body.Transactions))
		fmt.Printf("    Consensus Status: %s\n", cli.getConsensusStatus(block))
		fmt.Printf("    Confirmations: %d\n", cli.getConfirmations(block))
		fmt.Println()
	}
}

func (cli *CLI) getConsensusStatus(block *types.Block) string {
	// Check if block has a certificate
	if block.Certificate != nil && len(block.Certificate.Signatures) > 0 {
		return "Finalized"
	}
	return "Pending"
}

func (cli *CLI) getConfirmations(block *types.Block) int {
	// Count how many blocks have been added after this one
	currentHeight := cli.dag.GetHeight()
	if currentHeight < block.Header.Height {
		return 0
	}
	return int(currentHeight - block.Header.Height)
}

func (cli *CLI) printBlockDetails(hash string) {
	block, err := cli.dag.GetBlock(hash)
	if err != nil {
		fmt.Printf("Block not found: %s\n", hash)
		return
	}

	fmt.Printf("Block Details:\n")
	fmt.Printf("  Hash: %s\n", block.ComputeHash())
	fmt.Printf("  Height: %d\n", block.Header.Height)
	fmt.Printf("  Wave: %d\n", block.Header.Wave)
	fmt.Printf("  Timestamp: %s\n", block.Timestamp.Format(time.RFC3339))
	fmt.Printf("  Consensus Status: %s\n", cli.getConsensusStatus(block))
	fmt.Printf("  Confirmations: %d\n", cli.getConfirmations(block))
	fmt.Printf("  Transactions: %d\n", len(block.Body.Transactions))
	
	if len(block.Body.Transactions) > 0 {
		fmt.Printf("\nTransactions:\n")
		for i, tx := range block.Body.Transactions {
			fmt.Printf("  %d. Hash: %s\n", i+1, hex.EncodeToString(tx.ComputeHash()))
			fmt.Printf("     From: %s\n", string(tx.From))
			fmt.Printf("     To: %s\n", string(tx.To))
			fmt.Printf("     Amount: %d\n", tx.Value)
			fmt.Printf("     Nonce: %d\n", tx.Nonce)
		}
	}
}

func (cli *CLI) createNewAccount() {
	// Generate a new account with a random address
	addr := make([]byte, 20)
	for i := range addr {
		addr[i] = byte(time.Now().UnixNano() % 256)
	}

	account := &types.Account{
		Address: types.Address(hex.EncodeToString(addr)),
		Balance: types.Value(1000000), // Give initial balance
		Nonce:   0,
	}
	cli.accounts[hex.EncodeToString(addr)] = account
	fmt.Printf("Created new account: %s\n", hex.EncodeToString(addr))
	cli.logger.Printf("Created new account: %s with initial balance: %d", 
		hex.EncodeToString(addr), account.Balance)

	// Save state after creating account
	if err := cli.saveState(); err != nil {
		cli.logger.Printf("Warning: Failed to save state after creating account: %v", err)
	}
}

func (cli *CLI) listAccounts() {
	fmt.Printf("Accounts (%d):\n", len(cli.accounts))
	for addr, acc := range cli.accounts {
		fmt.Printf("  - Address: %s, Balance: %d, Nonce: %d\n",
			addr, acc.Balance, acc.Nonce)
	}
}

func (cli *CLI) checkBalance(address string) {
	if acc, ok := cli.accounts[address]; ok {
		fmt.Printf("Balance of %s: %d\n", address, acc.Balance)
	} else {
		fmt.Printf("Account not found: %s\n", address)
	}
}

func (cli *CLI) sendTransaction(from, to, amountStr string) {
	// Parse amount
	amount, err := strconv.ParseUint(amountStr, 10, 64)
	if err != nil {
		cli.logger.Printf("Error: Invalid amount: %s", amountStr)
		fmt.Printf("Invalid amount: %s\n", amountStr)
		return
	}

	// Get sender account
	sender, ok := cli.accounts[from]
	if !ok {
		cli.logger.Printf("Error: Sender account not found: %s", from)
		fmt.Printf("Sender account not found: %s\n", from)
		return
	}

	// Check balance
	if sender.Balance < types.Value(amount) {
		cli.logger.Printf("Error: Insufficient balance for %s: %d < %d", 
			from, sender.Balance, amount)
		fmt.Printf("Insufficient balance: %d < %d\n", sender.Balance, amount)
		return
	}

	// Create transaction
	tx := &types.Transaction{
		From:   types.Address(from),
		To:     types.Address(to),
		Value:  types.Value(amount),
		Nonce:  sender.Nonce,
		Data:   nil,
	}

	// Add transaction to mempool
	cli.mempool.AddTransaction(tx)
	cli.logger.Printf("Added transaction to mempool: %s", hex.EncodeToString(tx.ComputeHash()))

	// Update balances
	sender.Balance -= types.Value(amount)
	if recipient, ok := cli.accounts[to]; ok {
		recipient.Balance += types.Value(amount)
		cli.logger.Printf("Updated existing recipient account %s balance: %d", 
			to, recipient.Balance)
	} else {
		// Create recipient account if it doesn't exist
		recipient := &types.Account{
			Address: types.Address(to),
			Balance: types.Value(amount),
			Nonce:   0,
		}
		cli.accounts[to] = recipient
		cli.logger.Printf("Created new recipient account %s with balance: %d", 
			to, amount)
	}

	// Update sender nonce
	sender.Nonce++
	cli.logger.Printf("Updated sender %s nonce: %d", from, sender.Nonce)

	// Broadcast transaction
	if err := cli.msgHandler.BroadcastMessage(network.MessageTypeBlock, tx); err != nil {
		cli.logger.Printf("Error: Failed to broadcast transaction: %v", err)
		fmt.Printf("Failed to broadcast transaction: %v\n", err)
		return
	}

	txHash := hex.EncodeToString(tx.ComputeHash())
	fmt.Printf("Transaction sent: %s\n", txHash)
	cli.logger.Printf("Transaction sent: %s (From: %s, To: %s, Amount: %d)", 
		txHash, from, to, amount)

	// Save state after transaction
	if err := cli.saveState(); err != nil {
		cli.logger.Printf("Warning: Failed to save state after transaction: %v", err)
	}
}

func (cli *CLI) createBlock() {
	if !cli.isValidator {
		cli.logger.Printf("Error: Non-validator attempted to create block")
		fmt.Println("Only validators can create blocks")
		return
	}

	// Get pending transactions from mempool
	txs := cli.mempool.GetTransactions()
	if len(txs) == 0 {
		cli.logger.Printf("No pending transactions to create block")
		fmt.Println("No pending transactions")
		return
	}

	// Convert []*Transaction to []Transaction
	blockTxs := make([]types.Transaction, len(txs))
	for i, tx := range txs {
		blockTxs[i] = *tx
	}

	// Get latest block hash
	latestBlock, err := cli.dag.GetBlockByHeight(cli.dag.GetHeight())
	if err != nil {
		cli.logger.Printf("Error: Failed to get latest block: %v", err)
		fmt.Printf("Failed to get latest block: %v\n", err)
		return
	}

	// Create block
	block := &types.Block{
		Header: &types.BlockHeader{
			Version:    1,
			Round:      types.Round(cli.currentWave),
			Wave:       types.Wave(cli.currentWave),
			Height:     cli.dag.GetHeight() + 1,
			ParentHash: []byte(latestBlock.ComputeHash()),
			StateRoot:  nil, // TODO: Calculate state root
			Validator:  []byte(cli.nodeID),
			Timestamp:  time.Now(),
		},
		Body: &types.BlockBody{
			Transactions: blockTxs,
			Receipts:    make([]types.Receipt, 0),
			Events:      make([]types.Event, 0),
			References:  make([]types.Reference, 0),
		},
		Timestamp: time.Now(),
	}

	// Add block to DAG
	if err := cli.dag.AddBlock(block); err != nil {
		cli.logger.Printf("Error: Failed to add block to DAG: %v", err)
		fmt.Printf("Failed to add block to DAG: %v\n", err)
		return
	}

	// Clear mempool after successful block creation
	cli.mempool.Clear()
	cli.logger.Printf("Cleared mempool after block creation")

	// Broadcast block
	if err := cli.msgHandler.BroadcastMessage(network.MessageTypeBlock, block); err != nil {
		cli.logger.Printf("Error: Failed to broadcast block: %v", err)
		fmt.Printf("Failed to broadcast block: %v\n", err)
		return
	}

	blockHashStr := block.ComputeHash()
	fmt.Printf("Block created and broadcast: %s\n", blockHashStr)
	cli.logger.Printf("Block created and broadcast: %s (Height: %d, Wave: %d, Transactions: %d)", 
		blockHashStr, block.Header.Height, block.Header.Wave, len(txs))

	// Save state after creating block
	if err := cli.saveState(); err != nil {
		cli.logger.Printf("Warning: Failed to save state after creating block: %v", err)
	}
}

func (cli *CLI) handleVote(args []string) error {
	if len(args) != 5 {
		return fmt.Errorf("usage: vote <proposal_id> <block_hash> <wave> <round> <validator>")
	}

	proposalID := make(types.Hash, 32)
	if _, err := hex.Decode(proposalID[:], []byte(args[0])); err != nil {
		return fmt.Errorf("invalid proposal ID: %v", err)
	}

	blockHash := make(types.Hash, 32)
	if _, err := hex.Decode(blockHash[:], []byte(args[1])); err != nil {
		return fmt.Errorf("invalid block hash: %v", err)
	}

	wave, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid wave number: %v", err)
	}
	round, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid round number: %v", err)
	}
	validator := args[4]

	vote := &types.Vote{
		ProposalID: proposalID,
		BlockHash:  blockHash,
		Wave:       types.Wave(wave),
		Round:      types.Round(round),
		Validator:  types.Address(validator),
		Timestamp:  time.Now(),
	}

	return cli.consensusEngine.HandleVote(vote)
} 