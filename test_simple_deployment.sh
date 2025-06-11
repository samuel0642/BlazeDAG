#!/bin/bash

# Simple Smart Contract Deployment Test

set -e

RPC_URL="http://localhost:8545"

echo "=== SIMPLE SMART CONTRACT DEPLOYMENT TEST ==="

# Test 1: Basic connectivity
echo "1. Testing connectivity..."
CHAIN_ID=$(curl -s -m 5 -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
    "$RPC_URL" | jq -r '.result // "failed"')

if [ "$CHAIN_ID" = "failed" ]; then
    echo "❌ EVM RPC not responding"
    exit 1
fi
echo "✅ EVM RPC responding (Chain ID: $CHAIN_ID)"

# Test 2: Get accounts
echo "2. Getting accounts..."
ACCOUNTS=$(curl -s -m 5 -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}' \
    "$RPC_URL")

DEPLOYER=$(echo "$ACCOUNTS" | jq -r '.result[0] // "failed"')
if [ "$DEPLOYER" = "failed" ]; then
    echo "❌ Failed to get accounts"
    exit 1
fi
echo "✅ Deployer account: $DEPLOYER"

# Test 3: Gas estimation for simple contract
echo "3. Testing gas estimation..."
# Simple contract bytecode (just stores a number)
SIMPLE_BYTECODE="0x608060405234801561001057600080fd5b5060405161008e38038061008e8339818101604052810190610032919061004d565b80600081905550506100a3565b60008151905061004781610086565b92915050565b60006020828403121561005f57600080fd5b600061006d84828501610038565b91505092915050565b6000819050919050565b61008381610076565b811461008e57600080fd5b50565b600a8061009c6000396000f3fe6080604052600080fdfea2646970667358221220"

GAS_RESPONSE=$(curl -s -m 10 -X POST -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_estimateGas\",\"params\":[{\"from\":\"$DEPLOYER\",\"data\":\"$SIMPLE_BYTECODE\"}],\"id\":1}" \
    "$RPC_URL")

GAS_ESTIMATE=$(echo "$GAS_RESPONSE" | jq -r '.result // "failed"')
if [ "$GAS_ESTIMATE" = "failed" ]; then
    echo "❌ Gas estimation failed"
    echo "Response: $GAS_RESPONSE"
    exit 1
fi
echo "✅ Gas estimate: $GAS_ESTIMATE"

# Test 4: Deploy simple contract
echo "4. Deploying simple contract..."
DEPLOY_RESPONSE=$(timeout 15 curl -s -X POST -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_sendTransaction\",\"params\":[{\"from\":\"$DEPLOYER\",\"data\":\"$SIMPLE_BYTECODE\",\"gas\":\"$GAS_ESTIMATE\"}],\"id\":1}" \
    "$RPC_URL" 2>/dev/null || echo '{"error":"timeout"}')

echo "Deploy response: $DEPLOY_RESPONSE"

DEPLOY_TX=$(echo "$DEPLOY_RESPONSE" | jq -r '.result // "failed"')
if [ "$DEPLOY_TX" != "failed" ] && [ "$DEPLOY_TX" != "null" ]; then
    echo "✅ Deployment transaction sent: $DEPLOY_TX"
    
    # Wait for receipt
    echo "5. Waiting for transaction receipt..."
    for i in {1..5}; do
        sleep 2
        RECEIPT=$(curl -s -m 5 -X POST -H "Content-Type: application/json" \
            --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getTransactionReceipt\",\"params\":[\"$DEPLOY_TX\"],\"id\":1}" \
            "$RPC_URL")
        
        CONTRACT_ADDR=$(echo "$RECEIPT" | jq -r '.result.contractAddress // "null"')
        if [ "$CONTRACT_ADDR" != "null" ]; then
            echo "✅ Contract deployed at: $CONTRACT_ADDR"
            break
        fi
        echo "   Waiting... ($i/5)"
    done
else
    echo "❌ Deployment failed: $DEPLOY_RESPONSE"
fi

echo "=== TEST COMPLETE ===" 