#!/bin/bash

# Debug Smart Contract Deployment Issues
set -e

RPC_URL="http://localhost:8545"
CONTRACT_FILE="contracts/SimpleToken.sol"
CONTRACT_NAME="SimpleToken"

echo "=== DEBUGGING SMART CONTRACT DEPLOYMENT ==="
echo

# Step 1: Test basic connectivity
echo "1. Testing EVM connectivity..."
CHAIN_ID=$(curl -s -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
    "$RPC_URL" | jq -r '.result')
echo "Chain ID: $CHAIN_ID"

# Step 2: Get accounts
echo "2. Getting accounts..."
ACCOUNTS=$(curl -s -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}' \
    "$RPC_URL")
echo "Accounts response: $ACCOUNTS"

DEPLOYER=$(echo "$ACCOUNTS" | jq -r '.result[0]')
echo "Deployer: $DEPLOYER"

# Step 3: Check balance
echo "3. Check deployer balance..."
BALANCE=$(curl -s -X POST -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"$DEPLOYER\",\"latest\"],\"id\":1}" \
    "$RPC_URL")
echo "Balance response: $BALANCE"

# Step 4: Compile contract
echo "4. Compiling contract..."
if [ ! -f "$CONTRACT_FILE" ]; then
    echo "ERROR: Contract file not found: $CONTRACT_FILE"
    exit 1
fi

COMPILE_OUTPUT=$(solc --evm-version london --combined-json abi,bin "$CONTRACT_FILE" 2>&1)
if [ $? -ne 0 ]; then
    echo "ERROR: Compilation failed"
    echo "$COMPILE_OUTPUT"
    exit 1
fi

BYTECODE=$(echo "$COMPILE_OUTPUT" | jq -r ".contracts[\"$CONTRACT_FILE:$CONTRACT_NAME\"].bin")
echo "Bytecode length: ${#BYTECODE}"
echo "First 100 chars of bytecode: ${BYTECODE:0:100}..."

# Step 5: Test gas estimation
echo "5. Testing gas estimation..."
CONSTRUCTOR_PARAMS=$(printf "%064x" 1000000)
FULL_BYTECODE="0x${BYTECODE}${CONSTRUCTOR_PARAMS}"

echo "Full bytecode length: ${#FULL_BYTECODE}"

GAS_ESTIMATE=$(curl -s -X POST -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_estimateGas\",\"params\":[{\"from\":\"$DEPLOYER\",\"data\":\"$FULL_BYTECODE\"}],\"id\":1}" \
    "$RPC_URL")
echo "Gas estimate response: $GAS_ESTIMATE"

# Step 6: Test with smaller gas limit first
echo "6. Testing deployment with different gas limits..."

# Try with estimated gas + 50%
ESTIMATED_GAS=$(echo "$GAS_ESTIMATE" | jq -r '.result' 2>/dev/null)
if [ "$ESTIMATED_GAS" != "null" ] && [ -n "$ESTIMATED_GAS" ]; then
    echo "Using estimated gas: $ESTIMATED_GAS"
    GAS_LIMIT=$ESTIMATED_GAS
else
    echo "Gas estimation failed, using default: 0x200000"
    GAS_LIMIT="0x200000"
fi

echo "7. Attempting deployment with gas limit: $GAS_LIMIT"
DEPLOY_RESPONSE=$(timeout 45 curl -s -X POST -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_sendTransaction\",\"params\":[{\"from\":\"$DEPLOYER\",\"data\":\"$FULL_BYTECODE\",\"gas\":\"$GAS_LIMIT\"}],\"id\":1}" \
    "$RPC_URL" 2>&1)

echo "Deploy response: $DEPLOY_RESPONSE"

DEPLOY_TX=$(echo "$DEPLOY_RESPONSE" | jq -r '.result' 2>/dev/null)
if [ -n "$DEPLOY_TX" ] && [ "$DEPLOY_TX" != "null" ]; then
    echo "✅ Deployment transaction sent: $DEPLOY_TX"
    
    echo "8. Waiting for transaction receipt..."
    for i in {1..10}; do
        echo "   Attempt $i/10..."
        sleep 3
        
        RECEIPT=$(curl -s -X POST -H "Content-Type: application/json" \
            --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getTransactionReceipt\",\"params\":[\"$DEPLOY_TX\"],\"id\":1}" \
            "$RPC_URL")
        
        echo "   Receipt: $RECEIPT"
        
        CONTRACT_ADDR=$(echo "$RECEIPT" | jq -r '.result.contractAddress' 2>/dev/null)
        if [ -n "$CONTRACT_ADDR" ] && [ "$CONTRACT_ADDR" != "null" ]; then
            echo "✅ Contract deployed at: $CONTRACT_ADDR"
            break
        fi
    done
else
    echo "❌ Deployment failed"
    echo "Error details: $DEPLOY_RESPONSE"
fi

echo
echo "=== DEBUGGING COMPLETE ===" 