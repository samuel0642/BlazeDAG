.PHONY: build run clean test demo-separation build-dagsync run-single-dag run-validator1-dag run-validator2-dag run-validator3-dag run-demo-dag stop-dag test-single-dag test-multi-dag help-dag build-combined build-wave run-combined-single test-combined help-wave

# Build the BlazeDAG binary
build:
	go build -o bin/blazedag ./cmd/blazedag
	./bin/blazedag -config config.validator1.yaml

# Run the BlazeDAG node
run: build
	./bin/blazedag

# Run as validator
run-validator: build
	./bin/blazedag --validator

# Clean build artifacts
clean:
	rm -rf bin/
	-pkill -f dagsync 2>/dev/null || true
	-pkill -f blazedag-combined 2>/dev/null || true
	-pkill -f wave-consensus 2>/dev/null || true
	-rm -f dagsync blazedag-combined wave-consensus

# Run tests
test:
	go test ./...

# Run with specific port
run-port: build
	./bin/blazedag --port $(port)

# Run with genesis file
run-genesis: build
	./bin/blazedag --genesis $(genesis)

# Run the separation demo
demo-separation:
	@echo "Running component separation demo..."
	@go run scripts/demo_separation.go

# ===== DAG SYNC TARGETS =====

# Build the DAG sync binary
build-dagsync:
	@echo "Building DAG sync..."
	go build -o dagsync ./cmd/dagsync
	@echo "DAG sync built successfully!"

# Run single validator (for testing DAG sync)
run-single-dag: build-dagsync
	@echo "Running single DAG validator..."
	./dagsync -id="validator1" -listen="localhost:4001"

# Run individual DAG validators (use in separate terminals)
run-validator1-dag: build-dagsync
	@echo "Starting DAG Validator 1..."
	./dagsync -id="validator1" -listen="localhost:4001" -peers="localhost:4002,localhost:4003"

run-validator2-dag: build-dagsync
	@echo "Starting DAG Validator 2..."
	./dagsync -id="validator2" -listen="localhost:4002" -peers="localhost:4001,localhost:4003"

run-validator3-dag: build-dagsync
	@echo "Starting DAG Validator 3..."
	./dagsync -id="validator3" -listen="localhost:4003" -peers="localhost:4001,localhost:4002"

# Run demo with all 3 DAG validators
run-demo-dag: build-dagsync
	@echo "Running DAG sync demo..."
	./scripts/demo_dagsync.sh

# Stop all running DAG validators
stop-dag:
	@echo "Stopping all DAG validators..."
	-pkill -f dagsync 2>/dev/null || true
	@echo "All DAG validators stopped!"

# Test single DAG validator for 10 seconds
test-single-dag: build-dagsync
	@echo "Testing single DAG validator for 10 seconds..."
	timeout 10s ./dagsync -id="test-validator" -listen="localhost:5001" || echo "Test completed!"

# Test multi-DAG validator setup for 15 seconds
test-multi-dag: build-dagsync
	@echo "Testing multi-DAG validator setup..."
	@echo "Starting DAG validator 1..."
	./dagsync -id="v1" -listen="localhost:5001" -peers="localhost:5002,localhost:5003" &
	sleep 1
	@echo "Starting DAG validator 2..."
	./dagsync -id="v2" -listen="localhost:5002" -peers="localhost:5001,localhost:5003" &
	sleep 1
	@echo "Starting DAG validator 3..."
	./dagsync -id="v3" -listen="localhost:5003" -peers="localhost:5001,localhost:5002" &
	@echo "Running for 15 seconds..."
	sleep 15
	@echo "Stopping test DAG validators..."
	-pkill -f dagsync 2>/dev/null || true
	@echo "Multi-DAG validator test completed!"

# ===== WAVE CONSENSUS TARGETS =====

# Build the combined DAG+Wave binary
build-combined:
	@echo "Building combined DAG sync + Wave consensus..."
	go build -o blazedag-combined ./cmd/combined/main.go
	@echo "Combined binary built successfully!"

# Build the wave-only binary
build-wave:
	@echo "Building wave consensus only..."
	go build -o wave-consensus ./cmd/wave
	@echo "Wave consensus binary built successfully!"

# Run combined DAG sync + Wave consensus (single validator)
run-combined-single: build-combined
	@echo "Running combined DAG sync + Wave consensus..."
	./blazedag-combined -id="combined-validator1" \
		-dag-listen="localhost:4001" \
		-wave-listen="localhost:6001"

# Test combined system for 20 seconds
test-combined: build-combined
	@echo "Testing combined DAG sync + Wave consensus..."
	timeout 20s ./blazedag-combined -id="test-combined" \
		-dag-listen="localhost:7001" \
		-wave-listen="localhost:7002" || echo "Combined test completed!"

# Show wave consensus help
help-wave:
	@echo "Wave Consensus Commands:"
	@echo ""
	@echo "Build:"
	@echo "  make build-combined      - Build combined DAG sync + Wave consensus"
	@echo "  make build-wave          - Build wave consensus only"
	@echo ""
	@echo "Single Combined Validator:"
	@echo "  make run-combined-single - Run combined validator (DAG + Wave)"
	@echo ""
	@echo "Testing:"
	@echo "  make test-combined       - Test combined system for 20s"
	@echo ""
	@echo "Manual Commands:"
	@echo "  Combined: ./blazedag-combined -id=validator1 -dag-listen=localhost:4001 -wave-listen=localhost:6001"
	@echo "  Wave Only: ./wave-consensus -id=wave1 -dag-addr=localhost:4001 -wave-listen=localhost:6001"

# Show DAG sync help
help-dag:
	@echo "BlazeDAG Sync Commands:"
	@echo ""
	@echo "Build:"
	@echo "  make build-dagsync       - Build the DAG sync binary"
	@echo ""
	@echo "Single Validator:"
	@echo "  make run-single-dag      - Run single DAG validator (testing)"
	@echo ""
	@echo "Multi-Validator (run in separate terminals):"
	@echo "  make run-validator1-dag  - Start DAG validator 1"
	@echo "  make run-validator2-dag  - Start DAG validator 2"
	@echo "  make run-validator3-dag  - Start DAG validator 3"
	@echo ""
	@echo "Demo & Testing:"
	@echo "  make run-demo-dag        - Run automated demo with 3 DAG validators"
	@echo "  make test-single-dag     - Test single DAG validator for 10s"
	@echo "  make test-multi-dag      - Test 3 DAG validators for 15s"
	@echo ""
	@echo "Utility:"
	@echo "  make stop-dag            - Stop all running DAG validators"
	@echo "  make help-dag            - Show this help"
	@echo ""
	@echo "Manual Commands:"
	@echo "  ./dagsync -id=validator1 -listen=localhost:4001 -peers=localhost:4002,localhost:4003"

# Overall help
help:
	@echo "BlazeDAG Makefile Commands:"
	@echo ""
	@echo "=== Original BlazeDAG ==="
	@echo "  make build           - Build BlazeDAG binary"
	@echo "  make run             - Run BlazeDAG node"
	@echo "  make run-validator   - Run as validator"
	@echo "  make test            - Run tests"
	@echo "  make clean           - Clean up"
	@echo ""
	@echo "=== DAG Sync ==="
	@echo "  make help-dag        - Show DAG sync commands"
	@echo "  make test-multi-dag  - Quick test of DAG sync"
	@echo "  make run-demo-dag    - Run DAG sync demo"
	@echo ""
	@echo "=== Wave Consensus (NEW) ==="
	@echo "  make help-wave       - Show wave consensus commands"
	@echo "  make test-combined   - Quick test of combined system"
	@echo ""
	@echo "Use 'make help-dag' or 'make help-wave' for detailed commands"

