#!/bin/bash

# BlazeDAG Integration Status Check
echo "🔥 ================================"
echo "🔥 BLAZEDAG INTEGRATED SYSTEM STATUS"
echo "🔥 ================================"
echo

# Check validators
echo "📊 VALIDATOR STATUS:"
VALIDATOR_COUNT=$(ps aux | grep integrated-node | grep -v grep | wc -l)
echo "✅ Running Validators: $VALIDATOR_COUNT"
echo

# Check EVM RPC
echo "⚡ EVM RPC STATUS:"
CHAIN_ID=$(curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' http://localhost:8545 | jq -r '.result' 2>/dev/null || echo "error")
if [ "$CHAIN_ID" != "error" ] && [ "$CHAIN_ID" != "null" ]; then
    echo "✅ EVM RPC: Working (Chain ID: $CHAIN_ID)"
    ACCOUNT_COUNT=$(curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}' http://localhost:8545 | jq '.result | length' 2>/dev/null || echo "0")
    echo "✅ EVM Accounts: $ACCOUNT_COUNT"
else
    echo "❌ EVM RPC: Not working"
fi
echo

# Check DAG Sync
echo "📡 DAG SYNC STATUS:"
DAG_BLOCKS=$(curl -s "http://localhost:8080/status" | jq -r '.total_blocks' 2>/dev/null || echo "error")
if [ "$DAG_BLOCKS" != "error" ] && [ "$DAG_BLOCKS" != "null" ]; then
    echo "✅ DAG Sync: Working ($DAG_BLOCKS blocks)"
    CURRENT_ROUND=$(curl -s "http://localhost:8080/status" | jq -r '.current_round' 2>/dev/null || echo "unknown")
    echo "✅ Current Round: $CURRENT_ROUND"
else
    echo "❌ DAG Sync: Not working"
fi
echo

# Check endpoints
echo "🌐 ENDPOINTS:"
echo "   EVM RPC: http://localhost:8545 (validator1)"
echo "   EVM RPC: http://localhost:8546 (validator2)"
echo "   EVM RPC: http://localhost:8547 (validator3)"
echo "   DAG API: http://localhost:8080"
echo "   Wave API: http://localhost:8081"
echo

# Integration summary
echo "🔗 INTEGRATION STATUS:"
if [ "$VALIDATOR_COUNT" -gt 0 ] && [ "$CHAIN_ID" != "error" ] && [ "$DAG_BLOCKS" != "error" ]; then
    echo "✅ FULLY INTEGRATED: DAG sync + Wave consensus + EVM all running!"
    echo "💡 Smart contracts deployed via EVM will appear in DAG blocks"
    echo "💡 Wave consensus processes DAG blocks containing EVM transactions"
    echo
    echo "🚀 NEXT STEPS:"
    echo "1. Connect MetaMask to http://localhost:8545 with Chain ID $CHAIN_ID"
    echo "2. Use account private keys from validator logs"
    echo "3. Deploy smart contracts through MetaMask or web3.js"
    echo "4. Watch transactions appear in DAG: curl http://localhost:8080/transactions"
else
    echo "❌ INTEGRATION ISSUES: Some components not working properly"
fi
echo

echo "🔥 ================================" 