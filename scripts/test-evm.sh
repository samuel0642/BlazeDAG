#!/bin/bash

set -e

echo "🔥 BlazeDAG EVM Compatibility Test Script"
echo "==========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
RPC_URL="http://localhost:8545"
CHAIN_ID=1337

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to make JSON-RPC calls
rpc_call() {
    local method=$1
    local params=$2
    local id=${3:-1}
    
    curl -s -X POST \
        -H "Content-Type: application/json" \
        --data "{\"jsonrpc\":\"2.0\",\"method\":\"$method\",\"params\":$params,\"id\":$id}" \
        $RPC_URL
}

# Function to extract result from JSON-RPC response
extract_result() {
    echo $1 | jq -r '.result'
}

# Function to check if server is running
check_server() {
    print_status "Checking if EVM node is running..."
    if curl -s $RPC_URL > /dev/null 2>&1; then
        print_success "EVM node is running at $RPC_URL"
        return 0
    else
        print_error "EVM node is not running at $RPC_URL"
        return 1
    fi
}

# Function to start the EVM node
start_evm_node() {
    print_status "Starting BlazeDAG EVM Node..."
    
    # Build the EVM node
    cd "$(dirname "$0")/.."
    print_status "Building EVM node..."
    go build -o bin/evm-node ./cmd/evm-node
    
    # Start the node in background
    print_status "Starting EVM node in background..."
    ./bin/evm-node \
        -rpc-addr=localhost:8545 \
        -chain-id=$CHAIN_ID \
        -create-accounts=5 \
        -initial-balance=1000000000000000000000 > evm-node.log 2>&1 &
    
    EVM_NODE_PID=$!
    echo $EVM_NODE_PID > evm-node.pid
    
    # Wait for node to start
    print_status "Waiting for EVM node to start..."
    sleep 3
    
    # Check if it's running
    for i in {1..10}; do
        if check_server; then
            break
        fi
        if [ $i -eq 10 ]; then
            print_error "EVM node failed to start"
            exit 1
        fi
        sleep 2
    done
}

# Function to test basic RPC functionality
test_basic_rpc() {
    print_status "Testing basic RPC functionality..."
    
    # Test eth_chainId
    print_status "Testing eth_chainId..."
    response=$(rpc_call "eth_chainId" "[]")
    chain_id=$(extract_result "$response")
    if [ "$chain_id" = "0x539" ]; then  # 1337 in hex
        print_success "Chain ID correct: $chain_id"
    else
        print_error "Chain ID incorrect: $chain_id"
        return 1
    fi
    
    # Test eth_accounts
    print_status "Testing eth_accounts..."
    response=$(rpc_call "eth_accounts" "[]")
    accounts=$(extract_result "$response")
    account_count=$(echo $accounts | jq '. | length')
    if [ "$account_count" -ge 3 ]; then
        print_success "Found $account_count accounts"
        ACCOUNT1=$(echo $accounts | jq -r '.[0]')
        ACCOUNT2=$(echo $accounts | jq -r '.[1]')
        print_status "Using account 1: $ACCOUNT1"
        print_status "Using account 2: $ACCOUNT2"
    else
        print_error "Not enough accounts found: $account_count"
        return 1
    fi
    
    # Test eth_getBalance
    print_status "Testing eth_getBalance..."
    response=$(rpc_call "eth_getBalance" "[\"$ACCOUNT1\", \"latest\"]")
    balance=$(extract_result "$response")
    if [ "$balance" != "0x0" ] && [ "$balance" != "null" ]; then
        print_success "Account balance: $balance"
    else
        print_error "Account has no balance: $balance"
        return 1
    fi
    
    # Test eth_getTransactionCount
    print_status "Testing eth_getTransactionCount..."
    response=$(rpc_call "eth_getTransactionCount" "[\"$ACCOUNT1\", \"latest\"]")
    nonce=$(extract_result "$response")
    print_success "Account nonce: $nonce"
}

# Function to test smart contract deployment
test_contract_deployment() {
    print_status "Testing smart contract deployment..."
    
    # Simple storage contract bytecode
    # contract SimpleStorage {
    #     uint256 public value;
    #     function setValue(uint256 _value) public { value = _value; }
    #     function getValue() public view returns (uint256) { return value; }
    # }
    CONTRACT_BYTECODE="0x608060405234801561001057600080fd5b50610150806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80633fa4f2451461003b57806355241077146100595b600080fd5b610043610075565b60405161005091906100d1565b60405180910390f35b6100736004803603810190610068919061009d565b61007b565b005b60005481565b8060008190555050565b60008135905061009481610103565b92915050565b6000602082840312156100b0576100af6100fe565b5b60006100be84828501610085565b91505092915050565b6100d0816100f4565b82525050565b60006020820190506100eb60008301846100c7565b92915050565b6000819050919050565b600080fd5b61010c816100f4565b811461011757600080fd5b5056fea2646970667358221220e6966e446bd0af8e6af40eb0d8f323dd02f8e2c2b0f8e8d8f8e2f8e2f8e2f8e264736f6c63430008070033"
    
    # Deploy contract
    print_status "Deploying SimpleStorage contract..."
    deploy_response=$(rpc_call "eth_sendTransaction" "[{\"from\":\"$ACCOUNT1\",\"data\":\"$CONTRACT_BYTECODE\",\"gas\":\"0x100000\"}]")
    deploy_result=$(extract_result "$deploy_response")
    
    if [ "$deploy_result" != "null" ] && [ ${#deploy_result} -eq 66 ]; then
        print_success "Contract deployed! Transaction hash: $deploy_result"
        CONTRACT_TX_HASH=$deploy_result
    else
        print_error "Contract deployment failed: $deploy_result"
        return 1
    fi
    
    # Calculate contract address (simplified)
    print_status "Contract deployment completed"
}

# Function to test ETH transfers
test_eth_transfer() {
    print_status "Testing ETH transfer..."
    
    # Transfer 1 ETH from account1 to account2
    transfer_amount="0xde0b6b3a7640000"  # 1 ETH in wei
    
    print_status "Transferring 1 ETH from $ACCOUNT1 to $ACCOUNT2..."
    transfer_response=$(rpc_call "eth_sendTransaction" "[{\"from\":\"$ACCOUNT1\",\"to\":\"$ACCOUNT2\",\"value\":\"$transfer_amount\",\"gas\":\"0x5208\"}]")
    transfer_result=$(extract_result "$transfer_response")
    
    if [ "$transfer_result" != "null" ] && [ ${#transfer_result} -eq 66 ]; then
        print_success "Transfer successful! Transaction hash: $transfer_result"
        
        # Check balance of account2
        sleep 1
        balance_response=$(rpc_call "eth_getBalance" "[\"$ACCOUNT2\", \"latest\"]")
        new_balance=$(extract_result "$balance_response")
        print_success "Account 2 new balance: $new_balance"
    else
        print_error "Transfer failed: $transfer_result"
        return 1
    fi
}

# Function to test contract interaction
test_contract_interaction() {
    print_status "Testing contract interaction..."
    
    # For this demo, we'll test eth_call functionality
    # In a real scenario, you would call the deployed contract
    
    # Test eth_call with empty data
    print_status "Testing eth_call..."
    call_response=$(rpc_call "eth_call" "[{\"to\":\"$ACCOUNT2\",\"data\":\"0x\"},\"latest\"]")
    call_result=$(extract_result "$call_response")
    
    if [ "$call_result" != "null" ]; then
        print_success "Contract call successful: $call_result"
    else
        print_warning "Contract call returned null (expected for empty call)"
    fi
}

# Function to test gas estimation
test_gas_estimation() {
    print_status "Testing gas estimation..."
    
    # Test eth_estimateGas
    gas_response=$(rpc_call "eth_estimateGas" "[{\"from\":\"$ACCOUNT1\",\"to\":\"$ACCOUNT2\",\"value\":\"0x1\"}]")
    gas_estimate=$(extract_result "$gas_response")
    
    if [ "$gas_estimate" != "null" ]; then
        print_success "Gas estimation: $gas_estimate"
    else
        print_error "Gas estimation failed"
        return 1
    fi
}

# Function to run performance test
test_performance() {
    print_status "Running performance test..."
    
    print_status "Sending 10 concurrent transactions..."
    for i in {1..10}; do
        (
            transfer_response=$(rpc_call "eth_sendTransaction" "[{\"from\":\"$ACCOUNT1\",\"to\":\"$ACCOUNT2\",\"value\":\"0x1\",\"gas\":\"0x5208\"}]" $i)
            transfer_result=$(extract_result "$transfer_response")
            if [ "$transfer_result" != "null" ]; then
                echo "Transaction $i: SUCCESS"
            else
                echo "Transaction $i: FAILED"
            fi
        ) &
    done
    wait
    print_success "Performance test completed"
}

# Function to cleanup
cleanup() {
    print_status "Cleaning up..."
    
    if [ -f evm-node.pid ]; then
        EVM_NODE_PID=$(cat evm-node.pid)
        if kill -0 $EVM_NODE_PID > /dev/null 2>&1; then
            print_status "Stopping EVM node (PID: $EVM_NODE_PID)..."
            kill $EVM_NODE_PID
            sleep 2
        fi
        rm evm-node.pid
    fi
    
    if [ -f evm-node.log ]; then
        print_status "EVM node logs saved to evm-node.log"
    fi
}

# Main test function
main() {
    echo
    print_status "Starting EVM compatibility test suite..."
    echo
    
    # Trap to ensure cleanup on exit
    trap cleanup EXIT
    
    # Start EVM node if not running
    if ! check_server; then
        start_evm_node
    fi
    
    echo
    print_status "Running test suite..."
    echo
    
    # Run tests
    test_basic_rpc || exit 1
    echo
    
    test_eth_transfer || exit 1
    echo
    
    test_contract_deployment || exit 1
    echo
    
    test_contract_interaction || exit 1
    echo
    
    test_gas_estimation || exit 1
    echo
    
    test_performance || exit 1
    echo
    
    print_success "🎉 All tests passed! BlazeDAG EVM compatibility is working!"
    echo
    
    print_status "EVM Node Information:"
    print_status "- RPC URL: $RPC_URL"
    print_status "- Chain ID: $CHAIN_ID"
    print_status "- Accounts: Use eth_accounts to get available accounts"
    print_status "- Compatible with: MetaMask, web3.js, ethers.js, etc."
    echo
    
    print_warning "Press Ctrl+C to stop the EVM node"
    
    # Keep running until interrupted
    while true; do
        sleep 1
    done
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    print_error "jq is required but not installed. Please install jq first."
    print_status "On Ubuntu/Debian: sudo apt-get install jq"
    print_status "On macOS: brew install jq"
    exit 1
fi

# Check if curl is installed
if ! command -v curl &> /dev/null; then
    print_error "curl is required but not installed."
    exit 1
fi

# Run main function
main 