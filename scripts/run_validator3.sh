#!/bin/bash

echo "Starting Validator 3..."
echo "Listen: localhost:3003"
echo "Peers: localhost:3001,localhost:3002"

./dagsync -id="validator3" -listen="localhost:3003" -peers="localhost:3001,localhost:3002" 