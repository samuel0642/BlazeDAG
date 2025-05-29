#!/bin/bash

# Script to run validator2 on server 52.53.192.236

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

echo "Starting Validator 2..."
echo "Listen: localhost:3002"
echo "Peers: localhost:3001,localhost:3003"

# Start validator2 with the updated config
./bin/blazedag --config config.validator2.yaml

./dagsync -id="validator2" -listen="localhost:3002" -peers="localhost:3001,localhost:3003"

echo "Validator2 stopped." 