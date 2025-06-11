#!/bin/bash

# Smart Contract Testing Script for BlazeDAG Integrated Validators
# This script deploys and tests smart contracts on the running BlazeDAG chain

set -e

# Configuration
VALIDATOR_PORT=8545  # EVM RPC port for validator 1
RPC_URL="http://localhost:$VALIDATOR_PORT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# Function to check if jq is available
check_dependencies() {
    if ! command -v jq &> /dev/null; then
        print_color $YELLOW "⚠️  jq not found, installing..."
        sudo apt-get update && sudo apt-get install -y jq
    fi
    
    if ! command -v curl &> /dev/null; then
        print_color $RED "❌ curl is required but not found"
        exit 1
    fi
}

# Function to test connection
test_connection() {
    print_header "🔌 TESTING CONNECTION TO BLAZEDAG"
    
    print_color $YELLOW "Testing connection to $RPC_URL..."
    
    local response=$(curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
        "$RPC_URL" || echo "")
    
    if [ -z "$response" ]; then
        print_color $RED "❌ Cannot connect to BlazeDAG EVM RPC at $RPC_URL"
        print_color $YELLOW "💡 Make sure the integrated validators are running:"
        print_color $CYAN "   ./scripts/run-integrated-validators.sh"
        exit 1
    fi
    
    local chain_id=$(echo $response | jq -r '.result' 2>/dev/null || echo "error")
    print_color $GREEN "✅ Connected to BlazeDAG EVM RPC"
    print_color $CYAN "   Chain ID: $chain_id"
    
    echo "$response"
}

# Function to get accounts
get_accounts() {
    print_header "👥 GETTING ACCOUNTS"
    
    local response=$(curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}' \
        "$RPC_URL")
    
    echo "Accounts response: $response"
    
    local accounts=$(echo $response | jq -r '.result[]' 2>/dev/null || echo "")
    if [ -z "$accounts" ]; then
        print_color $RED "❌ No accounts found"
        exit 1
    fi
    
    print_color $GREEN "✅ Available accounts:"
    echo $response | jq -r '.result[]' | while read account; do
        print_color $CYAN "   $account"
    done
    
    # Return first account cleanly without debug output
    local first_account=$(echo $response | jq -r '.result[0]' 2>/dev/null)
    echo "$first_account"
}

# Function to get balance
get_balance() {
    local account=$1
    
    print_header "💰 CHECKING BALANCE"
    
    if [ -z "$account" ] || [ "$account" = "null" ]; then
        print_color $RED "❌ Invalid account provided for balance check"
        return 1
    fi
    
    print_color $YELLOW "Checking balance for account: $account"
    
    local response=$(curl -s -X POST -H "Content-Type: application/json" \
        --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"$account\",\"latest\"],\"id\":1}" \
        "$RPC_URL")
    
    echo "Balance response: $response"
    
    local balance_hex=$(echo $response | jq -r '.result' 2>/dev/null || echo "0x0")
    if [ "$balance_hex" = "null" ] || [ -z "$balance_hex" ]; then
        balance_hex="0x0"
    fi
    
    # Convert hex to decimal (handle potential bc absence)
    local balance_wei=0
    if command -v bc &> /dev/null; then
        balance_wei=$(echo "ibase=16; ${balance_hex#0x}" | bc 2>/dev/null || echo "0")
        local balance_eth=$(echo "scale=6; $balance_wei / 1000000000000000000" | bc 2>/dev/null || echo "0")
    else
        # Simple fallback without bc
        balance_wei=$(printf "%d" "$balance_hex" 2>/dev/null || echo "0")
        balance_eth="$(echo $balance_wei | sed 's/.\{18\}$/\.&/' | sed 's/^\.*/0./')"
    fi
    
    print_color $GREEN "✅ Account: $account"
    print_color $CYAN "   Balance: $balance_eth ETH ($balance_wei wei)"
}

# Function to deploy SimpleStorage contract
deploy_simple_storage() {
    local from_account=$1
    
    print_header "📄 DEPLOYING SIMPLESTORAGE CONTRACT"
    
    # SimpleStorage contract bytecode
    # contract SimpleStorage {
    #     uint256 private value;
    #     
    #     function setValue(uint256 _value) public {
    #         value = _value;
    #     }
    #     
    #     function getValue() public view returns (uint256) {
    #         return value;
    #     }
    # }
    local bytecode="0x608060405234801561001057600080fd5b50610156806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80632096525514610041578063552410771461005f575b600080fd5b61004961007d565b6040516100569190610086565b60405180910390f35b61006761008b565b6040516100749190610086565b60405180910390f35b6000805490565b60008054905090565b6000819050919050565b6100a08161008d565b82525050565b60006020820190506100bb6000830184610097565b9291505056fea26469706673582212201234567890123456789012345678901234567890123456789012345678901234564736f6c63430008070033"
    
    print_color $YELLOW "Deploying SimpleStorage contract..."
    print_color $CYAN "From account: $from_account"
    
    local response=$(curl -s -X POST -H "Content-Type: application/json" \
        --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_sendTransaction\",\"params\":[{\"from\":\"$from_account\",\"data\":\"$bytecode\",\"gas\":\"0x100000\"}],\"id\":1}" \
        "$RPC_URL")
    
    echo "Deploy response: $response"
    
    local tx_hash=$(echo $response | jq -r '.result' 2>/dev/null || echo "")
    if [ -n "$tx_hash" ] && [ "$tx_hash" != "null" ] && [ "$tx_hash" != "error" ]; then
        print_color $GREEN "✅ Contract deployment transaction sent!"
        print_color $CYAN "   Transaction Hash: $tx_hash"
        print_color $YELLOW "💡 This transaction should appear in DAG blocks and wave consensus!"
        
        echo "$tx_hash"
    else
        print_color $RED "❌ Contract deployment failed"
        echo "Response: $response"
        return 1
    fi
}

# Function to send a simple transaction
send_simple_transaction() {
    local from_account=$1
    
    print_header "💸 SENDING SIMPLE TRANSACTION"
    
    if [ -z "$from_account" ] || [ "$from_account" = "null" ]; then
        print_color $RED "❌ Invalid account provided for transaction"
        return 1
    fi
    
    # Send 0.001 ETH to a test address
    local to_address="0x742d35Cc6635C0532925a3b8D4B9D13E2c8b3958"
    local value="0x38D7EA4C68000"  # 0.001 ETH in wei
    
    print_color $YELLOW "Sending transaction..."
    print_color $CYAN "From: $from_account"
    print_color $CYAN "To: $to_address"
    print_color $CYAN "Value: 0.001 ETH"
    
    local response=$(curl -s -X POST -H "Content-Type: application/json" \
        --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_sendTransaction\",\"params\":[{\"from\":\"$from_account\",\"to\":\"$to_address\",\"value\":\"$value\"}],\"id\":1}" \
        "$RPC_URL")
    
    echo "Transaction response: $response"
    
    local tx_hash=$(echo $response | jq -r '.result' 2>/dev/null || echo "")
    if [ -n "$tx_hash" ] && [ "$tx_hash" != "null" ] && [ "$tx_hash" != "error" ]; then
        print_color $GREEN "✅ Simple transaction sent!"
        print_color $CYAN "   Transaction Hash: $tx_hash"
        print_color $YELLOW "💡 This transaction should appear in DAG blocks!"
        
        echo "$tx_hash"
    else
        print_color $RED "❌ Transaction failed"
        echo "Response: $response"
        return 1
    fi
}

# Function to deploy multiple test contracts
deploy_multiple_contracts() {
    local from_account=$1
    local count=${2:-5}
    
    print_header "📄 DEPLOYING MULTIPLE TEST CONTRACTS"
    
    print_color $YELLOW "Deploying $count test contracts to generate more EVM transactions..."
    
    local successful=0
    for i in $(seq 1 $count); do
        print_color $BLUE "Deploying contract $i/$count..."
        
        if deploy_simple_storage "$from_account" > /dev/null 2>&1; then
            successful=$((successful + 1))
            print_color $GREEN "✅ Contract $i deployed successfully"
        else
            print_color $RED "❌ Contract $i deployment failed"
        fi
        
        # Small delay between deployments
        sleep 1
    done
    
    print_color $GREEN "✅ Successfully deployed $successful/$count contracts"
    print_color $YELLOW "💡 All these transactions should be visible in DAG blocks and wave consensus!"
}

# Function to monitor DAG for our transactions
monitor_dag_transactions() {
    print_header "🔍 MONITORING DAG FOR TRANSACTIONS"
    
    print_color $YELLOW "Fetching recent transactions from DAG..."
    
    # Get transactions from DAG API
    local dag_response=$(curl -s "http://localhost:8080/transactions?count=20" || echo "")
    
    if [ -n "$dag_response" ]; then
        echo "Recent DAG transactions:"
        echo "$dag_response" | jq -r '.[] | "Hash: \(.hash) | From: \(.from) | To: \(.to) | Block: \(.blockHeight)"' 2>/dev/null || echo "$dag_response"
    else
        print_color $RED "❌ Could not fetch DAG transactions"
    fi
}

# Function to check wave consensus status
check_wave_consensus() {
    print_header "🌊 CHECKING WAVE CONSENSUS STATUS"
    
    print_color $YELLOW "Fetching wave consensus status..."
    
    local wave_response=$(curl -s "http://localhost:8081/wave/status" || echo "")
    
    if [ -n "$wave_response" ]; then
        echo "Wave consensus status:"
        echo "$wave_response"
    else
        print_color $RED "❌ Could not fetch wave consensus status"
    fi
}

# Function to show integration status
show_integration_status() {
    print_header "🔗 INTEGRATION STATUS"
    
    print_color $YELLOW "Checking all services..."
    
    # Check DAG Sync
    if curl -s "http://localhost:8080/status" > /dev/null 2>&1; then
        print_color $GREEN "✅ DAG Sync: Running (http://localhost:8080)"
    else
        print_color $RED "❌ DAG Sync: Not responsive"
    fi
    
    # Check Wave Consensus
    if curl -s "http://localhost:8081/wave/status" > /dev/null 2>&1; then
        print_color $GREEN "✅ Wave Consensus: Running (http://localhost:8081)"
    else
        print_color $RED "❌ Wave Consensus: Not responsive"
    fi
    
    # Check EVM RPC
    if curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
        "$RPC_URL" > /dev/null 2>&1; then
        print_color $GREEN "✅ EVM RPC: Running ($RPC_URL)"
    else
        print_color $RED "❌ EVM RPC: Not responsive"
    fi
}

# Main function
main() {
    print_header "🧪 BLAZEDAG SMART CONTRACT TESTING"
    
    # Check dependencies
    check_dependencies
    
    # Test connection
    test_connection
    
    # Show integration status
    show_integration_status
    
    # Get accounts - store output to handle debug printing properly
    local account_output=$(get_accounts)
    local first_account=$(echo "$account_output" | tail -n 1)
    
    if [ -z "$first_account" ] || [ "$first_account" = "null" ]; then
        print_color $RED "❌ No valid account found"
        exit 1
    fi
    
    print_color $BLUE "🔍 Using account: $first_account"
    
    # Check balance
    get_balance "$first_account"
    
    # Send simple transaction
    print_color $YELLOW "⏳ Waiting 2 seconds before sending transactions..."
    sleep 2
    send_simple_transaction "$first_account"
    
    # Deploy a single contract
    print_color $YELLOW "⏳ Waiting 3 seconds before deploying contract..."
    sleep 3
    deploy_simple_storage "$first_account"
    
    # Deploy multiple contracts for more testing
    print_color $YELLOW "⏳ Waiting 5 seconds before deploying multiple contracts..."
    sleep 5
    deploy_multiple_contracts "$first_account" 3
    
    # Monitor DAG transactions
    print_color $YELLOW "⏳ Waiting 5 seconds for transactions to be processed..."
    sleep 5
    monitor_dag_transactions
    
    # Check wave consensus
    check_wave_consensus
    
    print_header "🎉 SMART CONTRACT TESTING COMPLETED"
    print_color $GREEN "✅ EVM transactions sent to BlazeDAG integrated validators"
    print_color $YELLOW "💡 Check validator logs to see EVM transactions in DAG blocks and wave consensus"
    print_color $CYAN "📋 You can also check:"
    print_color $CYAN "   - DAG API: http://localhost:8080/transactions"
    print_color $CYAN "   - Wave API: http://localhost:8081/wave/status"
    print_color $CYAN "   - EVM RPC: $RPC_URL"
}

# Execute main function
main "$@" 