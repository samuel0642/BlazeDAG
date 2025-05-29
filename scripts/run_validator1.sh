#!/bin/bash

# Script to run validator1 on server 54.183.204.244

# Kill any existing blazedag processes
pkill -f blazedag

# Create necessary directories
mkdir -p bin
mkdir -p data

# Build the project
echo "Building BlazeDAG..."
go build -o bin/blazedag ./cmd/blazedag

# Check if build was successful
if [ ! -f "bin/blazedag" ]; then
    echo "Build failed! Exiting..."
    exit 1
fi

echo "Starting Validator 1..."
echo "Listen: localhost:3001"
echo "Peers: localhost:3002,localhost:3003"

# Start validator1 with the updated config
./bin/blazedag --config config.validator1.yaml

echo "Validator1 stopped." 