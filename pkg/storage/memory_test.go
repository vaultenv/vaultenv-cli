package storage

import (
	"fmt"
	"sort"
	"sync"
	"testing"
)

func TestMemoryBackend_Set(t *testing.T) {
	backend := NewMemoryBackend()

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
		{"encrypted", "ENCRYPTED", "secret", true, false},
		{"multiline", "MULTILINE", "line1\nline2\nline3", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backend.Set(tt.key, tt.value, tt.encrypt)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryBackend_Get(t *testing.T) {
	backend := NewMemoryBackend()

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
			if err == ErrNotFound && !tt.wantErr {
				t.Error("Get() returned ErrNotFound for existing key")
			}
		})
	}
}

func TestMemoryBackend_Exists(t *testing.T) {
	backend := NewMemoryBackend()

	// Set test data
	backend.Set("EXISTING", "value", false)

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"existing", "EXISTING", true},
		{"not_found", "NOTFOUND", false},
		{"empty_key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := backend.Exists(tt.key)
			if err != nil {
				t.Errorf("Exists() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryBackend_Delete(t *testing.T) {
	backend := NewMemoryBackend()

	// Set test data
	backend.Set("TO_DELETE", "value", false)
	backend.Set("TO_KEEP", "value", false)

	// Delete existing key
	err := backend.Delete("TO_DELETE")
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

	// Delete non-existing key (should not error)
	err = backend.Delete("NOTFOUND")
	if err != nil {
		t.Errorf("Delete() error for non-existing key = %v", err)
	}
}

func TestMemoryBackend_List(t *testing.T) {
	backend := NewMemoryBackend()

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

func TestMemoryBackend_Close(t *testing.T) {
	backend := NewMemoryBackend()

	// Close should always succeed
	err := backend.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Should still work after close (for memory backend)
	err = backend.Set("KEY", "value", false)
	if err != nil {
		t.Errorf("Set() after Close() error = %v", err)
	}
}

func TestMemoryBackend_Overwrite(t *testing.T) {
	backend := NewMemoryBackend()

	// Set initial value
	backend.Set("KEY", "initial", false)

	// Overwrite
	err := backend.Set("KEY", "updated", false)
	if err != nil {
		t.Errorf("Set() overwrite error = %v", err)
	}

	// Verify new value
	value, _ := backend.Get("KEY")
	if value != "updated" {
		t.Errorf("Get() after overwrite = %v, want updated", value)
	}
}

func TestMemoryBackend_Concurrent(t *testing.T) {
	backend := NewMemoryBackend()

	// Test concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 100

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

	// Test concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("KEY_%d", n)
			expectedValue := fmt.Sprintf("value_%d", n)

			value, err := backend.Get(key)
			if err != nil {
				t.Errorf("Concurrent Get() error = %v", err)
				return
			}
			if value != expectedValue {
				t.Errorf("Concurrent Get() = %v, want %v", value, expectedValue)
			}
		}(i)
	}

	wg.Wait()
}

func TestMemoryBackend_MixedOperations(t *testing.T) {
	backend := NewMemoryBackend()

	// Perform mixed operations
	var wg sync.WaitGroup

	// Writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			backend.Set(fmt.Sprintf("WRITE_%d", i), "value", false)
		}
	}()

	// Reader
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			backend.List()
		}
	}()

	// Deleter
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 25; i++ {
			backend.Delete(fmt.Sprintf("WRITE_%d", i))
		}
	}()

	wg.Wait()

	// Verify state is consistent
	keys, _ := backend.List()
	if len(keys) < 25 {
		t.Errorf("Expected at least 25 keys remaining, got %d", len(keys))
	}
}

func BenchmarkMemoryBackend_Set(b *testing.B) {
	backend := NewMemoryBackend()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("KEY_%d", i)
		backend.Set(key, "value", false)
	}
}

func BenchmarkMemoryBackend_Get(b *testing.B) {
	backend := NewMemoryBackend()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		backend.Set(fmt.Sprintf("KEY_%d", i), "value", false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("KEY_%d", i%1000)
		backend.Get(key)
	}
}

func BenchmarkMemoryBackend_List(b *testing.B) {
	backend := NewMemoryBackend()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		backend.Set(fmt.Sprintf("KEY_%d", i), "value", false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend.List()
	}
}
