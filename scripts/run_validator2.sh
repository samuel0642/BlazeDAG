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

echo "Starting validator2 on 52.53.192.236:3000..."

# Start validator2 with the updated config
./bin/blazedag --config config.validator2.yaml

echo "Validator2 stopped." 