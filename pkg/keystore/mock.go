package keystore

import (
	"fmt"
)

// MockKeystore is an in-memory implementation for testing
type MockKeystore struct {
	keys map[string][]byte
}

func NewMockKeystore() *MockKeystore {
	return &MockKeystore{
		keys: make(map[string][]byte),
	}
}

func (m *MockKeystore) Store(service, account string, data []byte) error {
	if m.keys == nil {
		m.keys = make(map[string][]byte)
	}
	if len(data) == 0 {
		return fmt.Errorf("cannot store empty data")
	}
	key := fmt.Sprintf("%s:%s", service, account)
	// Store a copy to prevent external modifications
	m.keys[key] = append([]byte(nil), data...)
	return nil
}

func (m *MockKeystore) Retrieve(service, account string) ([]byte, error) {
	if m.keys == nil {
		m.keys = make(map[string][]byte)
	}
	key := fmt.Sprintf("%s:%s", service, account)
	data, exists := m.keys[key]
	if !exists {
		return nil, fmt.Errorf("key not found for %s", account)
	}
	// Return a copy to prevent external modifications
	return append([]byte(nil), data...), nil
}

func (m *MockKeystore) Delete(service, account string) error {
	if m.keys == nil {
		m.keys = make(map[string][]byte)
	}
	key := fmt.Sprintf("%s:%s", service, account)
	delete(m.keys, key)
	return nil
}

func (m *MockKeystore) List(service string) ([]string, error) {
	if m.keys == nil {
		m.keys = make(map[string][]byte)
	}
	var accounts []string
	prefix := service + ":"
	
	for key := range m.keys {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			account := key[len(prefix):]
			accounts = append(accounts, account)
		}
	}
	
	return accounts, nil
}