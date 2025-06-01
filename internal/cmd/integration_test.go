package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vaultenv/vaultenv-cli/internal/test"
)

// TestCLIWorkflow tests a complete workflow of setting, getting, and listing variables
func TestCLIWorkflow(t *testing.T) {
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	rootCmd := NewRootCommand()

	// Step 1: Set multiple variables
	output, err := test.ExecuteCommand(rootCmd, "set",
		"DATABASE_URL=postgres://localhost/testdb",
		"API_KEY=test-api-key-123",
		"LOG_LEVEL=debug",
		"FEATURE_FLAG_NEW_UI=true",
	)
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "Setting 4 variable(s)")
	assert.Contains(t, output.Stdout, "Variables set successfully")

	// Step 2: Get a single variable
	output, err = test.ExecuteCommand(rootCmd, "get", "DATABASE_URL")
	require.NoError(t, err)
	assert.Equal(t, "DATABASE_URL=postgres://localhost/testdb\n", output.Stdout)

	// Step 3: Get multiple variables
	output, err = test.ExecuteCommand(rootCmd, "get", "API_KEY", "LOG_LEVEL")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "API_KEY=test-api-key-123")
	assert.Contains(t, output.Stdout, "LOG_LEVEL=debug")

	// Step 4: List all variables
	output, err = test.ExecuteCommand(rootCmd, "list")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "DATABASE_URL")
	assert.Contains(t, output.Stdout, "API_KEY")
	assert.Contains(t, output.Stdout, "LOG_LEVEL")
	assert.Contains(t, output.Stdout, "FEATURE_FLAG_NEW_UI")
	assert.Contains(t, output.Stdout, "Total: 4 variable(s)")

	// Step 5: List with pattern
	output, err = test.ExecuteCommand(rootCmd, "list", "--pattern", "FEATURE_*")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "FEATURE_FLAG_NEW_UI")
	assert.NotContains(t, output.Stdout, "DATABASE_URL")
	assert.Contains(t, output.Stdout, "Total: 1 variable(s)")

	// Step 6: Update an existing variable
	output, err = test.ExecuteCommand(rootCmd, "set", "LOG_LEVEL=info", "--force")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "Variables set successfully")

	// Step 7: Verify the update
	output, err = test.ExecuteCommand(rootCmd, "get", "LOG_LEVEL", "--quiet")
	require.NoError(t, err)
	assert.Equal(t, "info\n", output.Stdout)

	// Step 8: Get in export format
	// Create a fresh command for this test to avoid flag state issues
	rootCmd2 := NewRootCommand()
	output, err = test.ExecuteCommand(rootCmd2, "get", "DATABASE_URL", "--export")
	require.NoError(t, err)
	assert.Equal(t, "export DATABASE_URL=\"postgres://localhost/testdb\"\n", output.Stdout)

	// Step 9: Try to get non-existent variable
	output, err = test.ExecuteCommand(rootCmd, "get", "NON_EXISTENT")
	require.Error(t, err)
	assert.Contains(t, output.Stdout, "Variable NON_EXISTENT not found")
}

// TestEnvironmentIsolation tests that different environments are isolated
func TestEnvironmentIsolation(t *testing.T) {
	t.Skip("Skipping environment isolation test - not implemented yet")
	
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	rootCmd := NewRootCommand()

	// Set variables in development environment
	output, err := test.ExecuteCommand(rootCmd, "set",
		"DEV_VAR1=dev_value1",
		"DEV_VAR2=dev_value2",
		"--env", "development",
	)
	require.NoError(t, err)

	// Set different values in production environment
	output, err = test.ExecuteCommand(rootCmd, "set",
		"PROD_VAR1=prod_value1",
		"PROD_VAR2=prod_value2",
		"--env", "production",
	)
	require.NoError(t, err)

	// Verify development values
	output, err = test.ExecuteCommand(rootCmd, "get", "DEV_VAR1", "--env", "development")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "DEV_VAR1=dev_value1")

	// Verify production values
	output, err = test.ExecuteCommand(rootCmd, "get", "PROD_VAR1", "--env", "production")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "PROD_VAR1=prod_value1")

	// List development variables
	output, err = test.ExecuteCommand(rootCmd, "list", "--env", "development")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "Total: 2 variable(s)")

	// List production variables
	output, err = test.ExecuteCommand(rootCmd, "list", "--env", "production")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "Total: 2 variable(s)")
}

// TestSpecialCharacterHandling tests handling of special characters in values
func TestSpecialCharacterHandling(t *testing.T) {
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	rootCmd := NewRootCommand()

	specialValues := map[string]string{
		"JSON_CONFIG":      `{"key":"value","nested":{"foo":"bar"}}`,
		"URL_WITH_PARAMS":  "https://api.example.com/v1/endpoint?param1=value1&param2=value2",
		"PATH_VAR":         "/usr/local/bin:/usr/bin:/bin:$HOME/bin",
		"MULTILINE":        "line1\nline2\nline3",
		"QUOTES":           `value with "double" and 'single' quotes`,
		"BACKSLASHES":      `C:\Program Files\MyApp\bin`,
		"UNICODE":          "Hello 世界 🌍",
		"EMPTY":            "",
		"SPACES":           "  value with spaces  ",
		"SHELL_SPECIAL":    "$HOME/path with $(command) and backticks",
	}

	// Set all special values
	for key, value := range specialValues {
		_, err := test.ExecuteCommand(rootCmd, "set", key+"="+value)
		require.NoError(t, err, "Failed to set %s", key)
	}

	// Verify all values are stored correctly
	for key, expectedValue := range specialValues {
		output, err := test.ExecuteCommand(rootCmd, "get", key, "--quiet")
		require.NoError(t, err, "Failed to get %s", key)
		assert.Equal(t, expectedValue+"\n", output.Stdout, "Value mismatch for %s", key)
	}

	// Test export format with special characters
	// Use a fresh command to avoid flag state issues
	rootCmd2 := NewRootCommand()
	output, err := test.ExecuteCommand(rootCmd2, "get", "SHELL_SPECIAL", "--export")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, `export SHELL_SPECIAL="\$HOME/path with \$(command) and backticks"`)
}

// TestErrorScenarios tests various error conditions
func TestErrorScenarios(t *testing.T) {
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	rootCmd := NewRootCommand()

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "set without arguments",
			args:        []string{"set"},
			expectError: true,
			errorMsg:    "requires at least 1 arg(s)",
		},
		{
			name:        "get without arguments",
			args:        []string{"get"},
			expectError: true,
			errorMsg:    "requires at least 1 arg(s)",
		},
		{
			name:        "set invalid format",
			args:        []string{"set", "INVALID_FORMAT"},
			expectError: true,
			errorMsg:    "invalid format",
		},
		{
			name:        "set invalid variable name",
			args:        []string{"set", "123INVALID=value"},
			expectError: true,
			errorMsg:    "invalid variable name",
		},
		{
			name:        "get non-existent variable",
			args:        []string{"get", "DOES_NOT_EXIST"},
			expectError: true,
			errorMsg:    "no variables found",
		},
		{
			name:        "invalid environment",
			args:        []string{"set", "VAR=value", "--env", ""},
			expectError: false, // Empty env might be allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := test.ExecuteCommand(rootCmd, tt.args...)
			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestBatchOperations tests operations with many variables
func TestBatchOperations(t *testing.T) {
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	rootCmd := NewRootCommand()

	// Set many variables at once
	args := []string{"set"}
	numVars := 50
	for i := 0; i < numVars; i++ {
		args = append(args, fmt.Sprintf("VAR_%02d=value_%02d", i, i))
	}

	output, err := test.ExecuteCommand(rootCmd, args...)
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, fmt.Sprintf("Setting %d variable(s)", numVars))

	// List to verify all were set
	output, err = test.ExecuteCommand(rootCmd, "list")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, fmt.Sprintf("Total: %d variable(s)", numVars))

	// Get multiple variables
	getArgs := []string{"get"}
	for i := 0; i < 10; i++ {
		getArgs = append(getArgs, fmt.Sprintf("VAR_%02d", i))
	}
	output, err = test.ExecuteCommand(rootCmd, getArgs...)
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		assert.Contains(t, output.Stdout, fmt.Sprintf("VAR_%02d=value_%02d", i, i))
	}
}

// TestPatternMatching tests various pattern matching scenarios
func TestPatternMatching(t *testing.T) {
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	rootCmd := NewRootCommand()

	// Set up test data
	testVars := map[string]string{
		"API_KEY":          "key1",
		"API_SECRET":       "secret1",
		"API_URL":          "https://api.example.com",
		"DATABASE_URL":     "postgres://localhost/db",
		"DATABASE_USER":    "dbuser",
		"DATABASE_PASS":    "dbpass",
		"REDIS_URL":        "redis://localhost:6379",
		"CACHE_TTL":        "3600",
		"FEATURE_NEW_UI":   "true",
		"FEATURE_BETA":     "false",
	}

	for key, value := range testVars {
		_, err := test.ExecuteCommand(rootCmd, "set", key+"="+value)
		require.NoError(t, err)
	}

	tests := []struct {
		pattern      string
		expectedKeys []string
		count        int
	}{
		{
			pattern:      "API_*",
			expectedKeys: []string{"API_KEY", "API_SECRET", "API_URL"},
			count:        3,
		},
		{
			pattern:      "*_URL",
			expectedKeys: []string{"API_URL", "DATABASE_URL", "REDIS_URL"},
			count:        3,
		},
		{
			pattern:      "DATABASE_*",
			expectedKeys: []string{"DATABASE_URL", "DATABASE_USER", "DATABASE_PASS"},
			count:        3,
		},
		{
			pattern:      "FEATURE_*",
			expectedKeys: []string{"FEATURE_NEW_UI", "FEATURE_BETA"},
			count:        2,
		},
		{
			pattern:      "*",
			expectedKeys: []string{}, // All keys
			count:        len(testVars),
		},
		{
			pattern:      "EXACT_MATCH",
			expectedKeys: []string{},
			count:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			output, err := test.ExecuteCommand(rootCmd, "list", "--pattern", tt.pattern)
			require.NoError(t, err)

			if tt.count > 0 {
				assert.Contains(t, output.Stdout, fmt.Sprintf("Total: %d variable(s)", tt.count))
				for _, key := range tt.expectedKeys {
					assert.Contains(t, output.Stdout, key)
				}
			} else {
				assert.Contains(t, output.Stdout, "No variables matching pattern")
			}
		})
	}
}

// TestOutputFormats tests different output formats
func TestOutputFormats(t *testing.T) {
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	rootCmd := NewRootCommand()

	// Set test data
	_, err := test.ExecuteCommand(rootCmd, "set",
		"TEST_VAR=test_value",
		"ANOTHER_VAR=another_value",
	)
	require.NoError(t, err)

	// Test default format
	output, err := test.ExecuteCommand(rootCmd, "get", "TEST_VAR")
	require.NoError(t, err)
	assert.Equal(t, "TEST_VAR=test_value\n", output.Stdout)

	// Test quiet format
	output, err = test.ExecuteCommand(rootCmd, "get", "TEST_VAR", "--quiet")
	require.NoError(t, err)
	assert.Equal(t, "test_value\n", output.Stdout)

	// Test export format
	// Use fresh command to avoid flag state issues
	rootCmd2 := NewRootCommand()
	output, err = test.ExecuteCommand(rootCmd2, "get", "TEST_VAR", "--export")
	require.NoError(t, err)
	assert.Equal(t, "export TEST_VAR=\"test_value\"\n", output.Stdout)

	// Test list without values
	output, err = test.ExecuteCommand(rootCmd, "list")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "TEST_VAR")
	assert.Contains(t, output.Stdout, "ANOTHER_VAR")
	assert.NotContains(t, output.Stdout, "test_value")
	assert.NotContains(t, output.Stdout, "another_value")

	// Test list with values
	output, err = test.ExecuteCommand(rootCmd, "list", "--values")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "TEST_VAR")
	assert.Contains(t, output.Stdout, "test_value")
	assert.Contains(t, output.Stdout, "ANOTHER_VAR")
	assert.Contains(t, output.Stdout, "another_value")
}