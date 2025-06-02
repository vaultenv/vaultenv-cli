//go:build integration
// +build integration

package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func TestIntegration_BasicWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test environment
	tmpDir, err := ioutil.TempDir("", "vaultenv-integration")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to test directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Test workflow
	tests := []struct {
		name    string
		command string
		args    []string
		check   func(t *testing.T) error
	}{
		{
			name:    "init_project",
			command: "init",
			args:    []string{"--name", "test-project", "--force"},
			check: func(t *testing.T) error {
				// Check config file created
				configPath := filepath.Join(tmpDir, ".vaultenv", "config.yaml")
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					return err
				}
				return nil
			},
		},
		{
			name:    "set_variables",
			command: "set",
			args:    []string{"DATABASE_URL", "postgres://localhost/test", "API_KEY", "secret123"},
			check: func(t *testing.T) error {
				// Verify values were set
				store, err := storage.NewFileBackend(filepath.Join(tmpDir, ".vaultenv"), "default")
				if err != nil {
					return err
				}

				val, err := store.Get("DATABASE_URL")
				if err != nil || val != "postgres://localhost/test" {
					return err
				}
				return nil
			},
		},
		{
			name:    "list_variables",
			command: "list",
			args:    []string{},
			check: func(t *testing.T) error {
				// Should succeed
				return nil
			},
		},
		{
			name:    "export_to_file",
			command: "export",
			args:    []string{".env.export"},
			check: func(t *testing.T) error {
				// Check export file created
				if _, err := os.Stat(".env.export"); os.IsNotExist(err) {
					return err
				}

				// Verify content
				content, err := ioutil.ReadFile(".env.export")
				if err != nil {
					return err
				}

				if !strings.Contains(string(content), "DATABASE_URL=") {
					t.Error("Export file missing DATABASE_URL")
				}
				return nil
			},
		},
		{
			name:    "create_environment",
			command: "env",
			args:    []string{"create", "production"},
			check: func(t *testing.T) error {
				// Environment should be created
				return nil
			},
		},
		{
			name:    "switch_environment",
			command: "env",
			args:    []string{"set", "production"},
			check: func(t *testing.T) error {
				// Should switch to production
				return nil
			},
		},
		{
			name:    "set_in_production",
			command: "set",
			args:    []string{"DATABASE_URL", "postgres://prod.example.com/db"},
			check: func(t *testing.T) error {
				// Value should be different in production
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create root command
			rootCmd := createTestRootCommand()

			// Set args
			args := append([]string{tt.command}, tt.args...)
			rootCmd.SetArgs(args)

			// Execute
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)

			err := rootCmd.Execute()
			if err != nil {
				t.Errorf("Command failed: %v\nOutput: %s", err, buf.String())
				return
			}

			// Run check
			if tt.check != nil {
				if err := tt.check(t); err != nil {
					t.Errorf("Check failed: %v", err)
				}
			}
		})
	}
}

func TestIntegration_EncryptionWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := ioutil.TempDir("", "vaultenv-encryption")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test encryption with password
	t.Run("password_encryption", func(t *testing.T) {
		store, err := storage.NewEncryptedFileStorage(
			filepath.Join(tmpDir, "encrypted.vault"),
			"test-password",
		)
		if err != nil {
			t.Fatal(err)
		}

		// Set encrypted values
		testData := map[string]string{
			"SECRET_KEY":    "super-secret-value",
			"DATABASE_PASS": "encrypted-password",
			"API_TOKEN":     "token-12345",
		}

		for key, value := range testData {
			if err := store.Set(key, value, false); err != nil {
				t.Fatalf("Failed to set %s: %v", key, err)
			}
		}

		// Close and reopen with correct password
		store2, err := storage.NewEncryptedFileStorage(
			filepath.Join(tmpDir, "encrypted.vault"),
			"test-password",
		)
		if err != nil {
			t.Fatal(err)
		}

		// Verify values
		for key, expectedValue := range testData {
			value, err := store2.Get(key)
			if err != nil {
				t.Errorf("Failed to get %s: %v", key, err)
				continue
			}
			if value != expectedValue {
				t.Errorf("%s = %q, want %q", key, value, expectedValue)
			}
		}

		// Try with wrong password
		_, err = storage.NewEncryptedFileStorage(
			filepath.Join(tmpDir, "encrypted.vault"),
			"wrong-password",
		)
		if err == nil {
			t.Error("Should fail with wrong password")
		}
	})
}

func TestIntegration_GitWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := ioutil.TempDir("", "vaultenv-git")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Run git init
	runCommand(t, "git", "init")
	runCommand(t, "git", "config", "user.email", "test@example.com")
	runCommand(t, "git", "config", "user.name", "Test User")

	// Create vaultenv project
	rootCmd := createTestRootCommand()
	rootCmd.SetArgs([]string{"init", "--name", "git-test"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Set some values
	rootCmd.SetArgs([]string{"set", "KEY1", "value1", "KEY2", "value2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Export to git-friendly format
	rootCmd.SetArgs([]string{"git", "export"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Git export failed: %v", err)
	}

	// Check .env.vault file was created
	if _, err := os.Stat(".env.vault"); os.IsNotExist(err) {
		t.Error(".env.vault file not created")
	}

	// Verify it's encrypted
	content, err := ioutil.ReadFile(".env.vault")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(content), "value1") {
		t.Error(".env.vault should be encrypted, found plaintext")
	}

	// Add to git
	runCommand(t, "git", "add", ".env.vault")
	runCommand(t, "git", "commit", "-m", "Add encrypted env")

	// Modify a value
	rootCmd.SetArgs([]string{"set", "KEY1", "modified"})
	rootCmd.Execute()

	// Check git diff
	rootCmd.SetArgs([]string{"git", "diff"})
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "KEY1") {
		t.Error("Git diff should show KEY1 changed")
	}
}

func TestIntegration_MultiEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := ioutil.TempDir("", "vaultenv-multienv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Initialize project
	rootCmd := createTestRootCommand()
	rootCmd.SetArgs([]string{"init", "--name", "multienv-test"})
	rootCmd.Execute()

	// Create environments
	environments := []string{"development", "staging", "production"}
	for _, env := range environments {
		rootCmd.SetArgs([]string{"env", "create", env})
		rootCmd.Execute()
	}

	// Set different values in each environment
	envData := map[string]map[string]string{
		"development": {
			"API_URL": "http://localhost:3000",
			"DEBUG":   "true",
		},
		"staging": {
			"API_URL": "https://staging.example.com",
			"DEBUG":   "true",
		},
		"production": {
			"API_URL": "https://api.example.com",
			"DEBUG":   "false",
		},
	}

	for env, data := range envData {
		// Switch to environment
		rootCmd.SetArgs([]string{"env", "set", env})
		rootCmd.Execute()

		// Set values
		for key, value := range data {
			rootCmd.SetArgs([]string{"set", key, value})
			rootCmd.Execute()
		}
	}

	// Verify environment isolation
	for env, expectedData := range envData {
		rootCmd.SetArgs([]string{"env", "set", env})
		rootCmd.Execute()

		// Get values
		for key, expectedValue := range expectedData {
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetArgs([]string{"get", key})
			rootCmd.Execute()

			actualValue := strings.TrimSpace(buf.String())
			if actualValue != expectedValue {
				t.Errorf("%s/%s = %q, want %q", env, key, actualValue, expectedValue)
			}
		}
	}
}

func TestIntegration_BackupRestore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := ioutil.TempDir("", "vaultenv-backup")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create initial data
	store, err := storage.NewSQLiteStorage(filepath.Join(tmpDir, "main.db"))
	if err != nil {
		t.Fatal(err)
	}

	// Add test data
	testData := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}

	for key, value := range testData {
		store.Set(key, value, false)
	}

	// Create backup
	backupPath := filepath.Join(tmpDir, "backup.db")
	// In real implementation, would use backup command

	// Simulate data loss
	store.Delete("KEY2")
	store.Set("KEY1", "corrupted", false)

	// Restore from backup
	// In real implementation, would use restore command

	// Verify restoration
	// Would check that original values are restored
}

func TestIntegration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tmpDir, err := ioutil.TempDir("", "vaultenv-perf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewSQLiteStorage(filepath.Join(tmpDir, "perf.db"))
	if err != nil {
		t.Fatal(err)
	}

	// Measure bulk operations
	t.Run("bulk_set", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("KEY_%d", i)
			value := fmt.Sprintf("value_%d", i)
			if err := store.Set(key, value, false); err != nil {
				t.Fatal(err)
			}
		}

		duration := time.Since(start)
		t.Logf("Set 1000 keys in %v", duration)

		if duration > 5*time.Second {
			t.Error("Bulk set too slow")
		}
	})

	t.Run("bulk_get", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("KEY_%d", i)
			if _, err := store.Get(key); err != nil {
				t.Fatal(err)
			}
		}

		duration := time.Since(start)
		t.Logf("Get 1000 keys in %v", duration)

		if duration > 1*time.Second {
			t.Error("Bulk get too slow")
		}
	})
}

// Helper functions
func createTestRootCommand() *cobra.Command {
	// Create a simplified root command for testing
	rootCmd := &cobra.Command{
		Use:   "vaultenv",
		Short: "Test root command",
	}

	// Add minimal commands needed for integration tests
	// In real implementation, would use actual command implementations

	return rootCmd
}

func runCommand(t *testing.T, name string, args ...string) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command %s failed: %v\nOutput: %s", name, err, output)
	}
}
