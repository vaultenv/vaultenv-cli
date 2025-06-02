package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileBackend implements persistent file-based storage
type FileBackend struct {
	mu       sync.RWMutex
	basePath string
	env      string
}

// NewFileBackend creates a new file-based storage backend
func NewFileBackend(basePath, environment string) (*FileBackend, error) {
	// Ensure the base path exists
	dataPath := filepath.Join(basePath, "data")
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &FileBackend{
		basePath: basePath,
		env:      environment,
	}, nil
}

// getDataFile returns the path to the data file for this environment
func (f *FileBackend) getDataFile() string {
	return filepath.Join(f.basePath, "data", f.env+".json")
}

// loadData loads the data from disk
func (f *FileBackend) loadData() (map[string]string, error) {
	dataFile := f.getDataFile()

	// If file doesn't exist, return empty map
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		return make(map[string]string), nil
	}

	// Read the file
	data, err := os.ReadFile(dataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read data file: %w", err)
	}

	// If file is empty, return empty map
	if len(data) == 0 {
		return make(map[string]string), nil
	}

	// Unmarshal the data
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return result, nil
}

// saveData saves the data to disk
func (f *FileBackend) saveData(data map[string]string) error {
	// Marshal the data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write to a temporary file first
	dataFile := f.getDataFile()
	tempFile := dataFile + ".tmp"

	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename temporary file to actual file (atomic operation)
	if err := os.Rename(tempFile, dataFile); err != nil {
		// Clean up temp file if rename fails
		os.Remove(tempFile)
		return fmt.Errorf("failed to save data file: %w", err)
	}

	return nil
}

// Set stores a variable
func (f *FileBackend) Set(key, value string, encrypt bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Load current data
	data, err := f.loadData()
	if err != nil {
		return err
	}

	// TODO: Implement encryption when encrypt is true
	data[key] = value

	// Save data back to disk
	return f.saveData(data)
}

// Get retrieves a variable
func (f *FileBackend) Get(key string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Load current data
	data, err := f.loadData()
	if err != nil {
		return "", err
	}

	value, exists := data[key]
	if !exists {
		return "", ErrNotFound
	}

	return value, nil
}

// Exists checks if a variable exists
func (f *FileBackend) Exists(key string) (bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Load current data
	data, err := f.loadData()
	if err != nil {
		return false, err
	}

	_, exists := data[key]
	return exists, nil
}

// Delete removes a variable
func (f *FileBackend) Delete(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Load current data
	data, err := f.loadData()
	if err != nil {
		return err
	}

	// Delete the key
	delete(data, key)

	// Save data back to disk
	return f.saveData(data)
}

// List returns all variable names
func (f *FileBackend) List() ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Load current data
	data, err := f.loadData()
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}

	return keys, nil
}

// Close closes the backend (no-op for file backend)
func (f *FileBackend) Close() error {
	return nil
}
