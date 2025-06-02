package access

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAccessLevel_Constants(t *testing.T) {
	// Verify access level constants
	if AccessLevelRead != "read" {
		t.Errorf("AccessLevelRead = %v, want 'read'", AccessLevelRead)
	}
	if AccessLevelWrite != "write" {
		t.Errorf("AccessLevelWrite = %v, want 'write'", AccessLevelWrite)
	}
	if AccessLevelAdmin != "admin" {
		t.Errorf("AccessLevelAdmin = %v, want 'admin'", AccessLevelAdmin)
	}
}

func TestLocalAccessControl_HasAccess(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "access_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	ac := NewLocalAccessControl(configPath)

	// Setup test access config
	config := &AccessConfig{
		Environments: map[string]*EnvironmentAccess{
			"dev": {
				AllowedUsers: []string{"user1", "user2"},
				AllowedRoles: []string{"developer"},
				Entries: []AccessEntry{
					{
						User:        "user3",
						Environment: "dev",
						Level:       AccessLevelRead,
						GrantedAt:   time.Now(),
						GrantedBy:   "admin",
					},
				},
			},
			"prod": {
				AllowedUsers: []string{"admin"},
				AllowedRoles: []string{"admin"},
				Entries:      []AccessEntry{},
			},
			"wildcard": {
				AllowedUsers: []string{"*"},
				Entries:      []AccessEntry{},
			},
		},
	}

	// Save config
	if err := ac.saveAccessConfig(config); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	tests := []struct {
		name        string
		user        string
		environment string
		wantAccess  bool
	}{
		{"allowed_user", "user1", "dev", true},
		{"allowed_user2", "user2", "dev", true},
		{"entry_user", "user3", "dev", true},
		{"not_allowed", "user4", "dev", false},
		{"admin_prod", "admin", "prod", true},
		{"non_admin_prod", "user1", "prod", false},
		{"wildcard_env", "anyone", "wildcard", true},
		{"no_env_config", "user1", "staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasAccess, err := ac.HasAccess(tt.user, tt.environment)
			if err != nil {
				t.Errorf("HasAccess() error = %v", err)
				return
			}
			if hasAccess != tt.wantAccess {
				t.Errorf("HasAccess() = %v, want %v", hasAccess, tt.wantAccess)
			}
		})
	}
}

func TestLocalAccessControl_HasAccess_Expiry(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "access_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	ac := NewLocalAccessControl(configPath)

	// Setup test with expired and non-expired entries
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	config := &AccessConfig{
		Environments: map[string]*EnvironmentAccess{
			"test": {
				Entries: []AccessEntry{
					{
						User:        "expired_user",
						Environment: "test",
						Level:       AccessLevelRead,
						GrantedAt:   time.Now().Add(-2 * time.Hour),
						ExpiresAt:   &past,
					},
					{
						User:        "valid_user",
						Environment: "test",
						Level:       AccessLevelRead,
						GrantedAt:   time.Now().Add(-1 * time.Hour),
						ExpiresAt:   &future,
					},
					{
						User:        "permanent_user",
						Environment: "test",
						Level:       AccessLevelRead,
						GrantedAt:   time.Now(),
						ExpiresAt:   nil, // No expiry
					},
				},
			},
		},
	}

	ac.saveAccessConfig(config)

	// Test expired access
	hasAccess, _ := ac.HasAccess("expired_user", "test")
	if hasAccess {
		t.Error("Expired user should not have access")
	}

	// Test valid access with future expiry
	hasAccess, _ = ac.HasAccess("valid_user", "test")
	if !hasAccess {
		t.Error("Valid user with future expiry should have access")
	}

	// Test permanent access
	hasAccess, _ = ac.HasAccess("permanent_user", "test")
	if !hasAccess {
		t.Error("Permanent user should have access")
	}
}

func TestLocalAccessControl_GrantAccess(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "access_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	ac := NewLocalAccessControl(configPath)

	// Grant access to new user
	err = ac.GrantAccess("newuser", "dev", AccessLevelWrite)
	if err != nil {
		t.Fatalf("GrantAccess() error = %v", err)
	}

	// Verify access was granted
	hasAccess, _ := ac.HasAccess("newuser", "dev")
	if !hasAccess {
		t.Error("User should have access after grant")
	}

	// Check that entry was created correctly
	entries, _ := ac.ListAccess("dev")
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.User != "newuser" {
		t.Errorf("Entry user = %v, want 'newuser'", entry.User)
	}
	if entry.Level != AccessLevelWrite {
		t.Errorf("Entry level = %v, want %v", entry.Level, AccessLevelWrite)
	}

	// Update existing access
	err = ac.GrantAccess("newuser", "dev", AccessLevelAdmin)
	if err != nil {
		t.Fatalf("GrantAccess() update error = %v", err)
	}

	// Verify update
	entries, _ = ac.ListAccess("dev")
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry after update, got %d", len(entries))
	}

	if entries[0].Level != AccessLevelAdmin {
		t.Errorf("Updated entry level = %v, want %v", entries[0].Level, AccessLevelAdmin)
	}
}

func TestLocalAccessControl_RevokeAccess(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "access_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	ac := NewLocalAccessControl(configPath)

	// Grant access first
	ac.GrantAccess("user1", "dev", AccessLevelRead)
	ac.GrantAccess("user2", "dev", AccessLevelWrite)

	// Verify both have access
	hasAccess, _ := ac.HasAccess("user1", "dev")
	if !hasAccess {
		t.Error("user1 should have access before revoke")
	}

	// Revoke access for user1
	err = ac.RevokeAccess("user1", "dev")
	if err != nil {
		t.Fatalf("RevokeAccess() error = %v", err)
	}

	// Verify access was revoked
	hasAccess, _ = ac.HasAccess("user1", "dev")
	if hasAccess {
		t.Error("user1 should not have access after revoke")
	}

	// Verify user2 still has access
	hasAccess, _ = ac.HasAccess("user2", "dev")
	if !hasAccess {
		t.Error("user2 should still have access")
	}

	// Revoke non-existent access (should not error)
	err = ac.RevokeAccess("nonexistent", "dev")
	if err != nil {
		t.Errorf("RevokeAccess() for non-existent user error = %v", err)
	}
}

func TestLocalAccessControl_ListAccess(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "access_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	ac := NewLocalAccessControl(configPath)

	// Test empty list
	entries, err := ac.ListAccess("dev")
	if err != nil {
		t.Fatalf("ListAccess() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("ListAccess() = %d entries, want 0", len(entries))
	}

	// Grant access to multiple users
	ac.GrantAccess("user1", "dev", AccessLevelRead)
	ac.GrantAccess("user2", "dev", AccessLevelWrite)
	ac.GrantAccess("user3", "dev", AccessLevelAdmin)

	// List access
	entries, err = ac.ListAccess("dev")
	if err != nil {
		t.Fatalf("ListAccess() error = %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("ListAccess() = %d entries, want 3", len(entries))
	}

	// Verify entries
	userLevels := make(map[string]AccessLevel)
	for _, entry := range entries {
		userLevels[entry.User] = entry.Level
	}

	if userLevels["user1"] != AccessLevelRead {
		t.Errorf("user1 level = %v, want %v", userLevels["user1"], AccessLevelRead)
	}
	if userLevels["user2"] != AccessLevelWrite {
		t.Errorf("user2 level = %v, want %v", userLevels["user2"], AccessLevelWrite)
	}
	if userLevels["user3"] != AccessLevelAdmin {
		t.Errorf("user3 level = %v, want %v", userLevels["user3"], AccessLevelAdmin)
	}
}

func TestLocalAccessControl_ConfigPersistence(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "access_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")

	// First instance
	ac1 := NewLocalAccessControl(configPath)
	ac1.GrantAccess("user1", "dev", AccessLevelRead)
	ac1.GrantAccess("user2", "prod", AccessLevelAdmin)

	// Second instance should see the same data
	ac2 := NewLocalAccessControl(configPath)

	hasAccess, _ := ac2.HasAccess("user1", "dev")
	if !hasAccess {
		t.Error("Persistence failed: user1 should have access to dev")
	}

	hasAccess, _ = ac2.HasAccess("user2", "prod")
	if !hasAccess {
		t.Error("Persistence failed: user2 should have access to prod")
	}

	// Verify the config file exists and is valid JSON
	accessPath := filepath.Join(filepath.Dir(configPath), "access.json")
	data, err := ioutil.ReadFile(accessPath)
	if err != nil {
		t.Fatalf("Failed to read access.json: %v", err)
	}

	var config AccessConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Invalid JSON in access.json: %v", err)
	}

	if len(config.Environments) != 2 {
		t.Errorf("Expected 2 environments, got %d", len(config.Environments))
	}
}

func TestLocalAccessControl_EmptyConfig(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "access_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	ac := NewLocalAccessControl(configPath)

	// Test with non-existent config file
	hasAccess, err := ac.HasAccess("user", "env")
	if err != nil {
		t.Errorf("HasAccess() with no config error = %v", err)
	}
	if hasAccess {
		t.Error("Should not have access with no config")
	}

	// List should return empty
	entries, err := ac.ListAccess("env")
	if err != nil {
		t.Errorf("ListAccess() with no config error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("ListAccess() = %d entries, want 0", len(entries))
	}
}

func TestAccessEntry_Structure(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)

	entry := AccessEntry{
		User:        "testuser",
		Environment: "dev",
		Level:       AccessLevelWrite,
		GrantedAt:   now,
		GrantedBy:   "admin",
		ExpiresAt:   &future,
	}

	// Verify all fields are set correctly
	if entry.User != "testuser" {
		t.Errorf("User = %v, want 'testuser'", entry.User)
	}
	if entry.Environment != "dev" {
		t.Errorf("Environment = %v, want 'dev'", entry.Environment)
	}
	if entry.Level != AccessLevelWrite {
		t.Errorf("Level = %v, want %v", entry.Level, AccessLevelWrite)
	}
	if entry.GrantedBy != "admin" {
		t.Errorf("GrantedBy = %v, want 'admin'", entry.GrantedBy)
	}
	if entry.ExpiresAt == nil || !entry.ExpiresAt.Equal(future) {
		t.Error("ExpiresAt not set correctly")
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test contains
	slice := []string{"a", "b", "c"}

	if !contains(slice, "b") {
		t.Error("contains() should return true for existing item")
	}
	if contains(slice, "d") {
		t.Error("contains() should return false for non-existing item")
	}
	if contains([]string{}, "a") {
		t.Error("contains() should return false for empty slice")
	}

	// Test removeString
	result := removeString(slice, "b")
	if len(result) != 2 {
		t.Errorf("removeString() returned %d items, want 2", len(result))
	}
	if contains(result, "b") {
		t.Error("removeString() should remove the item")
	}

	// Remove non-existent item
	result = removeString(slice, "d")
	if len(result) != 3 {
		t.Errorf("removeString() of non-existent item returned %d items, want 3", len(result))
	}

	// Test getCurrentUser
	user := getCurrentUser()
	if user == "" {
		t.Error("getCurrentUser() should not return empty string")
	}
}

func TestLocalAccessControl_RoleBasedAccess(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "access_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	ac := NewLocalAccessControl(configPath)

	// Setup config with role-based access
	config := &AccessConfig{
		Environments: map[string]*EnvironmentAccess{
			"dev": {
				AllowedRoles: []string{"developer", "tester"},
				Entries:      []AccessEntry{},
			},
		},
	}

	ac.saveAccessConfig(config)

	// Note: getUserRoles currently returns empty slice
	// So role-based access won't work in tests
	// This is expected behavior for the local version

	hasAccess, _ := ac.HasAccess("user_with_role", "dev")
	if hasAccess {
		t.Error("Role-based access should not work in local version (getUserRoles returns empty)")
	}
}

func BenchmarkLocalAccessControl_HasAccess(b *testing.B) {
	tmpDir, err := ioutil.TempDir("", "access_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	ac := NewLocalAccessControl(configPath)

	// Setup test data
	for i := 0; i < 100; i++ {
		ac.GrantAccess(fmt.Sprintf("user%d", i), "dev", AccessLevelRead)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ac.HasAccess(fmt.Sprintf("user%d", i%100), "dev")
	}
}

func BenchmarkLocalAccessControl_GrantAccess(b *testing.B) {
	tmpDir, err := ioutil.TempDir("", "access_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	ac := NewLocalAccessControl(configPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ac.GrantAccess(fmt.Sprintf("user%d", i), "dev", AccessLevelRead)
	}
}
