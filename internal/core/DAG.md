# DAG Singleton Pattern

## Overview

The BlazeDAG project uses a singleton pattern for the Directed Acyclic Graph (DAG) to ensure consistency across all components of the system. This document explains how to use the DAG singleton.

## Usage

### Getting the DAG Instance

Always use `GetDAG()` from the `core` package to access the DAG instance:

```go
// Import the core package
import "github.com/CrossDAG/BlazeDAG/internal/core"

// Get the DAG instance
dag := core.GetDAG()

// Use the DAG
blocks := dag.GetRecentBlocks(10)
```

### Important Notes

1. **Never** create a new DAG instance with `NewDAG()` directly in your code. This will create an isolated DAG that won't be synchronized with other components.

2. The singleton pattern ensures that all components (consensus engine, block processor, etc.) work with the same DAG instance, preventing data inconsistencies.

3. The DAG singleton is thread-safe and can be accessed from multiple goroutines.

## Implementation Details

The DAG singleton is implemented in `internal/core/dag_singleton.go`. It uses a mutex to ensure thread-safe initialization and access to the global DAG instance.

If you need to reset the DAG instance (for testing purposes), you can use the `SetGlobalDAG()` function:

```go
// Import the core package
import "github.com/CrossDAG/BlazeDAG/internal/core"

// Create a new DAG and set it as the global instance
newDag := core.NewDAG()
core.SetGlobalDAG(newDag)
```

## Troubleshooting

If you encounter any issues with the DAG singleton, check the following:

1. Make sure you're using `GetDAG()` instead of `NewDAG()` to access the DAG instance.
2. Verify that all components are importing the correct `core` package.
3. Check for any race conditions or concurrent access issues.

If you need to add DAG functionality, modify the methods in `internal/core/dag.go` rather than creating custom DAG implementations. 