.PHONY: build run clean test demo-separation build-dagsync run-single-dag run-validator1-dag run-validator2-dag run-validator3-dag run-demo-dag stop-dag test-single-dag test-multi-dag help-dag

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
	-rm -f dagsync

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
	go build -o dagsync ./cmd/dagsync/simple_main.go
	@echo "DAG sync built successfully!"


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

