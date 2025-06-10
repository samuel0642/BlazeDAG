package evm

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// KeyPair represents an Ethereum key pair
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
	Address    common.Address
}

// GenerateKeyPair generates a new Ethereum key pair
func GenerateKeyPair() (*KeyPair, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	publicKey := &privateKey.PublicKey
	address := crypto.PubkeyToAddress(*publicKey)

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}

// NewKeyPairFromPrivateKey creates a key pair from an existing private key
func NewKeyPairFromPrivateKey(privateKeyHex string) (*KeyPair, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	publicKey := &privateKey.PublicKey
	address := crypto.PubkeyToAddress(*publicKey)

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}

// GetPrivateKeyHex returns the private key as a hex string
func (kp *KeyPair) GetPrivateKeyHex() string {
	return fmt.Sprintf("%x", crypto.FromECDSA(kp.PrivateKey))
}

// GetAddressHex returns the address as a hex string
func (kp *KeyPair) GetAddressHex() string {
	return kp.Address.Hex()
}

// SignTransaction signs a transaction with the private key
func (kp *KeyPair) SignTransaction(tx *Transaction, chainID *big.Int) ([]byte, error) {
	// Create Ethereum transaction
	var ethTx *types.Transaction
	
	if tx.To == nil {
		// Contract creation
		ethTx = types.NewContractCreation(tx.Nonce, tx.Value, tx.GasLimit, tx.GasPrice, tx.Data)
	} else {
		// Regular transaction
		ethTx = types.NewTransaction(tx.Nonce, *tx.To, tx.Value, tx.GasLimit, tx.GasPrice, tx.Data)
	}

	// Sign the transaction
	signer := types.NewEIP155Signer(chainID)
	signedTx, err := types.SignTx(ethTx, signer, kp.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Extract signature
	v, r, s := signedTx.RawSignatureValues()
	
	// Convert to bytes (simplified)
	signature := append(r.Bytes(), s.Bytes()...)
	signature = append(signature, byte(v.Uint64()))
	
	return signature, nil
}

// VerifySignature verifies a transaction signature
func VerifySignature(tx *Transaction, signature []byte, chainID *big.Int) (common.Address, error) {
	// Reconstruct the transaction hash
	var ethTx *types.Transaction
	
	if tx.To == nil {
		ethTx = types.NewContractCreation(tx.Nonce, tx.Value, tx.GasLimit, tx.GasPrice, tx.Data)
	} else {
		ethTx = types.NewTransaction(tx.Nonce, *tx.To, tx.Value, tx.GasLimit, tx.GasPrice, tx.Data)
	}

	// Get the transaction hash
	signer := types.NewEIP155Signer(chainID)
	hash := signer.Hash(ethTx)

	// Recover public key from signature
	pubKey, err := crypto.SigToPub(hash.Bytes(), signature)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to recover public key: %v", err)
	}

	// Get address from public key
	address := crypto.PubkeyToAddress(*pubKey)
	return address, nil
}

// Keystore manages multiple key pairs
type Keystore struct {
	keys map[common.Address]*KeyPair
}

// NewKeystore creates a new keystore
func NewKeystore() *Keystore {
	return &Keystore{
		keys: make(map[common.Address]*KeyPair),
	}
}

// AddKey adds a key pair to the keystore
func (ks *Keystore) AddKey(keyPair *KeyPair) {
	ks.keys[keyPair.Address] = keyPair
}

// GenerateAndAddKey generates a new key pair and adds it to the keystore
func (ks *Keystore) GenerateAndAddKey() (*KeyPair, error) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	
	ks.AddKey(keyPair)
	return keyPair, nil
}

// GetKey retrieves a key pair by address
func (ks *Keystore) GetKey(address common.Address) (*KeyPair, bool) {
	keyPair, exists := ks.keys[address]
	return keyPair, exists
}

// ListAddresses returns all addresses in the keystore
func (ks *Keystore) ListAddresses() []common.Address {
	addresses := make([]common.Address, 0, len(ks.keys))
	for address := range ks.keys {
		addresses = append(addresses, address)
	}
	return addresses
}

// SignTransactionWithAddress signs a transaction using a key from the keystore
func (ks *Keystore) SignTransactionWithAddress(tx *Transaction, address common.Address, chainID *big.Int) ([]byte, error) {
	keyPair, exists := ks.GetKey(address)
	if !exists {
		return nil, fmt.Errorf("key not found for address %s", address.Hex())
	}
	
	return keyPair.SignTransaction(tx, chainID)
}

// CreateTestAccounts creates a set of test accounts with initial balances
func CreateTestAccounts(state *State, count int, initialBalance *big.Int) ([]*KeyPair, error) {
	accounts := make([]*KeyPair, count)
	
	for i := 0; i < count; i++ {
		keyPair, err := GenerateKeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate key pair %d: %v", i, err)
		}
		
		// Set initial balance
		state.SetBalance(keyPair.Address, initialBalance)
		
		accounts[i] = keyPair
	}
	
	return accounts, nil
} 