package evm

import (
	"math/big"
	"sync"
)

// GasManager handles gas management
type GasManager struct {
	baseGasPrice *big.Int
	minGasPrice  *big.Int
	maxGasPrice  *big.Int
	mu           sync.RWMutex
}

// NewGasManager creates a new gas manager
func NewGasManager(baseGasPrice *big.Int) *GasManager {
	return &GasManager{
		baseGasPrice: baseGasPrice,
		minGasPrice:  new(big.Int).Div(baseGasPrice, big.NewInt(2)),
		maxGasPrice:  new(big.Int).Mul(baseGasPrice, big.NewInt(2)),
	}
}

// CalculateGasPrice calculates the gas price based on network conditions
func (gm *GasManager) CalculateGasPrice(networkLoad float64) *big.Int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	// Adjust gas price based on network load
	multiplier := new(big.Float).SetFloat64(1.0 + networkLoad)
	gasPrice := new(big.Float).Mul(
		new(big.Float).SetInt(gm.baseGasPrice),
		multiplier,
	)

	// Convert to big.Int
	result, _ := gasPrice.Int(nil)

	// Ensure within bounds
	if result.Cmp(gm.minGasPrice) < 0 {
		return gm.minGasPrice
	}
	if result.Cmp(gm.maxGasPrice) > 0 {
		return gm.maxGasPrice
	}

	return result
}

// EstimateGas estimates the gas required for a transaction
func (gm *GasManager) EstimateGas(tx *Transaction) uint64 {
	// Base gas cost
	gas := uint64(21000)

	// Add gas for data
	if len(tx.Data) > 0 {
		gas += uint64(len(tx.Data)) * 16
	}

	// Add gas for contract creation
	if len(tx.To) == 0 {
		gas += 53000
	}

	return gas
}

// ValidateGasPrice validates a gas price
func (gm *GasManager) ValidateGasPrice(gasPrice *big.Int) bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	return gasPrice.Cmp(gm.minGasPrice) >= 0 && gasPrice.Cmp(gm.maxGasPrice) <= 0
}

// UpdateBaseGasPrice updates the base gas price
func (gm *GasManager) UpdateBaseGasPrice(baseGasPrice *big.Int) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.baseGasPrice = baseGasPrice
	gm.minGasPrice = new(big.Int).Div(baseGasPrice, big.NewInt(2))
	gm.maxGasPrice = new(big.Int).Mul(baseGasPrice, big.NewInt(2))
}

// GetBaseGasPrice returns the base gas price
func (gm *GasManager) GetBaseGasPrice() *big.Int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.baseGasPrice
}

// GetMinGasPrice returns the minimum gas price
func (gm *GasManager) GetMinGasPrice() *big.Int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.minGasPrice
}

// GetMaxGasPrice returns the maximum gas price
func (gm *GasManager) GetMaxGasPrice() *big.Int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.maxGasPrice
}

// CalculateTotalGasCost calculates the total gas cost for a transaction
func (gm *GasManager) CalculateTotalGasCost(gasPrice *big.Int, gasLimit uint64) *big.Int {
	return new(big.Int).Mul(gasPrice, big.NewInt(int64(gasLimit)))
}

// ValidateGasLimit validates a gas limit
func (gm *GasManager) ValidateGasLimit(gasLimit uint64) bool {
	// Minimum gas limit
	if gasLimit < 21000 {
		return false
	}

	// Maximum gas limit (8M)
	if gasLimit > 8000000 {
		return false
	}

	return true
} 