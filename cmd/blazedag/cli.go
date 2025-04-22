package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/consensus"
	"github.com/CrossDAG/BlazeDAG/internal/core"
	"github.com/CrossDAG/BlazeDAG/internal/network"
	"github.com/CrossDAG/BlazeDAG/internal/state"
	"github.com/CrossDAG/BlazeDAG/internal/types"
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
}

func NewCLI() *CLI {
	return &CLI{
		accounts:    make(map[string]*types.Account),
		currentWave: 0,
	}
}

func (cli *CLI) Start(port int, isValidator bool) error {
	cli.isValidator = isValidator

	// Create P2P node
	nodeID := network.PeerID(fmt.Sprintf("node-%d", port))
	node := network.NewP2PNode(nodeID)
	if err := node.Start(); err != nil {
		return fmt.Errorf("failed to start P2P node: %v", err)
	}
	cli.node = node

	// Create message handler
	msgHandler := network.NewMessageHandler(node)
	if err := msgHandler.Start(); err != nil {
		return fmt.Errorf("failed to start message handler: %v", err)
	}
	cli.msgHandler = msgHandler

	// Create DAG
	dag := core.NewDAG()
	cli.dag = dag

	// Create state manager
	stateManager := state.NewStateManager()
	cli.stateManager = stateManager

	// Create EVM executor
	evm := core.NewEVMExecutor(stateManager)
	cli.evm = evm

	// Create consensus engine
	consensusEngine := consensus.NewConsensusEngine(dag, stateManager, evm, isValidator)
	if err := consensusEngine.Start(); err != nil {
		return fmt.Errorf("failed to start consensus engine: %v", err)
	}
	cli.consensusEngine = consensusEngine

	// Create wave controller
	waveController := consensus.NewWaveController(consensusEngine)
	if err := waveController.Start(); err != nil {
		return fmt.Errorf("failed to start wave controller: %v", err)
	}
	cli.waveController = waveController

	// Register message handlers
	cli.registerMessageHandlers()

	return nil
}

func (cli *CLI) registerMessageHandlers() {
	cli.msgHandler.RegisterHandler(network.MessageTypeBlock, func(msg *network.Message) error {
		var block types.Block
		if err := msg.UnmarshalPayload(&block); err != nil {
			return fmt.Errorf("failed to unmarshal block: %v", err)
		}
		return cli.consensusEngine.HandleBlock(&block)
	})

	cli.msgHandler.RegisterHandler(network.MessageTypeVote, func(msg *network.Message) error {
		var vote consensus.Vote
		if err := msg.UnmarshalPayload(&vote); err != nil {
			return fmt.Errorf("failed to unmarshal vote: %v", err)
		}
		return cli.consensusEngine.HandleVote(&vote)
	})

	cli.msgHandler.RegisterHandler(network.MessageTypeCertificate, func(msg *network.Message) error {
		var cert types.Certificate
		if err := msg.UnmarshalPayload(&cert); err != nil {
			return fmt.Errorf("failed to unmarshal certificate: %v", err)
		}
		return cli.consensusEngine.HandleCertificate(&cert)
	})
}

func (cli *CLI) Stop() {
	if cli.waveController != nil {
		cli.waveController.Stop()
	}
	if cli.consensusEngine != nil {
		cli.consensusEngine.Stop()
	}
	if cli.msgHandler != nil {
		cli.msgHandler.Stop()
	}
	if cli.node != nil {
		cli.node.Stop()
	}
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
	blocks := make([]*types.Block, 0)
	for _, block := range cli.dag.GetBlocks() {
		blocks = append(blocks, block)
		if len(blocks) >= 10 {
			break
		}
	}

	fmt.Printf("Recent Blocks (%d):\n", len(blocks))
	for _, block := range blocks {
		fmt.Printf("  - Hash: %s, Height: %d, Wave: %d\n",
			block.Hash(),
			block.Header.Height,
			block.Header.Wave)
	}
}

func (cli *CLI) createNewAccount() {
	// Generate a new account with a random address
	addr := make([]byte, 20)
	for i := range addr {
		addr[i] = byte(time.Now().UnixNano() % 256)
	}

	account := &types.Account{
		Address: addr,
		Balance: 0,
		Nonce:   0,
	}
	cli.accounts[hex.EncodeToString(addr)] = account
	fmt.Printf("Created new account: %s\n", hex.EncodeToString(addr))
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
		fmt.Printf("Invalid amount: %s\n", amountStr)
		return
	}

	// Get sender account
	sender, ok := cli.accounts[from]
	if !ok {
		fmt.Printf("Sender account not found: %s\n", from)
		return
	}

	// Check balance
	if sender.Balance < amount {
		fmt.Printf("Insufficient balance: %d < %d\n", sender.Balance, amount)
		return
	}

	// Create transaction
	tx := &types.Transaction{
		From:     []byte(from),
		To:       []byte(to),
		Value:    amount,
		Nonce:    sender.Nonce,
		GasPrice: 1,
		GasLimit: 21000,
		Data:     nil,
	}

	// Broadcast transaction
	if err := cli.msgHandler.BroadcastMessage(network.MessageTypeBlock, tx); err != nil {
		fmt.Printf("Failed to broadcast transaction: %v\n", err)
		return
	}

	// Update sender nonce
	sender.Nonce++
	fmt.Printf("Transaction sent: %s\n", hex.EncodeToString(tx.Hash()))
}

func (cli *CLI) createBlock() {
	if !cli.isValidator {
		fmt.Println("Only validators can create blocks")
		return
	}

	// Get pending transactions from mempool
	txs := make([]types.Transaction, 0)
	// TODO: Implement mempool and get pending transactions

	if len(txs) == 0 {
		fmt.Println("No pending transactions")
		return
	}

	// Create block
	block := &types.Block{
		Header: types.BlockHeader{
			Version:    1,
			Round:      cli.currentWave,
			Wave:       cli.currentWave,
			Height:     0, // Will be set by DAG
			ParentHash: nil, // Will be set by DAG
		},
		Body: types.BlockBody{
			Transactions: txs,
		},
		Timestamp: time.Now(),
	}

	// Sign block
	blockHash := sha256.Sum256([]byte(block.Hash()))
	block.Signature = blockHash[:]

	// Broadcast block
	if err := cli.msgHandler.BroadcastMessage(network.MessageTypeBlock, block); err != nil {
		fmt.Printf("Failed to broadcast block: %v\n", err)
		return
	}

	fmt.Printf("Block created and broadcast: %s\n", block.Hash())
}

func main() {
	// Parse command line flags
	port := flag.Int("port", 3000, "Port to listen on")
	isValidator := flag.Bool("validator", false, "Whether this node is a validator")
	flag.Parse()

	// Create and start CLI
	cli := NewCLI()
	if err := cli.Start(*port, *isValidator); err != nil {
		log.Fatalf("Failed to start CLI: %v", err)
	}
	defer cli.Stop()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cli.Stop()
		os.Exit(0)
	}()

	// Run CLI
	cli.Run()
} 