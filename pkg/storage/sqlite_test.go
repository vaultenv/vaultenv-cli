package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestSQLiteBackend_NewSQLiteBackend(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
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
			backend, err := NewSQLiteBackend(tt.basePath, tt.environment)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSQLiteBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err == nil {
				// Verify database file was created
				dbPath := filepath.Join(tt.basePath, "vaultenv.db")
				if _, err := os.Stat(dbPath); os.IsNotExist(err) {
					t.Error("NewSQLiteBackend() did not create database file")
				}
				
				// Verify tables were created
				var count int
				err := backend.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
				if err != nil {
					t.Errorf("Failed to query tables: %v", err)
				}
				if count < 3 { // secrets, history, audit_log
					t.Errorf("Expected at least 3 tables, got %d", count)
				}
				
				backend.Close()
			}
		})
	}
}

func TestSQLiteBackend_SetGet(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend.Close()
	
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
		{"special_chars", "SPECIAL", "!@#$%^&*()_+-=[]{}|;':\",./<>?", false, false},
		{"multiline", "MULTILINE", "line1\nline2\nline3", false, false},
		{"json_chars", "JSON", `{"key": "value"}`, false, false},
		{"sql_injection", "KEY'); DROP TABLE secrets; --", "value", false, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backend.Set(tt.key, tt.value, tt.encrypt)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if !tt.wantErr {
				// Verify value was stored
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

func TestSQLiteBackend_Get(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend.Close()
	
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
		{"sql_chars", "KEY'; --", "", true},
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

func TestSQLiteBackend_Delete(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend.Close()
	
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
	
	// Delete non-existing key (should not error)
	err = backend.Delete("NOTFOUND")
	if err != nil {
		t.Errorf("Delete() error for non-existing key = %v", err)
	}
}

func TestSQLiteBackend_List(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend.Close()
	
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

func TestSQLiteBackend_History(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend.Close()
	
	// Set initial value
	backend.Set("HISTORY_KEY", "version1", false)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	
	// Update value
	backend.Set("HISTORY_KEY", "version2", false)
	time.Sleep(10 * time.Millisecond)
	
	// Update again
	backend.Set("HISTORY_KEY", "version3", false)
	
	// Get history
	history, err := backend.GetHistory("HISTORY_KEY", 10)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	
	// Should have 3 versions
	if len(history) != 3 {
		t.Errorf("GetHistory() returned %d entries, want 3", len(history))
	}
	
	// Verify versions are in reverse chronological order
	expectedValues := []string{"version3", "version2", "version1"}
	for i, h := range history {
		if h.Value != expectedValues[i] {
			t.Errorf("History[%d].Value = %v, want %v", i, h.Value, expectedValues[i])
		}
		if h.Version != len(history)-i {
			t.Errorf("History[%d].Version = %v, want %v", i, h.Version, len(history)-i)
		}
	}
	
	// Test with limit
	limitedHistory, err := backend.GetHistory("HISTORY_KEY", 2)
	if err != nil {
		t.Fatalf("GetHistory() with limit error = %v", err)
	}
	
	if len(limitedHistory) != 2 {
		t.Errorf("GetHistory() with limit returned %d entries, want 2", len(limitedHistory))
	}
	
	// Test non-existent key
	emptyHistory, err := backend.GetHistory("NONEXISTENT", 10)
	if err != nil {
		t.Fatalf("GetHistory() for non-existent key error = %v", err)
	}
	
	if len(emptyHistory) != 0 {
		t.Errorf("GetHistory() for non-existent key returned %d entries, want 0", len(emptyHistory))
	}
}

func TestSQLiteBackend_DeleteHistory(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend.Close()
	
	// Set and update value
	backend.Set("DELETE_TEST", "value1", false)
	backend.Set("DELETE_TEST", "value2", false)
	
	// Delete the key
	backend.Delete("DELETE_TEST")
	
	// History should show deletion
	history, _ := backend.GetHistory("DELETE_TEST", 10)
	
	if len(history) < 1 {
		t.Fatal("Expected at least one history entry after deletion")
	}
	
	// Latest entry should be deletion
	if history[0].ChangeType != "DELETE" {
		t.Errorf("Latest history ChangeType = %v, want DELETE", history[0].ChangeType)
	}
}

func TestSQLiteBackend_AuditLog(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend.Close()
	
	// Perform various operations
	backend.Set("AUDIT_KEY1", "value1", false)
	backend.Get("AUDIT_KEY1")
	backend.Get("NONEXISTENT") // This should fail
	backend.Delete("AUDIT_KEY1")
	backend.List()
	
	// Small delay to allow async audit logging to complete
	time.Sleep(100 * time.Millisecond)
	
	// Get audit log
	entries, err := backend.GetAuditLog(10)
	if err != nil {
		t.Fatalf("GetAuditLog() error = %v", err)
	}
	
	// Should have at least 5 entries
	if len(entries) < 5 {
		t.Errorf("GetAuditLog() returned %d entries, want at least 5", len(entries))
	}
	
	// Verify different action types
	actionTypes := make(map[string]bool)
	for _, entry := range entries {
		actionTypes[entry.Action] = true
	}
	
	expectedActions := []string{"SET", "GET", "DELETE", "LIST"}
	for _, action := range expectedActions {
		if !actionTypes[action] {
			t.Errorf("Missing action type %s in audit log", action)
		}
	}
	
	// Verify failed operation is logged
	foundFailure := false
	for _, entry := range entries {
		if !entry.Success {
			foundFailure = true
			break
		}
	}
	
	if !foundFailure {
		t.Error("Expected to find failed operation in audit log")
	}
}

func TestSQLiteBackend_MultipleEnvironments(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create backends for different environments
	devBackend, _ := NewSQLiteBackend(tmpDir, "dev")
	prodBackend, _ := NewSQLiteBackend(tmpDir, "prod")
	defer devBackend.Close()
	defer prodBackend.Close()
	
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
	
	// Verify separate lists
	devBackend.Set("DEV_ONLY", "value", false)
	prodBackend.Set("PROD_ONLY", "value", false)
	
	devKeys, _ := devBackend.List()
	_, _ = prodBackend.List()
	
	// Check dev keys
	hasDevOnly := false
	hasProdOnly := false
	for _, key := range devKeys {
		if key == "DEV_ONLY" {
			hasDevOnly = true
		}
		if key == "PROD_ONLY" {
			hasProdOnly = true
		}
	}
	
	if !hasDevOnly {
		t.Error("Dev environment missing DEV_ONLY key")
	}
	if hasProdOnly {
		t.Error("Dev environment has PROD_ONLY key")
	}
}

func TestSQLiteBackend_Concurrent(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend.Close()
	
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
	
	// Concurrent reads
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

func TestSQLiteBackend_Transaction(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend.Close()
	
	// Test that operations are atomic
	// Set initial value
	backend.Set("TX_KEY", "initial", false)
	
	// Simulate concurrent update
	var wg sync.WaitGroup
	results := make([]string, 10)
	
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			
			// Read current value
			val, _ := backend.Get("TX_KEY")
			
			// Update based on current value
			newVal := fmt.Sprintf("%s_%d", val, n)
			backend.Set("TX_KEY", newVal, false)
			
			// Read back
			results[n], _ = backend.Get("TX_KEY")
		}(i)
	}
	
	wg.Wait()
	
	// Final value should be one of the updates
	finalValue, _ := backend.Get("TX_KEY")
	found := false
	for _, result := range results {
		if result == finalValue {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Final value doesn't match any of the concurrent updates")
	}
}

func TestSQLiteBackend_Persistence(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create backend and set data
	backend1, _ := NewSQLiteBackend(tmpDir, "test")
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
	backend2, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend2.Close()
	
	for key, expectedValue := range testData {
		value, err := backend2.Get(key)
		if err != nil {
			t.Errorf("Get(%s) error = %v", key, err)
		}
		if value != expectedValue {
			t.Errorf("Get(%s) = %v, want %v", key, value, expectedValue)
		}
	}
	
	// Verify history persists
	history, _ := backend2.GetHistory("KEY1", 10)
	if len(history) == 0 {
		t.Error("History not persisted")
	}
}

func TestSQLiteBackend_LargeValues(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend.Close()
	
	// Test with large values
	largeValue := string(make([]byte, 1024*1024)) // 1MB
	for i := range largeValue {
		largeValue = string(rune('A' + (i % 26)))
	}
	
	err = backend.Set("LARGE_KEY", largeValue, false)
	if err != nil {
		t.Fatalf("Set() with large value error = %v", err)
	}
	
	retrieved, err := backend.Get("LARGE_KEY")
	if err != nil {
		t.Fatalf("Get() with large value error = %v", err)
	}
	
	if len(retrieved) != len(largeValue) {
		t.Errorf("Retrieved value length = %d, want %d", len(retrieved), len(largeValue))
	}
}

func TestSQLiteBackend_WALMode(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sqlite_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "test")
	defer backend.Close()
	
	// Verify WAL mode is enabled
	var mode string
	err = backend.db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("Failed to query journal mode: %v", err)
	}
	
	if mode != "wal" {
		t.Errorf("Journal mode = %v, want wal", mode)
	}
}

func BenchmarkSQLiteBackend_Set(b *testing.B) {
	tmpDir, err := ioutil.TempDir("", "sqlite_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "bench")
	defer backend.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("KEY_%d", i)
		backend.Set(key, "value", false)
	}
}

func BenchmarkSQLiteBackend_Get(b *testing.B) {
	tmpDir, err := ioutil.TempDir("", "sqlite_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "bench")
	defer backend.Close()
	
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

func BenchmarkSQLiteBackend_List(b *testing.B) {
	tmpDir, err := ioutil.TempDir("", "sqlite_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	backend, _ := NewSQLiteBackend(tmpDir, "bench")
	defer backend.Close()
	
	// Pre-populate
	for i := 0; i < 1000; i++ {
		backend.Set(fmt.Sprintf("KEY_%d", i), "value", false)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend.List()
	}
}