#!/bin/bash

echo "=== BlazeDAG Sync Demo ==="
echo "This demo shows DAG synchronization between validators"
echo ""

# Kill any existing processes
pkill -f dagsync 2>/dev/null || true
sleep 1

echo "1. Starting Validator 1 (localhost:4001)..."
./dagsync -id="validator1" -listen="localhost:4001" -peers="localhost:4002,localhost:4003" &
VALIDATOR1_PID=$!
sleep 2

echo "2. Starting Validator 2 (localhost:4002)..."
./dagsync -id="validator2" -listen="localhost:4002" -peers="localhost:4001,localhost:4003" &
VALIDATOR2_PID=$!
sleep 2

echo "3. Starting Validator 3 (localhost:4003)..."
./dagsync -id="validator3" -listen="localhost:4003" -peers="localhost:4001,localhost:4002" &
VALIDATOR3_PID=$!
sleep 3

echo ""
echo "=== All 3 validators are now running and syncing DAG ==="
echo "- Each validator creates blocks every 2 seconds"
echo "- Blocks reference previous round blocks to form DAG"
echo "- Validators broadcast blocks to each other"
echo "- DAG grows with proper round progression"
echo ""
echo "Let the demo run for 20 seconds..."
sleep 20

echo ""
echo "=== Stopping Demo ==="
kill $VALIDATOR1_PID $VALIDATOR2_PID $VALIDATOR3_PID 2>/dev/null || true
pkill -f dagsync 2>/dev/null || true

echo "Demo completed!"
echo ""
echo "What you should have seen:"
echo "- 3 validators starting up"
echo "- Each creating blocks at different rounds"
echo "- Blocks being broadcasted and received"
echo "- DAG height growing with proper references"
echo "- Network connectivity messages" 