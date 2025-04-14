#!/bin/bash

# Test timeout configurations using the CLI
echo "Starting automated timeout tests..."

# Create test directory
TEST_DIR="/tmp/blazedag_test_$(date +%s)"
mkdir -p $TEST_DIR
echo "Test directory: $TEST_DIR"

# Function to run test with specific timeout
run_test() {
    local timeout=$1
    local config_file="$TEST_DIR/config_${timeout}.toml"
    
    echo "Testing with timeout=${timeout}s"
    
    # Create config with specific timeout
    cat > $config_file << EOF
[consensus]
timeout_propose = "${timeout}s"
timeout_prevote = "1s"
timeout_precommit = "1s"
timeout_commit = "1s"
EOF
    
    # Initialize node
    echo "Initializing node..."
    ./blazedag init --config $config_file --home $TEST_DIR/node_${timeout}
    if [ $? -ne 0 ]; then
        echo "Failed to initialize node with timeout=${timeout}s"
        return 1
    fi
    
    # Start node with timeout
    echo "Starting node (timeout=${timeout}s)..."
    timeout ${timeout}s ./blazedag start --config $config_file --home $TEST_DIR/node_${timeout} &
    PID=$!
    
    # Wait for process
    wait $PID
    STATUS=$?
    
    # Check result
    if [ $STATUS -eq 124 ]; then
        echo "✅ Test passed: Node timed out as expected after ${timeout}s"
    else
        echo "❌ Test failed: Node exited with status $STATUS (expected timeout)"
        return 1
    fi
    
    return 0
}

# Run tests with different timeouts
for timeout in 1 2 3 5; do
    run_test $timeout
    if [ $? -ne 0 ]; then
        echo "❌ Overall test failed at timeout=${timeout}s"
        exit 1
    fi
done

# Cleanup
echo "Cleaning up..."
rm -rf $TEST_DIR

echo "✅ All timeout tests passed!"
exit 0 