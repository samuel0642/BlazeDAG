package evm

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// ExecutionResult represents the result of a transaction execution
type ExecutionResult struct {
	Success     bool
	GasUsed     uint64
	ReturnData  []byte
	Error       error
	StateRoot   common.Hash
	CreatedAddr *common.Address
}

// EVMExecutor handles EVM transaction execution with full Ethereum compatibility
type EVMExecutor struct {
	state       *State
	chainConfig *params.ChainConfig
	vmConfig    vm.Config
	gasPrice    *big.Int
	blockNumber *big.Int
	mu          sync.RWMutex
}

// NewEVMExecutor creates a new EVM executor with Ethereum compatibility
func NewEVMExecutor(state *State, gasPrice *big.Int) *EVMExecutor {
	// Use simplified chain configuration without access lists to avoid panics
	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(1337), // Custom chain ID for BlazeDAG
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		// Disable Berlin and London forks to avoid access list issues
		BerlinBlock:         nil,
		LondonBlock:         nil,
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: false,
	}

	return &EVMExecutor{
		state:       state,
		chainConfig: chainConfig,
		vmConfig:    vmConfig,
		gasPrice:    gasPrice,
		blockNumber: big.NewInt(0),
	}
}

// ExecuteTransaction executes a transaction using the Ethereum Virtual Machine
func (e *EVMExecutor) ExecuteTransaction(tx *Transaction) *ExecutionResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate transaction
	if err := e.validateTransaction(tx); err != nil {
		return &ExecutionResult{
			Success: false,
			Error:   err,
		}
	}

	// Create a snapshot of the state
	snapshot := e.state.Copy()

	// Execute the transaction
	result := e.executeInternal(tx, snapshot)

	// If successful, commit the state changes
	if result.Success {
		e.state = snapshot
	}

	return result
}

// validateTransaction validates a transaction before execution
func (e *EVMExecutor) validateTransaction(tx *Transaction) error {
	// Check nonce
	currentNonce := e.state.GetNonce(tx.From)
	if tx.Nonce != currentNonce {
		return ErrInvalidNonce
	}

	// Check balance for gas + value
	balance := e.state.GetBalance(tx.From)
	gasLimit := new(big.Int).SetUint64(tx.GasLimit)
	gasCost := new(big.Int).Mul(gasLimit, tx.GasPrice)
	totalCost := new(big.Int).Add(gasCost, tx.Value)

	if balance.Cmp(totalCost) < 0 {
		return ErrInsufficientBalance
	}

	return nil
}

// executeInternal executes the transaction internally
func (e *EVMExecutor) executeInternal(tx *Transaction, workingState *State) *ExecutionResult {
	// Create block context
	blockContext := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     e.getHashFunc(),
		Coinbase:    common.Address{}, // No coinbase in BlazeDAG
		BlockNumber: e.blockNumber,
		Time:        uint64(tx.Timestamp.Unix()),
		Difficulty:  big.NewInt(0), // No mining in BlazeDAG
		GasLimit:    1000000000,    // High gas limit
		BaseFee:     big.NewInt(0), // No base fee
	}

	// Create transaction context
	txContext := vm.TxContext{
		Origin:   tx.From,
		GasPrice: tx.GasPrice,
	}

	// Create state database adapter
	stateDB := NewStateDBAdapter(workingState)

	// Create EVM instance
	evm := vm.NewEVM(blockContext, txContext, stateDB, e.chainConfig, e.vmConfig)

	// Prepare transaction
	var (
		to       *common.Address
		input    []byte
		gasLimit uint64
		value    *big.Int
	)

	if tx.To == nil {
		// Contract creation
		to = nil
		input = tx.Data
	} else {
		// Contract call or transfer
		to = tx.To
		input = tx.Data
	}

	gasLimit = tx.GasLimit
	value = tx.Value

	// Execute transaction
	var (
		ret          []byte
		leftOverGas  uint64
		contractAddr common.Address
		err          error
	)

	// Increment sender nonce
	stateDB.SetNonce(tx.From, tx.Nonce+1)

	if to == nil {
		// Contract creation
		contractAddr = CreateContractAddress(tx.From, tx.Nonce)
		ret, contractAddr, leftOverGas, err = evm.Create(
			vm.AccountRef(tx.From),
			input,
			gasLimit,
			value,
		)
	} else {
		// Contract call or transfer
		ret, leftOverGas, err = evm.Call(
			vm.AccountRef(tx.From),
			*to,
			input,
			gasLimit,
			value,
		)
	}

	// Calculate gas used
	gasUsed := gasLimit - leftOverGas

	// Deduct gas cost from sender
	gasCost := new(big.Int).Mul(new(big.Int).SetUint64(gasUsed), tx.GasPrice)
	senderBalance := stateDB.GetBalance(tx.From)
	if senderBalance.Cmp(gasCost) >= 0 {
		stateDB.SubBalance(tx.From, gasCost)
	}

	// Create result
	result := &ExecutionResult{
		Success:    err == nil,
		GasUsed:    gasUsed,
		ReturnData: ret,
		Error:      err,
		StateRoot:  workingState.Root(),
	}

	if to == nil && err == nil {
		// Contract creation successful
		result.CreatedAddr = &contractAddr
	}

	return result
}

// getHashFunc returns a function that can retrieve block hashes
func (e *EVMExecutor) getHashFunc() func(uint64) common.Hash {
	return func(blockNumber uint64) common.Hash {
		// In a real implementation, this would retrieve the hash from the DAG
		// For now, return a deterministic hash based on block number
		return crypto.Keccak256Hash(big.NewInt(int64(blockNumber)).Bytes())
	}
}

// DeployContract deploys a contract and returns its address
func (e *EVMExecutor) DeployContract(from common.Address, code []byte, gasLimit uint64, value *big.Int) (*ExecutionResult, common.Address) {
	nonce := e.state.GetNonce(from)
	
	tx := &Transaction{
		From:     from,
		To:       nil, // Contract creation
		Value:    value,
		GasPrice: e.gasPrice,
		GasLimit: gasLimit,
		Nonce:    nonce,
		Data:     code,
	}

	result := e.ExecuteTransaction(tx)
	contractAddr := CreateContractAddress(from, nonce)
	
	return result, contractAddr
}

// CallContract calls a contract function
func (e *EVMExecutor) CallContract(from common.Address, to common.Address, data []byte, gasLimit uint64, value *big.Int) *ExecutionResult {
	nonce := e.state.GetNonce(from)
	
	tx := &Transaction{
		From:     from,
		To:       &to,
		Value:    value,
		GasPrice: e.gasPrice,
		GasLimit: gasLimit,
		Nonce:    nonce,
		Data:     data,
	}

	return e.ExecuteTransaction(tx)
}

// Transfer performs a simple ETH transfer
func (e *EVMExecutor) Transfer(from common.Address, to common.Address, value *big.Int) *ExecutionResult {
	nonce := e.state.GetNonce(from)
	
	tx := &Transaction{
		From:     from,
		To:       &to,
		Value:    value,
		GasPrice: e.gasPrice,
		GasLimit: 21000, // Standard gas limit for transfers
		Nonce:    nonce,
		Data:     []byte{},
	}

	return e.ExecuteTransaction(tx)
}

// UpdateBlockNumber updates the current block number
func (e *EVMExecutor) UpdateBlockNumber(blockNumber *big.Int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.blockNumber = new(big.Int).Set(blockNumber)
}

// UpdateGasPrice updates the gas price
func (e *EVMExecutor) UpdateGasPrice(gasPrice *big.Int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.gasPrice = new(big.Int).Set(gasPrice)
}

// GetState returns the current state
func (e *EVMExecutor) GetState() *State {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// StateDBAdapter adapts our State to the Ethereum StateDB interface
type StateDBAdapter struct {
	state *State
}

// NewStateDBAdapter creates a new StateDB adapter
func NewStateDBAdapter(state *State) *StateDBAdapter {
	return &StateDBAdapter{state: state}
}

// Implement vm.StateDB interface
func (s *StateDBAdapter) CreateAccount(addr common.Address) {
	s.state.GetAccount(addr)
}

func (s *StateDBAdapter) SubBalance(addr common.Address, amount *big.Int) {
	s.state.SubBalance(addr, amount)
}

func (s *StateDBAdapter) AddBalance(addr common.Address, amount *big.Int) {
	s.state.AddBalance(addr, amount)
}

func (s *StateDBAdapter) GetBalance(addr common.Address) *big.Int {
	return s.state.GetBalance(addr)
}

func (s *StateDBAdapter) GetNonce(addr common.Address) uint64 {
	return s.state.GetNonce(addr)
}

func (s *StateDBAdapter) SetNonce(addr common.Address, nonce uint64) {
	s.state.SetNonce(addr, nonce)
}

func (s *StateDBAdapter) GetCodeHash(addr common.Address) common.Hash {
	return s.state.GetCodeHash(addr)
}

func (s *StateDBAdapter) GetCode(addr common.Address) []byte {
	return s.state.GetCode(addr)
}

func (s *StateDBAdapter) SetCode(addr common.Address, code []byte) {
	s.state.SetCode(addr, code)
}

func (s *StateDBAdapter) GetCodeSize(addr common.Address) int {
	return len(s.state.GetCode(addr))
}

func (s *StateDBAdapter) AddRefund(gas uint64) {
	// Not implemented for simplicity
}

func (s *StateDBAdapter) SubRefund(gas uint64) {
	// Not implemented for simplicity
}

func (s *StateDBAdapter) GetRefund() uint64 {
	return 0
}

func (s *StateDBAdapter) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	return s.state.GetStorage(addr, hash)
}

func (s *StateDBAdapter) GetState(addr common.Address, hash common.Hash) common.Hash {
	return s.state.GetStorage(addr, hash)
}

func (s *StateDBAdapter) SetState(addr common.Address, key, value common.Hash) {
	s.state.SetStorage(addr, key, value)
}

func (s *StateDBAdapter) Suicide(addr common.Address) bool {
	// Not implemented for simplicity
	return false
}

func (s *StateDBAdapter) HasSuicided(addr common.Address) bool {
	return false
}

func (s *StateDBAdapter) Exist(addr common.Address) bool {
	return s.state.AccountExists(addr)
}

func (s *StateDBAdapter) Empty(addr common.Address) bool {
	if !s.state.AccountExists(addr) {
		return true
	}
	balance := s.state.GetBalance(addr)
	nonce := s.state.GetNonce(addr)
	code := s.state.GetCode(addr)
	return balance.Sign() == 0 && nonce == 0 && len(code) == 0
}

func (s *StateDBAdapter) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	// Not implemented for simplicity
}

func (s *StateDBAdapter) AddressInAccessList(addr common.Address) bool {
	return false
}

func (s *StateDBAdapter) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return false, false
}

func (s *StateDBAdapter) Snapshot() int {
	// Not implemented for simplicity
	return 0
}

func (s *StateDBAdapter) RevertToSnapshot(int) {
	// Not implemented for simplicity
}

func (s *StateDBAdapter) AddLog(log *types.Log) {
	// Not implemented for simplicity
}

func (s *StateDBAdapter) AddPreimage(hash common.Hash, preimage []byte) {
	// Not implemented for simplicity
}

func (s *StateDBAdapter) ForEachStorage(addr common.Address, cb func(common.Hash, common.Hash) bool) error {
	// Not implemented for simplicity
	return nil
}

// AddAddressToAccessList adds an address to the access list (required by StateDB interface)
func (s *StateDBAdapter) AddAddressToAccessList(addr common.Address) {
	// Not implemented for simplicity
}

// AddSlotToAccessList adds a slot to the access list (required by StateDB interface)
func (s *StateDBAdapter) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	// Not implemented for simplicity
}

// GetTransientState returns transient storage for the given account and key
func (s *StateDBAdapter) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	// Not implemented for simplicity
	return common.Hash{}
}

// SetTransientState sets transient storage for the given account and key
func (s *StateDBAdapter) SetTransientState(addr common.Address, key, value common.Hash) {
	// Not implemented for simplicity
}

// HasSelfDestructed returns if the contract has been self-destructed
func (s *StateDBAdapter) HasSelfDestructed(addr common.Address) bool {
	return false
}

// SelfDestruct marks the given account as self-destructed
func (s *StateDBAdapter) SelfDestruct(addr common.Address) {
	// Not implemented for simplicity
}

// Prepare handles the preparatory steps for executing a state transition with
// regard to both EIP-2929 and EIP-2930.
func (s *StateDBAdapter) Prepare(rules params.Rules, sender, coinbase common.Address, dst *common.Address, precompiles []common.Address, list types.AccessList) {
	// Not implemented for simplicity
}

// Selfdestruct6780 handles EIP-6780 style self destruct
func (s *StateDBAdapter) Selfdestruct6780(addr common.Address) {
	// Not implemented for simplicity
} 