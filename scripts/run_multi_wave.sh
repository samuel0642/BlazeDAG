#!/bin/bash

echo "=== BlazeDAG Multi-Wave Consensus Demo ==="
echo "This demo runs 1 DAG sync + 3 Wave Consensus validators"
echo ""

# Kill any existing processes
pkill -f dagsync 2>/dev/null || true
pkill -f wave-consensus 2>/dev/null || true
sleep 1

# Build if needed
if [ ! -f "dagsync" ] || [ ! -f "wave-consensus" ]; then
    echo "Building binaries..."
    go build -o dagsync ./cmd/dagsync/simple_main.go
    go build -o wave-consensus ./cmd/wave/main.go
fi

echo "1. Starting DAG Sync (localhost:4001)..."
./dagsync -id="validator1" -listen="localhost:4001" -peers="localhost:4002,localhost:4003" &
DAG_PID=$!
sleep 3

echo "2. Starting Wave Consensus Validator 1 (localhost:6001)..."
./wave-consensus -id="wave-validator1" -dag-addr="localhost:4001" -wave-listen="localhost:6001" -wave-peers="localhost:6002,localhost:6003" &
WAVE1_PID=$!
sleep 2

echo "3. Starting Wave Consensus Validator 2 (localhost:6002)..."
./wave-consensus -id="wave-validator2" -dag-addr="localhost:4001" -wave-listen="localhost:6002" -wave-peers="localhost:6001,localhost:6003" &
WAVE2_PID=$!
sleep 2

echo "4. Starting Wave Consensus Validator 3 (localhost:6003)..."
./wave-consensus -id="wave-validator3" -dag-addr="localhost:4001" -wave-listen="localhost:6003" -wave-peers="localhost:6001,localhost:6002" &
WAVE3_PID=$!
sleep 3

echo ""
echo "=== Multi-Wave Consensus Running ==="
echo "DAG Sync: localhost:4001 (HTTP API: localhost:5001)"
echo "Wave Validator 1: localhost:6001"
echo "Wave Validator 2: localhost:6002" 
echo "Wave Validator 3: localhost:6003"
echo ""
echo "All validators should be:"
echo "✅ Fetching blocks from DAG sync"
echo "✅ Voting on blocks in waves"
echo "✅ Reaching consensus with 3/3 votes"
echo ""
echo "Press Ctrl+C to stop all services..."

# Wait for interrupt
trap 'echo -e "\n=== Stopping All Services ==="; kill $DAG_PID $WAVE1_PID $WAVE2_PID $WAVE3_PID 2>/dev/null; pkill -f dagsync 2>/dev/null; pkill -f wave-consensus 2>/dev/null; echo "All services stopped."; exit 0' INT

# Keep script running
while true; do
    sleep 1
done 