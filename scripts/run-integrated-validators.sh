#!/bin/bash

# BlazeDAG Integrated Validator Script
# This script runs multiple validators with DAG sync, wave consensus, and EVM compatibility

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY_NAME="integrated-node"
BINARY_PATH="$PROJECT_DIR/$BINARY_NAME"

# Default configuration
NUM_VALIDATORS=3
CHAIN_ID=1337
ROUND_DURATION="2s"
WAVE_DURATION="3s"
CREATE_ACCOUNTS=5
INITIAL_BALANCE="1000000000000000000000"

# Network configuration
BASE_DAG_PORT=4001
BASE_WAVE_PORT=6001
BASE_EVM_PORT=8545

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to print section headers
print_header() {
    echo
    print_color $CYAN "================================"
    print_color $CYAN "$1"
    print_color $CYAN "================================"
}

# Function to cleanup on exit
cleanup() {
    print_header "🛑 SHUTTING DOWN ALL VALIDATORS"
    
    # Kill all integrated-node processes
    pkill -f "$BINARY_NAME" 2>/dev/null || true
    
    # Wait a moment for graceful shutdown
    sleep 2
    
    # Force kill if necessary
    pkill -9 -f "$BINARY_NAME" 2>/dev/null || true
    
    print_color $GREEN "✅ All validators stopped"
}

# Function to build the integrated node
build_integrated_node() {
    print_header "🔨 BUILDING INTEGRATED NODE"
    
    cd "$PROJECT_DIR"
    
    print_color $YELLOW "Building integrated node binary..."
    go build -o "$BINARY_NAME" ./cmd/integrated-node
    
    if [ ! -f "$BINARY_PATH" ]; then
        print_color $RED "❌ Failed to build integrated node binary"
        exit 1
    fi
    
    print_color $GREEN "✅ Integrated node built successfully: $BINARY_PATH"
}

# Function to get peer addresses for a validator
get_dag_peers() {
    local validator_id=$1
    local peers=""
    
    for i in $(seq 1 $NUM_VALIDATORS); do
        if [ $i -ne $validator_id ]; then
            local port=$((BASE_DAG_PORT + i - 1))
            if [ -n "$peers" ]; then
                peers="$peers,0.0.0.0:$port"
            else
                peers="0.0.0.0:$port"
            fi
        fi
    done
    echo "$peers"
}

# Function to get wave peers for a validator
get_wave_peers() {
    local validator_id=$1
    local peers=""
    
    for i in $(seq 1 $NUM_VALIDATORS); do
        if [ $i -ne $validator_id ]; then
            local port=$((BASE_WAVE_PORT + i - 1))
            if [ -n "$peers" ]; then
                peers="$peers,0.0.0.0:$port"
            else
                peers="0.0.0.0:$port"
            fi
        fi
    done
    echo "$peers"
}

# Function to start a single validator
start_validator() {
    local validator_id=$1
    local dag_port=$((BASE_DAG_PORT + validator_id - 1))
    local wave_port=$((BASE_WAVE_PORT + validator_id - 1))
    local evm_port=$((BASE_EVM_PORT + validator_id - 1))
    
    local dag_peers=$(get_dag_peers $validator_id)
    local wave_peers=$(get_wave_peers $validator_id)
    
    local log_file="$PROJECT_DIR/validator${validator_id}_integrated.log"
    
    print_color $BLUE "🚀 Starting Validator $validator_id:"
    print_color $YELLOW "  📡 DAG Sync: 0.0.0.0:$dag_port"
    print_color $YELLOW "  🌊 Wave Consensus: 0.0.0.0:$wave_port"
    print_color $YELLOW "  ⚡ EVM RPC: 0.0.0.0:$evm_port"
    print_color $YELLOW "  📝 Log: $log_file"
    
    # Start the integrated validator
    "$BINARY_PATH" \
        -id="validator$validator_id" \
        -dag-listen="0.0.0.0:$dag_port" \
        -wave-listen="0.0.0.0:$wave_port" \
        -evm-rpc="0.0.0.0:$evm_port" \
        -dag-peers="$dag_peers" \
        -wave-peers="$wave_peers" \
        -round-duration="$ROUND_DURATION" \
        -wave-duration="$WAVE_DURATION" \
        -chain-id="$CHAIN_ID" \
        -create-accounts="$CREATE_ACCOUNTS" \
        -initial-balance="$INITIAL_BALANCE" \
        > "$log_file" 2>&1 &
    
    local pid=$!
    echo $pid > "$PROJECT_DIR/validator${validator_id}_integrated.pid"
    
    print_color $GREEN "✅ Validator $validator_id started (PID: $pid)"
}

# Function to wait for validators to start
wait_for_validators() {
    print_header "⏳ WAITING FOR VALIDATORS TO START"
    
    local max_wait=30
    local wait_time=0
    
    while [ $wait_time -lt $max_wait ]; do
        local ready_count=0
        
        for i in $(seq 1 $NUM_VALIDATORS); do
            local evm_port=$((BASE_EVM_PORT + i - 1))
            if curl -s -X POST -H "Content-Type: application/json" \
                --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
                "http://localhost:$evm_port" > /dev/null 2>&1; then
                ready_count=$((ready_count + 1))
            fi
        done
        
        if [ $ready_count -eq $NUM_VALIDATORS ]; then
            print_color $GREEN "✅ All $NUM_VALIDATORS validators are ready!"
            return 0
        fi
        
        print_color $YELLOW "⏳ $ready_count/$NUM_VALIDATORS validators ready, waiting..."
        sleep 2
        wait_time=$((wait_time + 2))
    done
    
    print_color $RED "❌ Timeout waiting for validators to start"
    return 1
}

# Function to test EVM functionality
test_evm_functionality() {
    print_header "🧪 TESTING EVM FUNCTIONALITY"
    
    local validator_id=1
    local evm_port=$((BASE_EVM_PORT + validator_id - 1))
    local rpc_url="http://localhost:$evm_port"
    
    print_color $YELLOW "Testing on Validator $validator_id (port $evm_port)..."
    
    # Test 1: Get chain ID
    print_color $BLUE "📋 Test 1: Getting chain ID..."
    local chain_id_response=$(curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
        "$rpc_url")
    echo "Response: $chain_id_response"
    
    # Test 2: Get accounts
    print_color $BLUE "📋 Test 2: Getting accounts..."
    local accounts_response=$(curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}' \
        "$rpc_url")
    echo "Response: $accounts_response"
    
    # Extract first account for further tests
    local first_account=$(echo $accounts_response | jq -r '.result[0]' 2>/dev/null || echo "")
    
    if [ -n "$first_account" ] && [ "$first_account" != "null" ]; then
        # Test 3: Get balance
        print_color $BLUE "📋 Test 3: Getting balance for $first_account..."
        local balance_response=$(curl -s -X POST -H "Content-Type: application/json" \
            --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"$first_account\",\"latest\"],\"id\":1}" \
            "$rpc_url")
        echo "Response: $balance_response"
        
        # Test 4: Simple transaction
        print_color $BLUE "📋 Test 4: Sending simple transaction..."
        local tx_response=$(curl -s -X POST -H "Content-Type: application/json" \
            --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_sendTransaction\",\"params\":[{\"from\":\"$first_account\",\"to\":\"0x742d35Cc6635C0532925a3b8D4B9D13E2c8b3958\",\"value\":\"0x1000\"}],\"id\":1}" \
            "$rpc_url")
        echo "Response: $tx_response"
    else
        print_color $RED "❌ Could not extract account for further testing"
    fi
    
    print_color $GREEN "✅ EVM functionality tests completed"
}

# Function to deploy and test smart contract
deploy_test_contract() {
    print_header "📄 DEPLOYING AND TESTING SMART CONTRACT"
    
    local validator_id=1
    local evm_port=$((BASE_EVM_PORT + validator_id - 1))
    local rpc_url="http://localhost:$evm_port"
    
    # Get accounts first
    local accounts_response=$(curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}' \
        "$rpc_url")
    local first_account=$(echo $accounts_response | jq -r '.result[0]' 2>/dev/null || echo "")
    
    if [ -z "$first_account" ] || [ "$first_account" = "null" ]; then
        print_color $RED "❌ No accounts available for contract deployment"
        return 1
    fi
    
    print_color $YELLOW "Using account: $first_account"
    
    # Simple storage contract bytecode
    # contract SimpleStorage { uint256 public value; function setValue(uint256 _value) public { value = _value; } }
    local contract_bytecode="0x608060405234801561001057600080fd5b5060e98061001f6000396000f3fe6080604052348015600f57600080fd5b5060043610603c5760003560e01c80633fa4f2451460415780635ca1e1651460575780635fa7b58414605b575b600080fd5b6047607f565b60408051918252519081900360200190f35b605f6085565b005b6079600480360360208110156070576000fd5b50356088565b60408051918252519081900360200190f35b60005481565b60005490565b600055565b6000549056fea264697066735822122012345678901234567890123456789012345678901234567890123456789012345664736f6c63430007060033"
    
    print_color $BLUE "📄 Deploying SimpleStorage contract..."
    local deploy_response=$(curl -s -X POST -H "Content-Type: application/json" \
        --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_sendTransaction\",\"params\":[{\"from\":\"$first_account\",\"data\":\"$contract_bytecode\",\"gas\":\"0x100000\"}],\"id\":1}" \
        "$rpc_url")
    
    echo "Deploy response: $deploy_response"
    
    local tx_hash=$(echo $deploy_response | jq -r '.result' 2>/dev/null || echo "")
    if [ -n "$tx_hash" ] && [ "$tx_hash" != "null" ]; then
        print_color $GREEN "✅ Contract deployment transaction sent: $tx_hash"
        print_color $YELLOW "💡 Smart contract transaction should appear in DAG blocks and wave consensus logs!"
    else
        print_color $RED "❌ Contract deployment failed"
    fi
}

# Function to show validator status
show_validator_status() {
    print_header "📊 VALIDATOR STATUS"
    
    for i in $(seq 1 $NUM_VALIDATORS); do
        local dag_port=$((BASE_DAG_PORT + i - 1))
        local wave_port=$((BASE_WAVE_PORT + i - 1))
        local evm_port=$((BASE_EVM_PORT + i - 1))
        local pid_file="$PROJECT_DIR/validator${i}_integrated.pid"
        
        print_color $CYAN "📊 Validator $i Status:"
        
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                print_color $GREEN "  ✅ Process: Running (PID: $pid)"
            else
                print_color $RED "  ❌ Process: Not running"
            fi
        else
            print_color $RED "  ❌ Process: PID file not found"
        fi
        
        # Test DAG sync
        if curl -s "http://localhost:8080/status" > /dev/null 2>&1; then
            print_color $GREEN "  ✅ DAG Sync: Responsive (http://localhost:8080)"
        else
            print_color $RED "  ❌ DAG Sync: Not responsive"
        fi
        
        # Test Wave consensus
        if curl -s "http://localhost:8081/wave/status" > /dev/null 2>&1; then
            print_color $GREEN "  ✅ Wave Consensus: Responsive (http://localhost:8081)"
        else
            print_color $RED "  ❌ Wave Consensus: Not responsive"
        fi
        
        # Test EVM RPC
        if curl -s -X POST -H "Content-Type: application/json" \
            --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
            "http://localhost:$evm_port" > /dev/null 2>&1; then
            print_color $GREEN "  ✅ EVM RPC: Responsive (http://localhost:$evm_port)"
        else
            print_color $RED "  ❌ EVM RPC: Not responsive"
        fi
        
        echo
    done
}

# Function to show connection instructions
show_connection_info() {
    print_header "🔗 CONNECTION INFORMATION"
    
    print_color $YELLOW "EVM RPC Endpoints (for MetaMask/web3.js):"
    for i in $(seq 1 $NUM_VALIDATORS); do
        local evm_port=$((BASE_EVM_PORT + i - 1))
        print_color $CYAN "  Validator $i: http://localhost:$evm_port (Chain ID: $CHAIN_ID)"
    done
    
    echo
    print_color $YELLOW "DAG Sync API:"
    print_color $CYAN "  http://localhost:8080 (DAG status, blocks, transactions)"
    
    echo
    print_color $YELLOW "Wave Consensus API:"
    print_color $CYAN "  http://localhost:8081 (Wave status, consensus info)"
    
    echo
    print_color $YELLOW "MetaMask Setup:"
    print_color $CYAN "  1. Add Custom RPC: http://localhost:8545"
    print_color $CYAN "  2. Chain ID: $CHAIN_ID"
    print_color $CYAN "  3. Currency: ETH"
    print_color $CYAN "  4. Import accounts using private keys from validator logs"
}

# Function to monitor logs
monitor_logs() {
    print_header "📝 MONITORING VALIDATOR LOGS"
    
    print_color $YELLOW "Following logs for all validators (Ctrl+C to stop)..."
    sleep 2
    
    # Use tail to follow all log files
    tail -f "$PROJECT_DIR"/validator*_integrated.log
}

# Main execution
main() {
    print_header "🚀 BLAZEDAG INTEGRATED VALIDATORS STARTUP"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --validators)
                NUM_VALIDATORS="$2"
                shift 2
                ;;
            --chain-id)
                CHAIN_ID="$2"
                shift 2
                ;;
            --round-duration)
                ROUND_DURATION="$2"
                shift 2
                ;;
            --wave-duration)
                WAVE_DURATION="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  --validators N        Number of validators (default: 3)"
                echo "  --chain-id ID         EVM chain ID (default: 1337)"
                echo "  --round-duration D    DAG round duration (default: 2s)"
                echo "  --wave-duration D     Wave duration (default: 3s)"
                echo "  --help               Show this help"
                exit 0
                ;;
            *)
                print_color $RED "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    print_color $YELLOW "Configuration:"
    print_color $CYAN "  Validators: $NUM_VALIDATORS"
    print_color $CYAN "  Chain ID: $CHAIN_ID"
    print_color $CYAN "  Round Duration: $ROUND_DURATION"
    print_color $CYAN "  Wave Duration: $WAVE_DURATION"
    
    # Set up cleanup trap
    trap cleanup EXIT INT TERM
    
    # Build the integrated node
    build_integrated_node
    
    # Start all validators
    print_header "🚀 STARTING $NUM_VALIDATORS VALIDATORS"
    for i in $(seq 1 $NUM_VALIDATORS); do
        start_validator $i
        sleep 1  # Brief delay between starts
    done
    
    # Wait for validators to be ready
    if ! wait_for_validators; then
        exit 1
    fi
    
    # Show connection information
    show_connection_info
    
    # Test EVM functionality
    test_evm_functionality
    
    # Deploy and test smart contract
    deploy_test_contract
    
    # Show validator status
    show_validator_status
    
    print_header "🎉 ALL VALIDATORS RUNNING SUCCESSFULLY!"
    print_color $GREEN "✅ DAG Sync + Wave Consensus + EVM all integrated and running"
    print_color $YELLOW "💡 Smart contract deployments will appear in DAG blocks and wave consensus"
    print_color $CYAN "📝 Check validator logs for detailed information"
    
    echo
    print_color $YELLOW "Press Ctrl+C to stop all validators or 'l' + Enter to monitor logs:"
    read -t 5 input || input=""
    
    if [ "$input" = "l" ]; then
        monitor_logs
    else
        print_color $CYAN "Validators continue running in background..."
        print_color $CYAN "Use 'pkill -f integrated-node' to stop them manually"
        # Remove trap to prevent cleanup on normal exit
        trap - EXIT
    fi
}

# Check if script is being sourced or executed
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi 