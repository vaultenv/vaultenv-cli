package keystore

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewKeystore(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "keystore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Test successful creation
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		t.Fatalf("NewKeystore() error = %v", err)
	}
	defer ks.Close()
	
	// Verify database file was created
	dbPath := filepath.Join(tmpDir, keystoreDBName)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
	
	// Verify tables were created
	var count int
	err = ks.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query tables: %v", err)
	}
	
	// Should have schema_version, keys, and environment_keys tables
	if count < 3 {
		t.Errorf("Expected at least 3 tables, got %d", count)
	}
}

func TestKeystore_StoreAndGetKey(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "keystore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ks.Close()
	
	// Test data
	projectID := "test-project"
	entry := &KeyEntry{
		ProjectID:        projectID,
		Salt:             []byte("test-salt-32-bytes-for-testing!!"),
		VerificationHash: "test-hash",
		CreatedAt:        time.Now(),
	}
	
	// Store key
	err = ks.StoreKey(projectID, entry)
	if err != nil {
		t.Fatalf("StoreKey() error = %v", err)
	}
	
	// Retrieve key
	retrieved, err := ks.GetKey(projectID)
	if err != nil {
		t.Fatalf("GetKey() error = %v", err)
	}
	
	// Verify retrieved data
	if retrieved.ProjectID != entry.ProjectID {
		t.Errorf("ProjectID = %v, want %v", retrieved.ProjectID, entry.ProjectID)
	}
	
	if string(retrieved.Salt) != string(entry.Salt) {
		t.Errorf("Salt = %v, want %v", retrieved.Salt, entry.Salt)
	}
	
	if retrieved.VerificationHash != entry.VerificationHash {
		t.Errorf("VerificationHash = %v, want %v", retrieved.VerificationHash, entry.VerificationHash)
	}
	
	// Test update
	entry.VerificationHash = "updated-hash"
	err = ks.StoreKey(projectID, entry)
	if err != nil {
		t.Fatalf("StoreKey() update error = %v", err)
	}
	
	retrieved, err = ks.GetKey(projectID)
	if err != nil {
		t.Fatalf("GetKey() after update error = %v", err)
	}
	
	if retrieved.VerificationHash != "updated-hash" {
		t.Errorf("Updated VerificationHash = %v, want 'updated-hash'", retrieved.VerificationHash)
	}
}

func TestKeystore_GetKey_NotFound(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "keystore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ks.Close()
	
	// Try to get non-existent key
	_, err = ks.GetKey("non-existent")
	if err != ErrKeyNotFound {
		t.Errorf("GetKey() error = %v, want %v", err, ErrKeyNotFound)
	}
}

func TestKeystore_DeleteKey(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "keystore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ks.Close()
	
	// Store a key
	projectID := "delete-test"
	entry := &KeyEntry{
		ProjectID:        projectID,
		Salt:             []byte("salt"),
		VerificationHash: "hash",
		CreatedAt:        time.Now(),
	}
	
	ks.StoreKey(projectID, entry)
	
	// Delete key
	err = ks.DeleteKey(projectID)
	if err != nil {
		t.Fatalf("DeleteKey() error = %v", err)
	}
	
	// Verify deletion
	_, err = ks.GetKey(projectID)
	if err != ErrKeyNotFound {
		t.Errorf("GetKey() after delete error = %v, want %v", err, ErrKeyNotFound)
	}
	
	// Try to delete non-existent key
	err = ks.DeleteKey("non-existent")
	if err != ErrKeyNotFound {
		t.Errorf("DeleteKey() non-existent error = %v, want %v", err, ErrKeyNotFound)
	}
}

func TestKeystore_ListProjects(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "keystore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ks.Close()
	
	// Test empty list
	projects, err := ks.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	
	if len(projects) != 0 {
		t.Errorf("ListProjects() = %v, want empty", projects)
	}
	
	// Add some projects
	projectIDs := []string{"project1", "project2", "project3"}
	for _, id := range projectIDs {
		entry := &KeyEntry{
			ProjectID:        id,
			Salt:             []byte("salt"),
			VerificationHash: "hash",
			CreatedAt:        time.Now(),
		}
		ks.StoreKey(id, entry)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}
	
	// List projects
	projects, err = ks.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	
	if len(projects) != len(projectIDs) {
		t.Errorf("ListProjects() returned %d projects, want %d", len(projects), len(projectIDs))
	}
	
	// Verify all projects are present
	projectMap := make(map[string]bool)
	for _, p := range projects {
		projectMap[p] = true
	}
	
	for _, id := range projectIDs {
		if !projectMap[id] {
			t.Errorf("Missing project: %s", id)
		}
	}
}

func TestKeystore_EnvironmentKeys(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "keystore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ks.Close()
	
	// Test data
	projectID := "test-project"
	environment := "production"
	entry := &EnvironmentKeyEntry{
		ProjectID:        projectID,
		Environment:      environment,
		Salt:             []byte("env-salt-32-bytes-for-testing!!!"),
		VerificationHash: "env-hash",
		CreatedAt:        time.Now(),
		Algorithm:        "argon2id",
		Iterations:       3,
		Memory:           65536,
		Parallelism:      4,
	}
	
	// Store environment key
	err = ks.StoreEnvironmentKey(projectID, environment, entry)
	if err != nil {
		t.Fatalf("StoreEnvironmentKey() error = %v", err)
	}
	
	// Retrieve environment key
	retrieved, err := ks.GetEnvironmentKey(projectID, environment)
	if err != nil {
		t.Fatalf("GetEnvironmentKey() error = %v", err)
	}
	
	// Verify retrieved data
	if retrieved.ProjectID != entry.ProjectID {
		t.Errorf("ProjectID = %v, want %v", retrieved.ProjectID, entry.ProjectID)
	}
	
	if retrieved.Environment != entry.Environment {
		t.Errorf("Environment = %v, want %v", retrieved.Environment, entry.Environment)
	}
	
	if retrieved.Algorithm != entry.Algorithm {
		t.Errorf("Algorithm = %v, want %v", retrieved.Algorithm, entry.Algorithm)
	}
	
	if retrieved.Iterations != entry.Iterations {
		t.Errorf("Iterations = %v, want %v", retrieved.Iterations, entry.Iterations)
	}
	
	// Test not found
	_, err = ks.GetEnvironmentKey(projectID, "non-existent")
	if err != ErrKeyNotFound {
		t.Errorf("GetEnvironmentKey() non-existent error = %v, want %v", err, ErrKeyNotFound)
	}
}

func TestKeystore_DeleteEnvironmentKey(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "keystore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ks.Close()
	
	// Store an environment key
	projectID := "test-project"
	environment := "staging"
	entry := &EnvironmentKeyEntry{
		ProjectID:        projectID,
		Environment:      environment,
		Salt:             []byte("salt"),
		VerificationHash: "hash",
		CreatedAt:        time.Now(),
	}
	
	ks.StoreEnvironmentKey(projectID, environment, entry)
	
	// Delete environment key
	err = ks.DeleteEnvironmentKey(projectID, environment)
	if err != nil {
		t.Fatalf("DeleteEnvironmentKey() error = %v", err)
	}
	
	// Verify deletion
	_, err = ks.GetEnvironmentKey(projectID, environment)
	if err != ErrKeyNotFound {
		t.Errorf("GetEnvironmentKey() after delete error = %v, want %v", err, ErrKeyNotFound)
	}
	
	// Try to delete non-existent key
	err = ks.DeleteEnvironmentKey(projectID, "non-existent")
	if err != ErrKeyNotFound {
		t.Errorf("DeleteEnvironmentKey() non-existent error = %v, want %v", err, ErrKeyNotFound)
	}
}

func TestKeystore_ListEnvironments(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "keystore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ks.Close()
	
	projectID := "test-project"
	
	// Test empty list
	environments, err := ks.ListEnvironments(projectID)
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v", err)
	}
	
	if len(environments) != 0 {
		t.Errorf("ListEnvironments() = %v, want empty", environments)
	}
	
	// Add some environments
	envs := []string{"development", "staging", "production"}
	for _, env := range envs {
		entry := &EnvironmentKeyEntry{
			ProjectID:        projectID,
			Environment:      env,
			Salt:             []byte("salt"),
			VerificationHash: "hash",
			CreatedAt:        time.Now(),
		}
		ks.StoreEnvironmentKey(projectID, env, entry)
	}
	
	// List environments
	environments, err = ks.ListEnvironments(projectID)
	if err != nil {
		t.Fatalf("ListEnvironments() error = %v", err)
	}
	
	if len(environments) != len(envs) {
		t.Errorf("ListEnvironments() returned %d environments, want %d", len(environments), len(envs))
	}
	
	// Verify all environments are present and sorted
	for i, env := range environments {
		if env != envs[i] {
			t.Errorf("Environment[%d] = %v, want %v", i, env, envs[i])
		}
	}
	
	// Test different project
	otherEnvironments, err := ks.ListEnvironments("other-project")
	if err != nil {
		t.Fatalf("ListEnvironments() other project error = %v", err)
	}
	
	if len(otherEnvironments) != 0 {
		t.Errorf("ListEnvironments() other project = %v, want empty", otherEnvironments)
	}
}

func TestKeystore_BackupRestore(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "keystore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ks.Close()
	
	// Store some data
	projectID := "backup-test"
	entry := &KeyEntry{
		ProjectID:        projectID,
		Salt:             []byte("backup-salt"),
		VerificationHash: "backup-hash",
		CreatedAt:        time.Now(),
	}
	
	ks.StoreKey(projectID, entry)
	
	// Create backup
	backupPath := filepath.Join(tmpDir, "backup.db")
	err = ks.Backup(backupPath)
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}
	
	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}
	
	// Delete the key
	ks.DeleteKey(projectID)
	
	// Verify key is gone
	_, err = ks.GetKey(projectID)
	if err != ErrKeyNotFound {
		t.Error("Key should be deleted before restore")
	}
	
	// Restore from backup
	err = ks.Restore(backupPath)
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	
	// Verify data is restored
	restored, err := ks.GetKey(projectID)
	if err != nil {
		t.Fatalf("GetKey() after restore error = %v", err)
	}
	
	if restored.VerificationHash != entry.VerificationHash {
		t.Errorf("Restored VerificationHash = %v, want %v", restored.VerificationHash, entry.VerificationHash)
	}
}

func TestKeystore_Schema(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "keystore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ks.Close()
	
	// Verify schema version
	var version int
	err = ks.db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to query schema version: %v", err)
	}
	
	// Should have migrations 1 and 2 applied
	if version != 2 {
		t.Errorf("Schema version = %d, want 2", version)
	}
	
	// Verify tables exist
	tables := []string{"keys", "environment_keys", "schema_version"}
	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='%s'", table)
		err = ks.db.QueryRow(query).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		
		if count != 1 {
			t.Errorf("Table %s not found", table)
		}
	}
	
	// Verify indexes
	indexes := []string{"idx_keys_updated_at", "idx_env_keys_project", "idx_env_keys_updated"}
	for _, index := range indexes {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='%s'", index)
		err = ks.db.QueryRow(query).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to check index %s: %v", index, err)
		}
		
		if count != 1 {
			t.Errorf("Index %s not found", index)
		}
	}
}

func TestKeyEntry_Serialization(t *testing.T) {
	entry := &KeyEntry{
		ProjectID:        "test-project",
		Salt:             []byte("test-salt"),
		VerificationHash: "test-hash",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	
	// Serialize
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	
	// Deserialize
	var decoded KeyEntry
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	
	// Verify
	if decoded.ProjectID != entry.ProjectID {
		t.Errorf("ProjectID = %v, want %v", decoded.ProjectID, entry.ProjectID)
	}
	
	if string(decoded.Salt) != string(entry.Salt) {
		t.Errorf("Salt = %v, want %v", decoded.Salt, entry.Salt)
	}
	
	if decoded.VerificationHash != entry.VerificationHash {
		t.Errorf("VerificationHash = %v, want %v", decoded.VerificationHash, entry.VerificationHash)
	}
}

func TestEnvironmentKeyEntry_Serialization(t *testing.T) {
	entry := &EnvironmentKeyEntry{
		ProjectID:        "test-project",
		Environment:      "production",
		Salt:             []byte("env-salt"),
		VerificationHash: "env-hash",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Algorithm:        "argon2id",
		Iterations:       3,
		Memory:           65536,
		Parallelism:      4,
	}
	
	// Serialize
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	
	// Deserialize
	var decoded EnvironmentKeyEntry
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	
	// Verify
	if decoded.Environment != entry.Environment {
		t.Errorf("Environment = %v, want %v", decoded.Environment, entry.Environment)
	}
	
	if decoded.Algorithm != entry.Algorithm {
		t.Errorf("Algorithm = %v, want %v", decoded.Algorithm, entry.Algorithm)
	}
	
	if decoded.Iterations != entry.Iterations {
		t.Errorf("Iterations = %v, want %v", decoded.Iterations, entry.Iterations)
	}
}

func TestKeystore_ConcurrentAccess(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "keystore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer ks.Close()
	
	// SQLite should handle concurrent access with WAL mode
	done := make(chan bool, 10)
	
	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			projectID := fmt.Sprintf("project-%d", n)
			entry := &KeyEntry{
				ProjectID:        projectID,
				Salt:             []byte(fmt.Sprintf("salt-%d", n)),
				VerificationHash: fmt.Sprintf("hash-%d", n),
				CreatedAt:        time.Now(),
			}
			
			if err := ks.StoreKey(projectID, entry); err != nil {
				t.Errorf("Concurrent StoreKey() error = %v", err)
			}
			
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify all keys were stored
	projects, err := ks.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	
	if len(projects) != 10 {
		t.Errorf("Expected 10 projects, got %d", len(projects))
	}
}

func BenchmarkKeystore_StoreKey(b *testing.B) {
	tmpDir, err := ioutil.TempDir("", "keystore_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		b.Fatal(err)
	}
	defer ks.Close()
	
	entry := &KeyEntry{
		Salt:             []byte("benchmark-salt-32-bytes-testing!"),
		VerificationHash: "benchmark-hash",
		CreatedAt:        time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		projectID := fmt.Sprintf("project-%d", i)
		entry.ProjectID = projectID
		_ = ks.StoreKey(projectID, entry)
	}
}

func BenchmarkKeystore_GetKey(b *testing.B) {
	tmpDir, err := ioutil.TempDir("", "keystore_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	ks, err := NewKeystore(tmpDir)
	if err != nil {
		b.Fatal(err)
	}
	defer ks.Close()
	
	// Pre-populate
	for i := 0; i < 100; i++ {
		projectID := fmt.Sprintf("project-%d", i)
		entry := &KeyEntry{
			ProjectID:        projectID,
			Salt:             []byte("salt"),
			VerificationHash: "hash",
			CreatedAt:        time.Now(),
		}
		ks.StoreKey(projectID, entry)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		projectID := fmt.Sprintf("project-%d", i%100)
		_, _ = ks.GetKey(projectID)
	}
}