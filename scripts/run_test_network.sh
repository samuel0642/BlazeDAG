#!/bin/bash

# Kill any existing blazedag processes
pkill -f blazedag

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the project
echo "Building BlazeDAG..."
go build -o bin/blazedag ./cmd/blazedag

# Start validator nodes
echo "Starting validator nodes..."

# Start validator 1
./bin/blazedag --validator --nodeid "validator1" --port 3000 --block-interval 5s --consensus-timeout 10s &
VALIDATOR1_PID=$!

# Start validator 2
./bin/blazedag --validator --nodeid "validator2" --port 3001 --block-interval 5s --consensus-timeout 10s &
VALIDATOR2_PID=$!

# Start validator 3
./bin/blazedag --validator --nodeid "validator3" --port 3002 --block-interval 5s --consensus-timeout 10s &
VALIDATOR3_PID=$!

# Start regular nodes
echo "Starting regular nodes..."

# Start regular node 1
./bin/blazedag --port 3003 &
NODE1_PID=$!

# Start regular node 2
./bin/blazedag --port 3004 &
NODE2_PID=$!

# Wait for user input to stop
echo "Network is running. Press Enter to stop..."
read

# Stop all nodes
echo "Stopping nodes..."
kill $VALIDATOR1_PID $VALIDATOR2_PID $VALIDATOR3_PID $NODE1_PID $NODE2_PID

echo "Network stopped." 