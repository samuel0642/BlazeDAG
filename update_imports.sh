#!/bin/bash

# Update all import paths from samuel0642 to CrossDAG
find . -type f -name "*.go" -exec sed -i 's|github.com/samuel0642/BlazeDAG|github.com/CrossDAG/BlazeDAG|g' {} \; 