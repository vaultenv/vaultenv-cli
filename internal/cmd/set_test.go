package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vaultenv/vaultenv-cli/internal/test"
)

func TestSetCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError bool
		errorMsg  string
		setup     func(t *testing.T, env *test.TestEnvironment)
		verify    func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput)
	}{
		{
			name:      "set single variable",
			args:      []string{"set", "TEST_VAR=value"},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "Variables set successfully")
				assert.Contains(t, output.Stdout, "Setting 1 variable(s)")
			},
		},
		{
			name:      "set multiple variables",
			args:      []string{"set", "VAR1=value1", "VAR2=value2"},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "Variables set successfully")
				assert.Contains(t, output.Stdout, "Setting 2 variable(s)")
			},
		},
		{
			name:      "invalid variable format",
			args:      []string{"set", "INVALID"},
			wantError: true,
			errorMsg:  "invalid format",
		},
		{
			name:      "empty variable name",
			args:      []string{"set", "=value"},
			wantError: true,
			errorMsg:  "invalid variable name",
		},
		{
			name:      "invalid variable name with spaces",
			args:      []string{"set", "INVALID VAR=value"},
			wantError: true,
			errorMsg:  "invalid format",
		},
		{
			name:      "variable name starting with number",
			args:      []string{"set", "1VAR=value"},
			wantError: true,
			errorMsg:  "invalid variable name",
		},
		{
			name:      "variable name with special characters",
			args:      []string{"set", "VAR-NAME=value"},
			wantError: true,
			errorMsg:  "invalid variable name",
		},
		{
			name:      "set with specific environment",
			args:      []string{"set", "API_KEY=secret", "--env", "production"},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "production environment")
			},
		},
		{
			name:      "set without encryption",
			args:      []string{"set", "LOG_LEVEL=debug", "--encrypt=false"},
			wantError: false,
		},
		{
			name:      "empty value allowed",
			args:      []string{"set", "EMPTY_VAR="},
			wantError: false,
		},
		{
			name:      "value with equals sign",
			args:      []string{"set", "DATABASE_URL=postgres://user:pass@host/db?ssl=true"},
			wantError: false,
		},
		{
			name:      "no arguments provided",
			args:      []string{"set"},
			wantError: true,
			errorMsg:  "requires at least 1 arg(s), only received 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test environment
			env := test.NewTestEnvironment(t)
			defer env.Cleanup()

			// Setup
			if tt.setup != nil {
				tt.setup(t, env)
			}

			// Create command
			rootCmd := NewRootCommand()
			output, err := test.ExecuteCommand(rootCmd, tt.args...)

			// Check error
			if tt.wantError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify
			if tt.verify != nil {
				tt.verify(t, env, output)
			}
		})
	}
}

func TestParseVariables(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		expected  map[string]string
		wantError bool
	}{
		{
			name:  "single variable",
			input: []string{"KEY=value"},
			expected: map[string]string{
				"KEY": "value",
			},
		},
		{
			name:  "multiple variables",
			input: []string{"KEY1=value1", "KEY2=value2"},
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name:  "value with equals sign",
			input: []string{"DATABASE_URL=postgres://user:pass@host/db?ssl=true"},
			expected: map[string]string{
				"DATABASE_URL": "postgres://user:pass@host/db?ssl=true",
			},
		},
		{
			name:  "empty value",
			input: []string{"EMPTY="},
			expected: map[string]string{
				"EMPTY": "",
			},
		},
		{
			name:      "invalid format",
			input:     []string{"INVALID"},
			wantError: true,
		},
		{
			name:      "empty variable name",
			input:     []string{"=value"},
			wantError: true,
		},
		{
			name:      "variable with spaces in name",
			input:     []string{"INVALID VAR=value"},
			wantError: true,
		},
		{
			name:  "underscore in name",
			input: []string{"VALID_VAR=value"},
			expected: map[string]string{
				"VALID_VAR": "value",
			},
		},
		{
			name:  "starting with underscore",
			input: []string{"_VALID=value"},
			expected: map[string]string{
				"_VALID": "value",
			},
		},
		{
			name:      "starting with number",
			input:     []string{"1INVALID=value"},
			wantError: true,
		},
		{
			name:  "mixed case",
			input: []string{"ValidVar=value"},
			expected: map[string]string{
				"ValidVar": "value",
			},
		},
		{
			name:  "value with spaces",
			input: []string{"KEY=value with spaces"},
			expected: map[string]string{
				"KEY": "value with spaces",
			},
		},
		{
			name:  "value with special characters",
			input: []string{"KEY=!@#$%^&*()"},
			expected: map[string]string{
				"KEY": "!@#$%^&*()",
			},
		},
		{
			name:  "json value",
			input: []string{`CONFIG={"key":"value","nested":{"foo":"bar"}}`},
			expected: map[string]string{
				"CONFIG": `{"key":"value","nested":{"foo":"bar"}}`,
			},
		},
		{
			name:  "base64 value",
			input: []string{"SECRET=SGVsbG8gV29ybGQh"},
			expected: map[string]string{
				"SECRET": "SGVsbG8gV29ybGQh",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseVariables(tt.input)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsValidVariableName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty", "", false},
		{"simple", "VAR", true},
		{"with underscore", "VAR_NAME", true},
		{"with numbers", "VAR123", true},
		{"starting with underscore", "_VAR", true},
		{"starting with number", "1VAR", false},
		{"with hyphen", "VAR-NAME", false},
		{"with space", "VAR NAME", false},
		{"with dot", "VAR.NAME", false},
		{"lowercase", "var", true},
		{"mixed case", "VarName", true},
		{"all numbers after letter", "V123", true},
		{"single underscore", "_", true},
		{"multiple underscores", "__VAR__", true},
		{"with special char", "VAR@", false},
		{"unicode", "VAR_名前", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidVariableName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSetCommandWithExistingVariables tests the overwrite confirmation flow
func TestSetCommandWithExistingVariables(t *testing.T) {
	t.Run("force flag overwrites without confirmation", func(t *testing.T) {
		env := test.NewTestEnvironment(t)
		defer env.Cleanup()

		// Pre-populate storage
		env.Storage.Set("EXISTING_VAR", "old_value", false)

		rootCmd := NewRootCommand()
		output, err := test.ExecuteCommand(rootCmd, "set", "EXISTING_VAR=new_value", "--force")

		require.NoError(t, err)
		assert.Contains(t, output.Stdout, "Variables set successfully")

		// Verify the value was updated
		value, err := env.Storage.Get("EXISTING_VAR")
		require.NoError(t, err)
		assert.Equal(t, "new_value", value)
	})
}

// Benchmark to ensure performance
func BenchmarkParseVariables(b *testing.B) {
	input := []string{
		"VAR1=value1",
		"VAR2=value2",
		"VAR3=value3",
		"DATABASE_URL=postgres://localhost/db",
		"API_KEY=secret-key-value",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseVariables(input)
	}
}

func BenchmarkIsValidVariableName(b *testing.B) {
	names := []string{
		"VALID_VAR",
		"_UNDERSCORE",
		"VAR123",
		"MIXED_Case_123",
		"LONG_VARIABLE_NAME_WITH_MANY_UNDERSCORES",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, name := range names {
			_ = isValidVariableName(name)
		}
	}
}

func BenchmarkSetCommand(b *testing.B) {
	// Disable color output during benchmarks
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create fresh command for each iteration
		rootCmd := NewRootCommand()
		
		// Capture and discard output
		buf := &bytes.Buffer{}
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		
		rootCmd.SetArgs([]string{"set", "BENCH_VAR=value", "--force", "--no-color"})
		_ = rootCmd.Execute()
	}
}

// TestSetCommandErrorHandling tests various error scenarios
// TODO: Implement error injection in storage backend to test error cases
func TestSetCommandErrorHandling(t *testing.T) {
	t.Skip("Skipping error handling tests - need mock backend with error injection")
	
	// These tests would require a mock storage backend that can simulate errors
	// For now, the memory backend always succeeds, so these tests would fail
	/*
	tests := []struct {
		name        string
		args        []string
		setupError  string
		wantError   bool
		errorMsg    string
	}{
		{
			name:      "storage initialization failure",
			args:      []string{"set", "VAR=value"},
			wantError: true,
			errorMsg:  "failed to initialize storage",
		},
		{
			name:      "variable check failure",
			args:      []string{"set", "VAR=value"},
			wantError: true,
			errorMsg:  "failed to check variable",
		},
		{
			name:      "set operation failure",
			args:      []string{"set", "VAR=value"},
			wantError: true,
			errorMsg:  "failed to set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := test.NewTestEnvironment(t)
			defer env.Cleanup()

			// Create command and execute
			rootCmd := NewRootCommand()
			output, err := test.ExecuteCommand(rootCmd, tt.args...)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}

			// Additional error handling verification
			if err != nil {
				// Ensure error output is properly formatted
				assert.NotEmpty(t, output.Stderr)
			}
		})
	}
	*/
}

// TestSetCommandConcurrency tests concurrent set operations
func TestSetCommandConcurrency(t *testing.T) {
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	// Number of concurrent operations
	numOps := 10
	done := make(chan bool, numOps)
	errors := make(chan error, numOps)

	// Run concurrent set operations
	for i := 0; i < numOps; i++ {
		go func(index int) {
			rootCmd := NewRootCommand()
			varName := fmt.Sprintf("CONCURRENT_VAR_%d", index)
			varValue := fmt.Sprintf("value_%d", index)

			_, err := test.ExecuteCommand(rootCmd, "set", fmt.Sprintf("%s=%s", varName, varValue), "--force")
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < numOps; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Verify all variables were set
	for i := 0; i < numOps; i++ {
		varName := fmt.Sprintf("CONCURRENT_VAR_%d", i)
		expectedValue := fmt.Sprintf("value_%d", i)

		value, err := env.Storage.Get(varName)
		require.NoError(t, err)
		assert.Equal(t, expectedValue, value)
	}
}

// TestSetCommandValidation tests input validation edge cases
func TestSetCommandValidation(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "very long variable name",
			args:      []string{"set", strings.Repeat("A", 1000) + "=value"},
			wantError: false, // Should be valid
		},
		{
			name:      "very long value",
			args:      []string{"set", "VAR=" + strings.Repeat("x", 10000)},
			wantError: false, // Should be valid
		},
		{
			name:      "multiple equals in value",
			args:      []string{"set", "URL=https://example.com?foo=bar&baz=qux"},
			wantError: false,
		},
		{
			name:      "newline in value",
			args:      []string{"set", "MULTILINE=line1\nline2\nline3"},
			wantError: false,
		},
		{
			name:      "tab in value",
			args:      []string{"set", "TABBED=col1\tcol2\tcol3"},
			wantError: false,
		},
		{
			name:      "null byte in value",
			args:      []string{"set", "NULL=value\x00null"},
			wantError: false, // Go strings can contain null bytes
		},
		{
			name:      "unicode in value",
			args:      []string{"set", "UNICODE=Hello 世界 🌍"},
			wantError: false,
		},
		{
			name:      "reserved variable name PATH",
			args:      []string{"set", "PATH=/custom/path"},
			wantError: false, // We allow setting any valid name
		},
		{
			name:      "single character name",
			args:      []string{"set", "X=value"},
			wantError: false,
		},
		{
			name:      "only underscores",
			args:      []string{"set", "___=value"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := test.NewTestEnvironment(t)
			defer env.Cleanup()

			rootCmd := NewRootCommand()
			output, err := test.ExecuteCommand(rootCmd, tt.args...)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Contains(t, output.Stdout, "Variables set successfully")
			}
		})
	}
}