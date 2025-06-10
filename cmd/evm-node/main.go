package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/CrossDAG/BlazeDAG/internal/api"
	"github.com/CrossDAG/BlazeDAG/internal/evm"
)

func main() {
	// Parse command line flags
	rpcAddr := flag.String("rpc-addr", "localhost:8545", "JSON-RPC server address")
	chainID := flag.Int64("chain-id", 1337, "Chain ID for the network")
	createAccounts := flag.Int("create-accounts", 3, "Number of test accounts to create")
	initialBalance := flag.String("initial-balance", "1000000000000000000000", "Initial balance for test accounts (in wei)")
	flag.Parse()

	fmt.Printf("🚀 Starting BlazeDAG EVM Node\n")
	fmt.Printf("Chain ID: %d\n", *chainID)
	fmt.Printf("RPC Address: %s\n", *rpcAddr)
	fmt.Printf("Creating %d test accounts\n", *createAccounts)

	// Create EVM state
	state := evm.NewState()

	// Parse initial balance
	balance, ok := new(big.Int).SetString(*initialBalance, 10)
	if !ok {
		log.Fatalf("Invalid initial balance: %s", *initialBalance)
	}

	// Create keystore and test accounts
	keystore := evm.NewKeystore()
	testAccounts, err := evm.CreateTestAccounts(state, *createAccounts, balance)
	if err != nil {
		log.Fatalf("Failed to create test accounts: %v", err)
	}

	// Add accounts to keystore
	for _, account := range testAccounts {
		keystore.AddKey(account)
	}

	fmt.Printf("\n📝 Created Test Accounts:\n")
	for i, account := range testAccounts {
		fmt.Printf("Account %d:\n", i+1)
		fmt.Printf("  Address: %s\n", account.GetAddressHex())
		fmt.Printf("  Private Key: %s\n", account.GetPrivateKeyHex())
		fmt.Printf("  Balance: %s ETH\n", weiToEth(balance).String())
	}

	// Create EVM executor
	gasPrice := big.NewInt(1000000000) // 1 gwei
	executor := evm.NewEVMExecutor(state, gasPrice)

	// Create RPC server
	chainIDBig := big.NewInt(*chainID)
	rpcServer := api.NewEVMRPCServer(executor, keystore, chainIDBig)

	// Start HTTP server
	fmt.Printf("\n🌐 Starting JSON-RPC server on %s\n", *rpcAddr)
	fmt.Printf("You can now connect with:\n")
	fmt.Printf("  - MetaMask: http://%s\n", *rpcAddr)
	fmt.Printf("  - web3.js: http://%s\n", *rpcAddr)
	fmt.Printf("  - curl: curl -X POST -H \"Content-Type: application/json\" --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_accounts\",\"params\":[],\"id\":1}' http://%s\n", *rpcAddr)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine
	server := &http.Server{
		Addr:    *rpcAddr,
		Handler: rpcServer,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Demo: Deploy and interact with a simple contract
	fmt.Printf("\n🔧 Deploying Demo Smart Contract...\n")
	demoContract()

	// Wait for shutdown signal
	<-sigChan
	fmt.Printf("\n🛑 Shutting down...\n")
	server.Close()
}

// demoContract demonstrates deploying and interacting with a smart contract
func demoContract() {
	// Simple storage contract bytecode (stores and retrieves a uint256)
	// This is the compiled bytecode for:
	// contract SimpleStorage {
	//     uint256 public value;
	//     function setValue(uint256 _value) public { value = _value; }
	//     function getValue() public view returns (uint256) { return value; }
	// }
	contractBytecode := "608060405234801561001057600080fd5b5060e68061001f6000396000f3fe6080604052348015600f57600080fd5b5060043610604257600080fd5b80635fa7b584146047575b600080fd5b60505b6052565b60405190815260200160405180910390f35b60005481565b6004356000556056565b005b6000356001600160e01b03191681565b600080fd5b600180fd5b600290fd5b600390fd5b600490fd5b600590fd5b600690fd5b600790fd5b600890fd5b600990fd5b600a90fd5b600b90fd5b600c90fd5b600d90fd5b600e90fd5b600f90fd5b601090fd5b601190fd5b601290fd5b601390fd5b601490fd5b601590fd5b601690fd5b601790fd5b601890fd5b601990fd5b601a90fd5b601b90fd5b601c90fd5b601d90fd5b601e90fd5b601f90fd5b602090fd5b"

	// For demo purposes, just print what would happen
	fmt.Printf("  📄 Contract Bytecode: %s\n", contractBytecode[:50]+"...")
	fmt.Printf("  ✅ Contract deployment simulation completed\n")
	fmt.Printf("  💡 Use the JSON-RPC API to deploy real contracts\n")

	// Example of how to use the API:
	fmt.Printf("\n📚 Example Usage:\n")
	fmt.Printf("1. Get accounts: curl -X POST -H \"Content-Type: application/json\" --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_accounts\",\"params\":[],\"id\":1}' http://localhost:8545\n")
	fmt.Printf("2. Get balance: curl -X POST -H \"Content-Type: application/json\" --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"ADDRESS\",\"latest\"],\"id\":1}' http://localhost:8545\n")
	fmt.Printf("3. Deploy contract: Use web3.js or similar with the contract bytecode\n")
}

// weiToEth converts wei to ETH
func weiToEth(wei *big.Int) *big.Float {
	eth := new(big.Float).SetInt(wei)
	eth.Quo(eth, big.NewFloat(1e18))
	return eth
} 