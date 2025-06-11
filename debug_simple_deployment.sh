#!/bin/bash

# Simple Smart Contract Deployment Debug
set -e

RPC_URL="http://localhost:8546"

echo "=== DEBUGGING SMART CONTRACT DEPLOYMENT ==="
echo

# Step 1: Test basic connectivity
echo "1. Testing EVM connectivity..."
RESPONSE=$(timeout 5 curl -s -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
    "$RPC_URL" || echo "timeout")

echo "Chain ID response: $RESPONSE"

if [[ "$RESPONSE" == "timeout" || "$RESPONSE" == "" ]]; then
    echo "❌ EVM RPC not responding"
    exit 1
fi

CHAIN_ID=$(echo "$RESPONSE" | jq -r '.result // "null"')
echo "✅ Chain ID: $CHAIN_ID"

# Step 2: Get accounts
echo "2. Getting accounts..."
ACCOUNTS_RESPONSE=$(timeout 5 curl -s -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}' \
    "$RPC_URL" || echo "timeout")

echo "Accounts response: $ACCOUNTS_RESPONSE"

if [[ "$ACCOUNTS_RESPONSE" == "timeout" ]]; then
    echo "❌ Failed to get accounts"
    exit 1
fi

DEPLOYER=$(echo "$ACCOUNTS_RESPONSE" | jq -r '.result[0] // "null"')
echo "✅ Deployer: $DEPLOYER"

# Step 3: Test gas estimation for simple bytecode
echo "3. Testing gas estimation..."
SIMPLE_BYTECODE="0x608060405234801561001057600080fd5b50610150806100206000396000f3fe"

GAS_RESPONSE=$(timeout 5 curl -s -X POST -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_estimateGas\",\"params\":[{\"from\":\"$DEPLOYER\",\"data\":\"$SIMPLE_BYTECODE\"}],\"id\":1}" \
    "$RPC_URL" || echo "timeout")

echo "Gas estimation response: $GAS_RESPONSE"

if [[ "$GAS_RESPONSE" == "timeout" ]]; then
    echo "❌ Gas estimation failed"
    exit 1
fi

GAS_ESTIMATE=$(echo "$GAS_RESPONSE" | jq -r '.result // "null"')
echo "✅ Gas estimate: $GAS_ESTIMATE"

# Step 4: Try simple transaction
echo "4. Testing simple deployment..."
DEPLOY_RESPONSE=$(timeout 10 curl -s -X POST -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_sendTransaction\",\"params\":[{\"from\":\"$DEPLOYER\",\"data\":\"$SIMPLE_BYTECODE\",\"gas\":\"$GAS_ESTIMATE\"}],\"id\":1}" \
    "$RPC_URL" || echo "timeout")

echo "Deploy response: $DEPLOY_RESPONSE"

if [[ "$DEPLOY_RESPONSE" == "timeout" ]]; then
    echo "❌ Deployment timed out"
    exit 1
fi

TX_HASH=$(echo "$DEPLOY_RESPONSE" | jq -r '.result // "null"')
echo "✅ Transaction hash: $TX_HASH"

# Step 5: Check receipt
echo "5. Checking transaction receipt..."
sleep 3

RECEIPT_RESPONSE=$(timeout 5 curl -s -X POST -H "Content-Type: application/json" \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getTransactionReceipt\",\"params\":[\"$TX_HASH\"],\"id\":1}" \
    "$RPC_URL" || echo "timeout")

echo "Receipt response: $RECEIPT_RESPONSE"

if [[ "$RECEIPT_RESPONSE" == "timeout" ]]; then
    echo "❌ Receipt check timed out"
    exit 1
fi

echo "🎉 Simple deployment test completed successfully!" 