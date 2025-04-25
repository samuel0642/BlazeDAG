.PHONY: build run clean test demo-separation

# Build the BlazeDAG binary
build:
	go build -o bin/blazedag ./cmd/blazedag

# Run the BlazeDAG node
run: build
	./bin/blazedag

# Run as validator
run-validator: build
	./bin/blazedag --validator

# Clean build artifacts
clean:
	rm -rf bin/

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