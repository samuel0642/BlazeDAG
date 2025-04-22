package evm

import (
	"crypto/sha256"
	"errors"
	"sync"
)

// Errors
var (
	ErrAccountNotFound = errors.New("account not found")
)

// StorageManager handles storage management
type StorageManager struct {
	storage map[string]map[string][]byte
	mu      sync.RWMutex
}

// NewStorageManager creates a new storage manager
func NewStorageManager() *StorageManager {
	return &StorageManager{
		storage: make(map[string]map[string][]byte),
	}
}

// GetValue gets a value from storage
func (sm *StorageManager) GetValue(address string, key string) ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	accountStorage, exists := sm.storage[address]
	if !exists {
		return nil, ErrAccountNotFound
	}

	value, exists := accountStorage[key]
	if !exists {
		return nil, nil
	}

	return value, nil
}

// SetValue sets a value in storage
func (sm *StorageManager) SetValue(address string, key string, value []byte) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	accountStorage, exists := sm.storage[address]
	if !exists {
		accountStorage = make(map[string][]byte)
		sm.storage[address] = accountStorage
	}

	accountStorage[key] = value
	return nil
}

// DeleteValue deletes a value from storage
func (sm *StorageManager) DeleteValue(address string, key string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	accountStorage, exists := sm.storage[address]
	if !exists {
		return ErrAccountNotFound
	}

	delete(accountStorage, key)
	return nil
}

// GetStorageRoot gets the storage root for an account
func (sm *StorageManager) GetStorageRoot(address string) ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	accountStorage, exists := sm.storage[address]
	if !exists {
		return nil, ErrAccountNotFound
	}

	return calculateStorageRoot(accountStorage), nil
}

// GetStorageProof gets a storage proof for a value
func (sm *StorageManager) GetStorageProof(address string, key string) (*StorageProof, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	accountStorage, exists := sm.storage[address]
	if !exists {
		return nil, ErrAccountNotFound
	}

	value, exists := accountStorage[key]
	if !exists {
		return nil, nil
	}

	// Create proof
	proof := &StorageProof{
		Address: address,
		Key:     key,
		Value:   value,
		Proof:   make([][]byte, 0),
	}

	// Add proof elements
	for k, v := range accountStorage {
		if k != key {
			proof.Proof = append(proof.Proof, []byte(k))
			proof.Proof = append(proof.Proof, v)
		}
	}

	return proof, nil
}

// VerifyStorageProof verifies a storage proof
func (sm *StorageManager) VerifyStorageProof(proof *StorageProof) bool {
	// Calculate root from proof
	calculatedRoot := calculateStorageRootFromProof(proof)
	
	// Get actual root
	actualRoot, err := sm.GetStorageRoot(proof.Address)
	if err != nil {
		return false
	}

	// Compare roots
	return string(calculatedRoot) == string(actualRoot)
}

// StorageProof represents a storage proof
type StorageProof struct {
	Address string
	Key     string
	Value   []byte
	Proof   [][]byte
}

// calculateStorageRoot calculates the root of the storage trie
func calculateStorageRoot(storage map[string][]byte) []byte {
	h := sha256.New()
	
	// Sort keys for deterministic hashing
	keys := make([]string, 0, len(storage))
	for key := range storage {
		keys = append(keys, key)
	}
	
	// Hash each key-value pair
	for _, key := range keys {
		value := storage[key]
		h.Write([]byte(key))
		h.Write(value)
	}
	
	return h.Sum(nil)
}

// calculateStorageRootFromProof calculates the root from a storage proof
func calculateStorageRootFromProof(proof *StorageProof) []byte {
	h := sha256.New()
	
	// Hash the key-value pair being proven
	h.Write([]byte(proof.Key))
	h.Write(proof.Value)
	
	// Hash the proof elements
	for _, element := range proof.Proof {
		h.Write(element)
	}
	
	return h.Sum(nil)
}

// GetStorageSize gets the size of storage for an account
func (sm *StorageManager) GetStorageSize(address string) (int, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	accountStorage, exists := sm.storage[address]
	if !exists {
		return 0, ErrAccountNotFound
	}

	return len(accountStorage), nil
}

// ClearStorage clears storage for an account
func (sm *StorageManager) ClearStorage(address string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.storage[address]; !exists {
		return ErrAccountNotFound
	}

	sm.storage[address] = make(map[string][]byte)
	return nil
}

// Storage represents the storage of an account
type Storage struct {
	values map[string][]byte
}

// NewStorage creates a new storage
func NewStorage() *Storage {
	return &Storage{
		values: make(map[string][]byte),
	}
}

// Get gets a value from storage
func (s *Storage) Get(key string) ([]byte, error) {
	value, exists := s.values[key]
	if !exists {
		return nil, ErrAccountNotFound
	}
	return value, nil
}

// Set sets a value in storage
func (s *Storage) Set(key string, value []byte) error {
	s.values[key] = value
	return nil
}

// Delete deletes a value from storage
func (s *Storage) Delete(key string) error {
	delete(s.values, key)
	return nil
}

// Has checks if a key exists in storage
func (s *Storage) Has(key string) bool {
	_, exists := s.values[key]
	return exists
}

// Clear clears all values from storage
func (s *Storage) Clear() {
	s.values = make(map[string][]byte)
}

// GetKeys returns all keys in storage
func (s *Storage) GetKeys() []string {
	keys := make([]string, 0, len(s.values))
	for key := range s.values {
		keys = append(keys, key)
	}
	return keys
}

// GetValues returns all values in storage
func (s *Storage) GetValues() [][]byte {
	values := make([][]byte, 0, len(s.values))
	for _, value := range s.values {
		values = append(values, value)
	}
	return values
} 