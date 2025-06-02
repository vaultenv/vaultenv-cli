package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

func TestFileBackend_NewFileBackend(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "filebackend_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		basePath    string
		environment string
		wantErr     bool
	}{
		{"valid", tmpDir, "test", false},
		{"with_subdirs", filepath.Join(tmpDir, "sub", "dir"), "prod", false},
		{"empty_env", tmpDir, "", false},
		{"special_chars_env", tmpDir, "test-env_123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := NewFileBackend(tt.basePath, tt.environment)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				// Verify data directory was created
				dataPath := filepath.Join(tt.basePath, "data")
				if _, err := os.Stat(dataPath); os.IsNotExist(err) {
					t.Error("NewFileBackend() did not create data directory")
				}
				backend.Close()
			}
		})
	}
}

func TestFileBackend_Set(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "filebackend_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	backend, _ := NewFileBackend(tmpDir, "test")

	tests := []struct {
		name    string
		key     string
		value   string
		encrypt bool
		wantErr bool
	}{
		{"simple", "KEY1", "value1", false, false},
		{"with_spaces", "KEY_WITH_SPACES", "value with spaces", false, false},
		{"empty_value", "EMPTY", "", false, false},
		{"unicode", "UNICODE", "Hello 世界 🌍", false, false},
		{"special_chars", "SPECIAL", "!@#$%^&*()", false, false},
		{"multiline", "MULTILINE", "line1\nline2\nline3", false, false},
		{"json_chars", "JSON", `{"key": "value"}`, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backend.Set(tt.key, tt.value, tt.encrypt)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify value was persisted
			if !tt.wantErr {
				value, err := backend.Get(tt.key)
				if err != nil {
					t.Errorf("Get() after Set() error = %v", err)
				}
				if value != tt.value {
					t.Errorf("Get() = %v, want %v", value, tt.value)
				}
			}
		})
	}
}

func TestFileBackend_Get(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "filebackend_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	backend, _ := NewFileBackend(tmpDir, "test")

	// Set some test data
	backend.Set("EXISTING", "value", false)
	backend.Set("EMPTY", "", false)

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr bool
	}{
		{"existing", "EXISTING", "value", false},
		{"empty_value", "EMPTY", "", false},
		{"not_found", "NOTFOUND", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := backend.Get(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
			if tt.wantErr && err != ErrNotFound {
				t.Errorf("Get() error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestFileBackend_Delete(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "filebackend_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	backend, _ := NewFileBackend(tmpDir, "test")

	// Set test data
	backend.Set("TO_DELETE", "value", false)
	backend.Set("TO_KEEP", "value", false)

	// Delete existing key
	err = backend.Delete("TO_DELETE")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify deletion
	exists, _ := backend.Exists("TO_DELETE")
	if exists {
		t.Error("Delete() did not remove the key")
	}

	// Verify other keys remain
	exists, _ = backend.Exists("TO_KEEP")
	if !exists {
		t.Error("Delete() removed wrong key")
	}

	// Verify persistence
	backend2, _ := NewFileBackend(tmpDir, "test")
	exists, _ = backend2.Exists("TO_DELETE")
	if exists {
		t.Error("Deletion was not persisted")
	}
}

func TestFileBackend_List(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "filebackend_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	backend, _ := NewFileBackend(tmpDir, "test")

	// Test empty list
	keys, err := backend.List()
	if err != nil {
		t.Errorf("List() error = %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("List() = %v, want empty list", keys)
	}

	// Add test data
	testKeys := []string{"KEY1", "KEY2", "KEY3", "ANOTHER_KEY"}
	for _, key := range testKeys {
		backend.Set(key, "value", false)
	}

	// Get list
	keys, err = backend.List()
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	// Sort for comparison
	sort.Strings(keys)
	sort.Strings(testKeys)

	if len(keys) != len(testKeys) {
		t.Errorf("List() returned %d keys, want %d", len(keys), len(testKeys))
	}

	for i, key := range keys {
		if key != testKeys[i] {
			t.Errorf("List()[%d] = %v, want %v", i, key, testKeys[i])
		}
	}
}

func TestFileBackend_Persistence(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "filebackend_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend and set data
	backend1, _ := NewFileBackend(tmpDir, "test")
	testData := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}

	for key, value := range testData {
		backend1.Set(key, value, false)
	}
	backend1.Close()

	// Create new backend and verify data persists
	backend2, _ := NewFileBackend(tmpDir, "test")
	for key, expectedValue := range testData {
		value, err := backend2.Get(key)
		if err != nil {
			t.Errorf("Get(%s) error = %v", key, err)
		}
		if value != expectedValue {
			t.Errorf("Get(%s) = %v, want %v", key, value, expectedValue)
		}
	}
}

func TestFileBackend_MultipleEnvironments(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "filebackend_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backends for different environments
	devBackend, _ := NewFileBackend(tmpDir, "dev")
	prodBackend, _ := NewFileBackend(tmpDir, "prod")

	// Set different data in each environment
	devBackend.Set("KEY", "dev-value", false)
	prodBackend.Set("KEY", "prod-value", false)

	// Verify isolation
	devValue, _ := devBackend.Get("KEY")
	prodValue, _ := prodBackend.Get("KEY")

	if devValue != "dev-value" {
		t.Errorf("Dev Get() = %v, want dev-value", devValue)
	}
	if prodValue != "prod-value" {
		t.Errorf("Prod Get() = %v, want prod-value", prodValue)
	}
}

func TestFileBackend_FileFormat(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "filebackend_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	backend, _ := NewFileBackend(tmpDir, "test")

	// Set test data
	testData := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}

	for key, value := range testData {
		backend.Set(key, value, false)
	}

	// Read file directly
	dataFile := filepath.Join(tmpDir, "data", "test.json")
	data, err := ioutil.ReadFile(dataFile)
	if err != nil {
		t.Fatalf("Failed to read data file: %v", err)
	}

	// Verify JSON format
	var fileData map[string]string
	if err := json.Unmarshal(data, &fileData); err != nil {
		t.Fatalf("Invalid JSON format: %v", err)
	}

	// Verify content
	for key, expectedValue := range testData {
		if value, ok := fileData[key]; !ok || value != expectedValue {
			t.Errorf("File data[%s] = %v, want %v", key, value, expectedValue)
		}
	}
}

func TestFileBackend_CorruptedFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "filebackend_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend and data directory
	backend, _ := NewFileBackend(tmpDir, "test")
	backend.Set("KEY", "value", false) // Ensure file exists

	// Corrupt the file
	dataFile := filepath.Join(tmpDir, "data", "test.json")
	ioutil.WriteFile(dataFile, []byte("invalid json"), 0644)

	// Try to read - should get error
	_, err = backend.Get("KEY")
	if err == nil {
		t.Error("Expected error for corrupted file")
	}
}

func TestFileBackend_EmptyFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "filebackend_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create backend and empty file
	backend, _ := NewFileBackend(tmpDir, "test")
	dataFile := filepath.Join(tmpDir, "data", "test.json")
	os.MkdirAll(filepath.Dir(dataFile), 0755)
	ioutil.WriteFile(dataFile, []byte{}, 0644)

	// Should handle empty file gracefully
	keys, err := backend.List()
	if err != nil {
		t.Errorf("List() with empty file error = %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("List() with empty file = %v, want empty", keys)
	}
}

func TestFileBackend_Concurrent(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "filebackend_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	backend, _ := NewFileBackend(tmpDir, "test")

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("KEY_%d", n)
			value := fmt.Sprintf("value_%d", n)

			if err := backend.Set(key, value, false); err != nil {
				t.Errorf("Concurrent Set() error = %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all keys were set
	keys, _ := backend.List()
	if len(keys) != numGoroutines {
		t.Errorf("Expected %d keys, got %d", numGoroutines, len(keys))
	}
}

func BenchmarkFileBackend_Set(b *testing.B) {
	tmpDir, err := ioutil.TempDir("", "filebackend_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	backend, _ := NewFileBackend(tmpDir, "bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("KEY_%d", i)
		backend.Set(key, "value", false)
	}
}

func BenchmarkFileBackend_Get(b *testing.B) {
	tmpDir, err := ioutil.TempDir("", "filebackend_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	backend, _ := NewFileBackend(tmpDir, "bench")

	// Pre-populate
	for i := 0; i < 100; i++ {
		backend.Set(fmt.Sprintf("KEY_%d", i), "value", false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("KEY_%d", i%100)
		backend.Get(key)
	}
}
