package test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/vaultenv/vaultenv-cli/pkg/encryption"
	"github.com/vaultenv/vaultenv-cli/pkg/keystore"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

// TestEnvironment provides a complete test environment for CLI testing
type TestEnvironment struct {
	t          *testing.T
	HomeDir    string
	ConfigDir  string
	WorkDir    string
	Storage    storage.Backend
	Keystore   keystore.Keystore
	Encryptor  encryption.Encryptor
	CleanupFns []func()
}

// NewTestEnvironment creates a new test environment with isolated directories
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// Create temporary directories
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configDir := filepath.Join(homeDir, ".vaultenv-cli")
	workDir := filepath.Join(tempDir, "work")

	// Create directories
	require.NoError(t, os.MkdirAll(homeDir, 0755))
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(workDir, 0755))

	env := &TestEnvironment{
		t:          t,
		HomeDir:    homeDir,
		ConfigDir:  configDir,
		WorkDir:    workDir,
		Storage:    storage.NewMemoryBackend(),
		Keystore:   keystore.NewMockKeystore(),
		Encryptor:  encryption.NewAESGCMEncryptor(),
		CleanupFns: []func(){},
	}

	// Set environment variables
	env.SetEnv("HOME", homeDir)
	env.SetEnv("VAULTENV_CONFIG_DIR", configDir)
	env.SetEnv("VAULTENV_TEST", "1")

	// Set the test backend
	storage.SetTestBackend(env.Storage)
	env.AddCleanup(func() {
		storage.ResetTestBackend()
	})

	// Change to work directory
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workDir))

	env.AddCleanup(func() {
		os.Chdir(oldDir)
	})

	return env
}

// SetEnv sets an environment variable and schedules cleanup
func (e *TestEnvironment) SetEnv(key, value string) {
	e.t.Helper()

	oldValue, exists := os.LookupEnv(key)
	os.Setenv(key, value)

	e.AddCleanup(func() {
		if exists {
			os.Setenv(key, oldValue)
		} else {
			os.Unsetenv(key)
		}
	})
}

// AddCleanup adds a cleanup function to be called when the test finishes
func (e *TestEnvironment) AddCleanup(fn func()) {
	e.CleanupFns = append(e.CleanupFns, fn)
}

// Cleanup runs all cleanup functions
func (e *TestEnvironment) Cleanup() {
	for i := len(e.CleanupFns) - 1; i >= 0; i-- {
		e.CleanupFns[i]()
	}
}

// WriteFile writes a file in the test environment
func (e *TestEnvironment) WriteFile(path string, content string) {
	e.t.Helper()

	fullPath := filepath.Join(e.WorkDir, path)
	dir := filepath.Dir(fullPath)

	require.NoError(e.t, os.MkdirAll(dir, 0755))
	require.NoError(e.t, os.WriteFile(fullPath, []byte(content), 0644))
}

// ReadFile reads a file from the test environment
func (e *TestEnvironment) ReadFile(path string) string {
	e.t.Helper()

	fullPath := filepath.Join(e.WorkDir, path)
	content, err := os.ReadFile(fullPath)
	require.NoError(e.t, err)

	return string(content)
}

// CommandOutput captures the output of a cobra command
type CommandOutput struct {
	Stdout string
	Stderr string
}

// ExecuteCommand executes a cobra command and captures its output
func ExecuteCommand(cmd *cobra.Command, args ...string) (*CommandOutput, error) {
	// Create buffers for output
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	// Set output
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)

	// Execute command
	err := cmd.Execute()

	return &CommandOutput{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, err
}

// CaptureStdin temporarily replaces stdin for testing interactive prompts
func CaptureStdin(input string) (restore func()) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write input to pipe
	go func() {
		defer w.Close()
		io.WriteString(w, input)
	}()

	return func() {
		os.Stdin = oldStdin
		r.Close()
	}
}

// AssertFileExists checks if a file exists
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	require.NoError(t, err, "file should exist: %s", path)
}

// AssertFileNotExists checks if a file does not exist
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err), "file should not exist: %s", path)
}

// AssertFileContains checks if a file contains a string
func AssertFileContains(t *testing.T, path string, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(content), expected)
}

// MockStorage creates a pre-populated mock storage for testing
func MockStorage() *storage.MemoryBackend {
	backend := storage.NewMemoryBackend()

	// Add some test data
	backend.Set("TEST_VAR", "test_value", false)
	backend.Set("DATABASE_URL", "postgres://localhost/test", false)
	backend.Set("API_KEY", "test-api-key", false)

	return backend
}

// ParseVariables is a helper function for testing variable parsing
func ParseVariables(args []string) (map[string]string, error) {
	result := make(map[string]string)

	for _, arg := range args {
		parts := bytes.SplitN([]byte(arg), []byte("="), 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid format")
		}

		key := string(parts[0])
		value := string(parts[1])

		// Validate variable name
		if key == "" {
			return nil, errors.New("invalid variable name")
		}

		result[key] = value
	}

	return result, nil
}
