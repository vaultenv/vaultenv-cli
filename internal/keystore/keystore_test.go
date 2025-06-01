package keystore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewKeystore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "keystore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	ks, err := NewKeystore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()
	
	// Verify database file was created
	dbPath := filepath.Join(tempDir, keystoreDBName)
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("Database file not created: %v", err)
	}
	
	// Test creating keystore in non-existent directory
	nonExistentDir := filepath.Join(tempDir, "subdir", "subdir2")
	ks2, err := NewKeystore(nonExistentDir)
	if err != nil {
		t.Errorf("Failed to create keystore in non-existent dir: %v", err)
	}
	if ks2 != nil {
		ks2.Close()
	}
}

func TestKeystore_StoreAndGetKey(t *testing.T) {
	ks, cleanup := setupTestKeystore(t)
	defer cleanup()
	
	projectID := "test-project"
	entry := &KeyEntry{
		ProjectID:        projectID,
		Salt:             []byte("test-salt"),
		VerificationHash: "test-hash",
		CreatedAt:        time.Now(),
	}
	
	// Store key
	err := ks.StoreKey(projectID, entry)
	if err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}
	
	// Get key
	retrieved, err := ks.GetKey(projectID)
	if err != nil {
		t.Fatalf("Failed to get key: %v", err)
	}
	
	// Verify fields
	if retrieved.ProjectID != entry.ProjectID {
		t.Errorf("ProjectID mismatch: got %s, want %s", retrieved.ProjectID, entry.ProjectID)
	}
	if string(retrieved.Salt) != string(entry.Salt) {
		t.Errorf("Salt mismatch: got %s, want %s", retrieved.Salt, entry.Salt)
	}
	if retrieved.VerificationHash != entry.VerificationHash {
		t.Errorf("VerificationHash mismatch: got %s, want %s", retrieved.VerificationHash, entry.VerificationHash)
	}
	
	// Test getting non-existent key
	_, err = ks.GetKey("non-existent")
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestKeystore_UpdateKey(t *testing.T) {
	ks, cleanup := setupTestKeystore(t)
	defer cleanup()
	
	projectID := "test-project"
	entry := &KeyEntry{
		ProjectID:        projectID,
		Salt:             []byte("original-salt"),
		VerificationHash: "original-hash",
		CreatedAt:        time.Now(),
	}
	
	// Store original
	err := ks.StoreKey(projectID, entry)
	if err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}
	
	// Update key
	updatedEntry := &KeyEntry{
		ProjectID:        projectID,
		Salt:             []byte("updated-salt"),
		VerificationHash: "updated-hash",
		CreatedAt:        entry.CreatedAt,
	}
	
	time.Sleep(10 * time.Millisecond) // Ensure UpdatedAt is different
	err = ks.StoreKey(projectID, updatedEntry)
	if err != nil {
		t.Fatalf("Failed to update key: %v", err)
	}
	
	// Verify update
	retrieved, err := ks.GetKey(projectID)
	if err != nil {
		t.Fatalf("Failed to get updated key: %v", err)
	}
	
	if string(retrieved.Salt) != string(updatedEntry.Salt) {
		t.Errorf("Salt not updated: got %s, want %s", retrieved.Salt, updatedEntry.Salt)
	}
	if retrieved.VerificationHash != updatedEntry.VerificationHash {
		t.Errorf("VerificationHash not updated: got %s, want %s", retrieved.VerificationHash, updatedEntry.VerificationHash)
	}
	if !retrieved.UpdatedAt.After(retrieved.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt after update")
	}
}

func TestKeystore_DeleteKey(t *testing.T) {
	ks, cleanup := setupTestKeystore(t)
	defer cleanup()
	
	projectID := "test-project"
	entry := &KeyEntry{
		ProjectID:        projectID,
		Salt:             []byte("test-salt"),
		VerificationHash: "test-hash",
		CreatedAt:        time.Now(),
	}
	
	// Store key
	err := ks.StoreKey(projectID, entry)
	if err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}
	
	// Delete key
	err = ks.DeleteKey(projectID)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}
	
	// Verify deletion
	_, err = ks.GetKey(projectID)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound after deletion, got %v", err)
	}
	
	// Test deleting non-existent key
	err = ks.DeleteKey("non-existent")
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound for non-existent key, got %v", err)
	}
}

func TestKeystore_ListProjects(t *testing.T) {
	ks, cleanup := setupTestKeystore(t)
	defer cleanup()
	
	// Initially empty
	projects, err := ks.ListProjects()
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}
	
	// Add projects
	projectIDs := []string{"project1", "project2", "project3"}
	for i, pid := range projectIDs {
		entry := &KeyEntry{
			ProjectID:        pid,
			Salt:             []byte("salt"),
			VerificationHash: "hash",
			CreatedAt:        time.Now(),
		}
		if err := ks.StoreKey(pid, entry); err != nil {
			t.Fatalf("Failed to store key %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}
	
	// List projects
	projects, err = ks.ListProjects()
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}
	
	if len(projects) != len(projectIDs) {
		t.Errorf("Expected %d projects, got %d", len(projectIDs), len(projects))
	}
	
	// Verify order (should be by updated_at DESC)
	// Most recently updated should be first
	if projects[0] != "project3" {
		t.Errorf("Expected project3 first (most recent), got %s", projects[0])
	}
}

func TestKeystore_BackupRestore(t *testing.T) {
	ks, cleanup := setupTestKeystore(t)
	defer cleanup()
	
	// Store some data
	projectID := "backup-test-project"
	entry := &KeyEntry{
		ProjectID:        projectID,
		Salt:             []byte("backup-test-salt"),
		VerificationHash: "backup-test-hash",
		CreatedAt:        time.Now(),
	}
	
	if err := ks.StoreKey(projectID, entry); err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}
	
	// Create backup
	tempDir, err := os.MkdirTemp("", "keystore-backup-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	backupPath := filepath.Join(tempDir, "backup.db")
	if err := ks.Backup(backupPath); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	
	// Verify backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("Backup file not created: %v", err)
	}
	
	// Delete the key
	if err := ks.DeleteKey(projectID); err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}
	
	// Verify key is gone
	_, err = ks.GetKey(projectID)
	if err != ErrKeyNotFound {
		t.Error("Key should be deleted before restore")
	}
	
	// Restore from backup
	if err := ks.Restore(backupPath); err != nil {
		t.Fatalf("Failed to restore from backup: %v", err)
	}
	
	// Verify data is restored
	restored, err := ks.GetKey(projectID)
	if err != nil {
		t.Fatalf("Failed to get restored key: %v", err)
	}
	
	if string(restored.Salt) != string(entry.Salt) {
		t.Errorf("Restored salt mismatch: got %s, want %s", restored.Salt, entry.Salt)
	}
	if restored.VerificationHash != entry.VerificationHash {
		t.Errorf("Restored hash mismatch: got %s, want %s", restored.VerificationHash, entry.VerificationHash)
	}
	
	// Test restore from non-existent file
	err = ks.Restore("/non/existent/backup.db")
	if err == nil {
		t.Error("Restore from non-existent file should fail")
	}
}

// Helper function to setup test keystore
func setupTestKeystore(t *testing.T) (*Keystore, func()) {
	tempDir, err := os.MkdirTemp("", "keystore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	ks, err := NewKeystore(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create keystore: %v", err)
	}
	
	cleanup := func() {
		ks.Close()
		os.RemoveAll(tempDir)
	}
	
	return ks, cleanup
}