.PHONY: build test-all test-network test-consensus test-transaction test-dag test clean

# Build the main binary
build:
	go build -o blazedag ./cmd/blazedag


# Run network service tests
test-network:
	go test -v ./internal/network/...

# Run consensus engine tests
test-consensus:
	go test -v ./internal/consensus/...

# Run transaction pool tests
test-transaction:
	go test -v ./internal/transaction/... 

# Run DAG core tests
test-dag:
	go test -v ./internal/dag/...

# Run all tests with timeouts
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f blazedag
	rm -rf /tmp/blazedag_test_* 