package api

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/CrossDAG/BlazeDAG/internal/evm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// EVMRPCServer provides Ethereum-compatible JSON-RPC API
type EVMRPCServer struct {
	executor        *evm.EVMExecutor
	keystore        *evm.Keystore
	chainID         *big.Int
	currentBlockNum *big.Int
	
	// Transaction management
	pendingTxs     map[string]*evm.Transaction
	txReceipts     map[string]*TransactionReceipt
	pendingTxChan  chan *evm.Transaction
	mu             sync.RWMutex
}

// TransactionReceipt represents a transaction receipt
type TransactionReceipt struct {
	TransactionHash   string `json:"transactionHash"`
	TransactionIndex  string `json:"transactionIndex"`
	BlockHash         string `json:"blockHash"`
	BlockNumber       string `json:"blockNumber"`
	From              string `json:"from"`
	To                string `json:"to,omitempty"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	GasUsed           string `json:"gasUsed"`
	ContractAddress   string `json:"contractAddress,omitempty"`
	Logs              []interface{} `json:"logs"`
	LogsBloom         string `json:"logsBloom"`
	Status            string `json:"status"`
}

// NewEVMRPCServer creates a new EVM RPC server
func NewEVMRPCServer(executor *evm.EVMExecutor, keystore *evm.Keystore, chainID *big.Int) *EVMRPCServer {
	return &EVMRPCServer{
		executor:        executor,
		keystore:        keystore,
		chainID:         chainID,
		currentBlockNum: big.NewInt(1),
		pendingTxs:      make(map[string]*evm.Transaction),
		txReceipts:      make(map[string]*TransactionReceipt),
		pendingTxChan:   make(chan *evm.Transaction, 1000),
	}
}

// ServeHTTP handles HTTP requests
func (s *EVMRPCServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, req.ID, -32700, "Parse error")
		return
	}

	response := s.handleRequest(&req)
	json.NewEncoder(w).Encode(response)
}

// handleRequest processes a JSON-RPC request
func (s *EVMRPCServer) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "eth_chainId":
		return s.chainId(req)
	case "eth_accounts":
		return s.accounts(req)
	case "eth_getBalance":
		return s.getBalance(req)
	case "eth_getTransactionCount":
		return s.getTransactionCount(req)
	case "eth_sendTransaction":
		return s.sendTransaction(req)
	case "eth_call":
		return s.call(req)
	case "eth_estimateGas":
		return s.estimateGas(req)
	case "eth_getCode":
		return s.getCode(req)
	case "eth_blockNumber":
		return s.blockNumber(req)
	case "eth_getBlockByNumber":
		return s.getBlockByNumber(req)
	case "eth_getTransactionReceipt":
		return s.getTransactionReceipt(req)
	case "eth_gasPrice":
		return s.gasPrice(req)
	case "net_version":
		return s.netVersion(req)
	case "web3_clientVersion":
		return s.clientVersion(req)
	default:
		return s.writeErrorResponse(req.ID, -32601, "Method not found")
	}
}

// chainId returns the chain ID
func (s *EVMRPCServer) chainId(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  hexutil.EncodeBig(s.chainID),
	}
}

// accounts returns available accounts
func (s *EVMRPCServer) accounts(req *JSONRPCRequest) *JSONRPCResponse {
	addresses := s.keystore.ListAddresses()
	result := make([]string, len(addresses))
	for i, addr := range addresses {
		result[i] = addr.Hex()
	}
	
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// getBalance returns the balance of an account
func (s *EVMRPCServer) getBalance(req *JSONRPCRequest) *JSONRPCResponse {
	if len(req.Params) < 1 {
		return s.writeErrorResponse(req.ID, -32602, "Invalid params")
	}

	addrStr, ok := req.Params[0].(string)
	if !ok {
		return s.writeErrorResponse(req.ID, -32602, "Invalid address")
	}

	addr := common.HexToAddress(addrStr)
	balance := s.executor.GetState().GetBalance(addr)

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  hexutil.EncodeBig(balance),
	}
}

// getTransactionCount returns the nonce of an account
func (s *EVMRPCServer) getTransactionCount(req *JSONRPCRequest) *JSONRPCResponse {
	if len(req.Params) < 1 {
		return s.writeErrorResponse(req.ID, -32602, "Invalid params")
	}

	addrStr, ok := req.Params[0].(string)
	if !ok {
		return s.writeErrorResponse(req.ID, -32602, "Invalid address")
	}

	addr := common.HexToAddress(addrStr)
	nonce := s.executor.GetState().GetNonce(addr)

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  hexutil.EncodeUint64(nonce),
	}
}

// sendTransaction sends a transaction (asynchronous - adds to pending pool)
func (s *EVMRPCServer) sendTransaction(req *JSONRPCRequest) *JSONRPCResponse {
	if len(req.Params) < 1 {
		return s.writeErrorResponse(req.ID, -32602, "Invalid params")
	}

	txParams, ok := req.Params[0].(map[string]interface{})
	if !ok {
		return s.writeErrorResponse(req.ID, -32602, "Invalid transaction params")
	}

	// Parse transaction
	tx, err := s.parseTransaction(txParams)
	if err != nil {
		return s.writeErrorResponse(req.ID, -32602, err.Error())
	}

	// Basic validation (without execution)
	if err := s.validateTransactionBasic(tx); err != nil {
		return s.writeErrorResponse(req.ID, -32602, err.Error())
	}

	// Create transaction hash
	txHash := tx.CreateHash()
	tx.Hash = txHash

	// Add to pending transactions
	s.mu.Lock()
	s.pendingTxs[txHash.Hex()] = tx
	s.mu.Unlock()

	// Send to processing channel (non-blocking)
	select {
	case s.pendingTxChan <- tx:
		// Transaction queued for processing
	default:
		// Channel full, transaction will be processed later
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  txHash.Hex(),
	}
}

// call executes a message call
func (s *EVMRPCServer) call(req *JSONRPCRequest) *JSONRPCResponse {
	if len(req.Params) < 1 {
		return s.writeErrorResponse(req.ID, -32602, "Invalid params")
	}

	txParams, ok := req.Params[0].(map[string]interface{})
	if !ok {
		return s.writeErrorResponse(req.ID, -32602, "Invalid call params")
	}

	// Parse call parameters
	tx, err := s.parseCall(txParams)
	if err != nil {
		return s.writeErrorResponse(req.ID, -32602, err.Error())
	}

	// Execute call without state changes
	stateCopy := s.executor.GetState().Copy()
	executor := evm.NewEVMExecutor(stateCopy, big.NewInt(1000000000))
	result := executor.ExecuteTransaction(tx)

	if !result.Success {
		return s.writeErrorResponse(req.ID, -32000, result.Error.Error())
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  hexutil.Encode(result.ReturnData),
	}
}

// estimateGas estimates gas for a transaction
func (s *EVMRPCServer) estimateGas(req *JSONRPCRequest) *JSONRPCResponse {
	if len(req.Params) < 1 {
		return s.writeErrorResponse(req.ID, -32602, "Invalid params")
	}

	txParams, ok := req.Params[0].(map[string]interface{})
	if !ok {
		return s.writeErrorResponse(req.ID, -32602, "Invalid transaction params")
	}

	// Parse transaction for gas estimation
	tx, err := s.parseTransaction(txParams)
	if err != nil {
		return s.writeErrorResponse(req.ID, -32602, err.Error())
	}

	// Calculate gas estimate based on transaction type and data
	var gasEstimate uint64 = 21000 // Base gas for simple transfers

	// Check if it's a contract creation (no 'to' address)
	if tx.To == nil {
		// Contract creation - higher base cost
		gasEstimate = 53000 // Contract creation base cost
		
		// Add gas for contract code deployment
		if len(tx.Data) > 0 {
			// Gas for code storage: 200 gas per byte
			gasEstimate += uint64(len(tx.Data)) * 200
			
			// Additional gas for contract initialization
			gasEstimate += 32000
		}
	} else {
		// Contract call or transfer
		if len(tx.Data) > 0 {
			// Gas for transaction data: 16 gas per non-zero byte, 4 gas per zero byte
			for _, b := range tx.Data {
				if b == 0 {
					gasEstimate += 4
				} else {
					gasEstimate += 16
				}
			}
			
			// Additional gas for contract execution
			gasEstimate += 2300 // Minimum stipend for contract calls
		}
	}

	// Add a safety margin (25% more gas)
	gasEstimate = gasEstimate * 125 / 100

	// Ensure minimum and maximum limits
	if gasEstimate < 21000 {
		gasEstimate = 21000
	}
	if gasEstimate > 8000000 {
		gasEstimate = 8000000
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  hexutil.EncodeUint64(gasEstimate),
	}
}

// getCode returns the code at an address
func (s *EVMRPCServer) getCode(req *JSONRPCRequest) *JSONRPCResponse {
	if len(req.Params) < 1 {
		return s.writeErrorResponse(req.ID, -32602, "Invalid params")
	}

	addrStr, ok := req.Params[0].(string)
	if !ok {
		return s.writeErrorResponse(req.ID, -32602, "Invalid address")
	}

	addr := common.HexToAddress(addrStr)
	code := s.executor.GetState().GetCode(addr)

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  hexutil.Encode(code),
	}
}

// getTransactionReceipt returns the receipt of a transaction
func (s *EVMRPCServer) getTransactionReceipt(req *JSONRPCRequest) *JSONRPCResponse {
	if len(req.Params) < 1 {
		return s.writeErrorResponse(req.ID, -32602, "Invalid params")
	}

	txHashStr, ok := req.Params[0].(string)
	if !ok {
		return s.writeErrorResponse(req.ID, -32602, "Invalid transaction hash")
	}

	s.mu.RLock()
	receipt, exists := s.txReceipts[txHashStr]
	s.mu.RUnlock()

	if !exists {
		// Return null for pending or non-existent transactions
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  nil,
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  receipt,
	}
}

// blockNumber returns the current block number
func (s *EVMRPCServer) blockNumber(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  hexutil.EncodeBig(s.currentBlockNum),
	}
}

// getBlockByNumber returns block information
func (s *EVMRPCServer) getBlockByNumber(req *JSONRPCRequest) *JSONRPCResponse {
	block := map[string]interface{}{
		"number":           hexutil.EncodeBig(s.currentBlockNum),
		"hash":             "0x" + strings.Repeat("0", 64),
		"parentHash":       "0x" + strings.Repeat("0", 64),
		"timestamp":        hexutil.EncodeUint64(uint64(time.Now().Unix())),
		"gasLimit":         hexutil.EncodeUint64(1000000000),
		"gasUsed":          hexutil.EncodeUint64(0),
		"difficulty":       "0x0",
		"totalDifficulty":  "0x0",
		"transactions":     []interface{}{},
		"size":             hexutil.EncodeUint64(0),
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  block,
	}
}

// gasPrice returns the current gas price
func (s *EVMRPCServer) gasPrice(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  hexutil.EncodeBig(big.NewInt(1000000000)), // 1 gwei
	}
}

// netVersion returns the network version
func (s *EVMRPCServer) netVersion(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  s.chainID.String(),
	}
}

// clientVersion returns the client version
func (s *EVMRPCServer) clientVersion(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  "BlazeDAG/v1.0.0",
	}
}

// parseTransaction parses transaction parameters
func (s *EVMRPCServer) parseTransaction(params map[string]interface{}) (*evm.Transaction, error) {
	fromStr, ok := params["from"].(string)
	if !ok {
		return nil, fmt.Errorf("missing from address")
	}
	from := common.HexToAddress(fromStr)

	var to *common.Address
	if toStr, ok := params["to"].(string); ok && toStr != "" {
		addr := common.HexToAddress(toStr)
		to = &addr
	}

	var value *big.Int = big.NewInt(0)
	if valueStr, ok := params["value"].(string); ok {
		val, err := hexutil.DecodeBig(valueStr)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %v", err)
		}
		value = val
	}

	var gasLimit uint64 = 21000
	if gasStr, ok := params["gas"].(string); ok {
		gas, err := hexutil.DecodeUint64(gasStr)
		if err != nil {
			return nil, fmt.Errorf("invalid gas: %v", err)
		}
		gasLimit = gas
	}

	var gasPrice *big.Int = big.NewInt(1000000000)
	if gasPriceStr, ok := params["gasPrice"].(string); ok {
		price, err := hexutil.DecodeBig(gasPriceStr)
		if err != nil {
			return nil, fmt.Errorf("invalid gas price: %v", err)
		}
		gasPrice = price
	}

	var data []byte
	if dataStr, ok := params["data"].(string); ok {
		d, err := hexutil.Decode(dataStr)
		if err != nil {
			return nil, fmt.Errorf("invalid data: %v", err)
		}
		data = d
	}

	nonce := s.executor.GetState().GetNonce(from)

	return &evm.Transaction{
		From:      from,
		To:        to,
		Value:     value,
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		Nonce:     nonce,
		Data:      data,
		Timestamp: time.Now(),
	}, nil
}

// parseCall parses call parameters
func (s *EVMRPCServer) parseCall(params map[string]interface{}) (*evm.Transaction, error) {
	var from common.Address
	if fromStr, ok := params["from"].(string); ok {
		from = common.HexToAddress(fromStr)
	}

	var to *common.Address
	if toStr, ok := params["to"].(string); ok && toStr != "" {
		addr := common.HexToAddress(toStr)
		to = &addr
	}

	var value *big.Int = big.NewInt(0)
	if valueStr, ok := params["value"].(string); ok {
		val, err := hexutil.DecodeBig(valueStr)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %v", err)
		}
		value = val
	}

	var data []byte
	if dataStr, ok := params["data"].(string); ok {
		d, err := hexutil.Decode(dataStr)
		if err != nil {
			return nil, fmt.Errorf("invalid data: %v", err)
		}
		data = d
	}

	return &evm.Transaction{
		From:      from,
		To:        to,
		Value:     value,
		GasPrice:  big.NewInt(1000000000),
		GasLimit:  1000000,
		Nonce:     0, // For calls, nonce doesn't matter
		Data:      data,
		Timestamp: time.Now(),
	}, nil
}

// writeError writes an error response
func (s *EVMRPCServer) writeError(w http.ResponseWriter, id interface{}, code int, message string) {
	response := s.writeErrorResponse(id, code, message)
	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse creates an error response
func (s *EVMRPCServer) writeErrorResponse(id interface{}, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
}

// UpdateBlockNumber updates the current block number
func (s *EVMRPCServer) UpdateBlockNumber(blockNumber *big.Int) {
	s.currentBlockNum = new(big.Int).Set(blockNumber)
	s.executor.UpdateBlockNumber(blockNumber)
}

// validateTransactionBasic performs basic transaction validation without execution
func (s *EVMRPCServer) validateTransactionBasic(tx *evm.Transaction) error {
	// Check balance for gas + value
	balance := s.executor.GetState().GetBalance(tx.From)
	gasLimit := new(big.Int).SetUint64(tx.GasLimit)
	gasCost := new(big.Int).Mul(gasLimit, tx.GasPrice)
	totalCost := new(big.Int).Add(gasCost, tx.Value)

	if balance.Cmp(totalCost) < 0 {
		return fmt.Errorf("insufficient funds for gas * price + value")
	}

	return nil
}

// ProcessPendingTransactions processes pending transactions and creates receipts
func (s *EVMRPCServer) ProcessPendingTransactions() {
	for {
		select {
		case tx := <-s.pendingTxChan:
			s.processSingleTransaction(tx)
		}
	}
}

// processSingleTransaction processes a single transaction
func (s *EVMRPCServer) processSingleTransaction(tx *evm.Transaction) {
	// Execute transaction
	result := s.executor.ExecuteTransaction(tx)
	
	// Create transaction receipt
	receipt := &TransactionReceipt{
		TransactionHash:   tx.Hash.Hex(),
		TransactionIndex:  "0x0",
		BlockHash:         "0x" + strings.Repeat("0", 64),
		BlockNumber:       hexutil.EncodeBig(s.currentBlockNum),
		From:              tx.From.Hex(),
		CumulativeGasUsed: hexutil.EncodeUint64(result.GasUsed),
		GasUsed:           hexutil.EncodeUint64(result.GasUsed),
		Logs:              make([]interface{}, 0),
		LogsBloom:         "0x" + strings.Repeat("0", 512),
	}

	if tx.To != nil {
		receipt.To = tx.To.Hex()
	}

	if result.Success {
		receipt.Status = "0x1"
		if result.CreatedAddr != nil {
			receipt.ContractAddress = result.CreatedAddr.Hex()
		}
	} else {
		receipt.Status = "0x0"
	}

	// Store receipt
	s.mu.Lock()
	s.txReceipts[tx.Hash.Hex()] = receipt
	// Remove from pending
	delete(s.pendingTxs, tx.Hash.Hex())
	s.mu.Unlock()
}

// GetPendingTransactions returns pending EVM transactions for DAG inclusion
func (s *EVMRPCServer) GetPendingTransactions() []*evm.Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	pending := make([]*evm.Transaction, 0, len(s.pendingTxs))
	for _, tx := range s.pendingTxs {
		pending = append(pending, tx)
	}
	
	return pending
}

// StartTransactionProcessor starts the transaction processor
func (s *EVMRPCServer) StartTransactionProcessor() {
	go s.ProcessPendingTransactions()
} 