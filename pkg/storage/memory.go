package storage

import (
	"sync"
)

// MemoryBackend is a simple in-memory storage backend for development
type MemoryBackend struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewMemoryBackend creates a new in-memory storage backend
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		data: make(map[string]string),
	}
}

// Set stores a variable
func (m *MemoryBackend) Set(key, value string, encrypt bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// TODO: Implement encryption when encrypt is true
	m.data[key] = value
	return nil
}

// Get retrieves a variable
func (m *MemoryBackend) Get(key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.data[key]
	if !exists {
		return "", ErrNotFound
	}

	return value, nil
}

// Exists checks if a variable exists
func (m *MemoryBackend) Exists(key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.data[key]
	return exists, nil
}

// Delete removes a variable
func (m *MemoryBackend) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}

// List returns all variable names
func (m *MemoryBackend) List() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}

	return keys, nil
}

// Close closes the backend (no-op for memory backend)
func (m *MemoryBackend) Close() error {
	return nil
}
