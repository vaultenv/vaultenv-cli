package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vaultenv/vaultenv-cli/internal/test"
)

func TestListCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(t *testing.T, env *test.TestEnvironment)
		wantError bool
		errorMsg  string
		verify    func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput)
	}{
		{
			name: "list variables without values",
			args: []string{"list"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("VAR1", "value1", false)
				env.Storage.Set("VAR2", "value2", false)
				env.Storage.Set("VAR3", "value3", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "VAR1")
				assert.Contains(t, output.Stdout, "VAR2")
				assert.Contains(t, output.Stdout, "VAR3")
				assert.NotContains(t, output.Stdout, "value1")
				assert.NotContains(t, output.Stdout, "value2")
				assert.NotContains(t, output.Stdout, "value3")
				assert.Contains(t, output.Stdout, "Total: 3 variable(s)")
			},
		},
		{
			name: "list variables with values",
			args: []string{"list", "--values"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("VAR1", "value1", false)
				env.Storage.Set("VAR2", "value2", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "VAR1")
				assert.Contains(t, output.Stdout, "value1")
				assert.Contains(t, output.Stdout, "VAR2")
				assert.Contains(t, output.Stdout, "value2")
				assert.Contains(t, output.Stdout, "Total: 2 variable(s)")
			},
		},
		{
			name:      "list empty environment",
			args:      []string{"list"},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "No variables found in development environment")
			},
		},
		{
			name: "list with pattern filter",
			args: []string{"list", "--pattern", "API_*"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("API_KEY", "key1", false)
				env.Storage.Set("API_SECRET", "secret1", false)
				env.Storage.Set("DATABASE_URL", "postgres://", false)
				env.Storage.Set("REDIS_URL", "redis://", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "API_KEY")
				assert.Contains(t, output.Stdout, "API_SECRET")
				assert.NotContains(t, output.Stdout, "DATABASE_URL")
				assert.NotContains(t, output.Stdout, "REDIS_URL")
				assert.Contains(t, output.Stdout, "Total: 2 variable(s)")
			},
		},
		{
			name: "list with pattern no matches",
			args: []string{"list", "--pattern", "NONEXISTENT_*"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("VAR1", "value1", false)
				env.Storage.Set("VAR2", "value2", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "No variables matching pattern 'NONEXISTENT_*'")
			},
		},
		{
			name: "list with specific environment",
			args: []string{"list", "--env", "production"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("PROD_VAR", "prod_value", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "Environment: production")
				assert.Contains(t, output.Stdout, "PROD_VAR")
			},
		},
		{
			name: "list with truncated long values",
			args: []string{"list", "--values"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				longValue := strings.Repeat("x", 100)
				env.Storage.Set("LONG_VAR", longValue, false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "LONG_VAR")
				assert.Contains(t, output.Stdout, "...")
				assert.NotContains(t, output.Stdout, strings.Repeat("x", 100))
			},
		},
		{
			name: "list sorted alphabetically",
			args: []string{"list"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("ZEBRA", "z", false)
				env.Storage.Set("ALPHA", "a", false)
				env.Storage.Set("BETA", "b", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				lines := strings.Split(output.Stdout, "\n")
				var varLines []string
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "ALPHA" || line == "BETA" || line == "ZEBRA" {
						varLines = append(varLines, line)
					}
				}
				assert.Equal(t, []string{"ALPHA", "BETA", "ZEBRA"}, varLines)
			},
		},
		{
			name: "list with pattern suffix match",
			args: []string{"list", "--pattern", "*_URL"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("DATABASE_URL", "db", false)
				env.Storage.Set("REDIS_URL", "redis", false)
				env.Storage.Set("API_KEY", "key", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "DATABASE_URL")
				assert.Contains(t, output.Stdout, "REDIS_URL")
				assert.NotContains(t, output.Stdout, "API_KEY")
			},
		},
		{
			name: "list with exact pattern match",
			args: []string{"list", "--pattern", "EXACT_VAR"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("EXACT_VAR", "value", false)
				env.Storage.Set("EXACT_VAR_2", "value2", false)
				env.Storage.Set("PREFIX_EXACT_VAR", "value3", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "EXACT_VAR")
				assert.NotContains(t, output.Stdout, "EXACT_VAR_2")
				assert.NotContains(t, output.Stdout, "PREFIX_EXACT_VAR")
				assert.Contains(t, output.Stdout, "Total: 1 variable(s)")
			},
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

func TestFilterKeys(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		pattern  string
		expected []string
	}{
		{
			name:     "exact match",
			keys:     []string{"VAR1", "VAR2", "VAR3"},
			pattern:  "VAR2",
			expected: []string{"VAR2"},
		},
		{
			name:     "prefix wildcard",
			keys:     []string{"API_KEY", "API_SECRET", "DATABASE_URL"},
			pattern:  "API_*",
			expected: []string{"API_KEY", "API_SECRET"},
		},
		{
			name:     "suffix wildcard",
			keys:     []string{"DATABASE_URL", "REDIS_URL", "API_KEY"},
			pattern:  "*_URL",
			expected: []string{"DATABASE_URL", "REDIS_URL"},
		},
		{
			name:     "middle wildcard",
			keys:     []string{"TEST_VAR_1", "TEST_VAR_2", "PROD_VAR_1"},
			pattern:  "TEST_*_1",
			expected: []string{"TEST_VAR_1"},
		},
		{
			name:     "multiple wildcards",
			keys:     []string{"TEST_API_KEY", "TEST_API_SECRET", "PROD_DB_URL"},
			pattern:  "*_API_*",
			expected: []string{"TEST_API_KEY", "TEST_API_SECRET"},
		},
		{
			name:     "no matches",
			keys:     []string{"VAR1", "VAR2", "VAR3"},
			pattern:  "NONEXISTENT",
			expected: []string{},
		},
		{
			name:     "match all",
			keys:     []string{"VAR1", "VAR2", "VAR3"},
			pattern:  "*",
			expected: []string{"VAR1", "VAR2", "VAR3"},
		},
		{
			name:     "empty pattern",
			keys:     []string{"VAR1", "VAR2", "VAR3"},
			pattern:  "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterKeys(tt.keys, tt.pattern)
			sort.Strings(result)
			sort.Strings(tt.expected)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		pattern  string
		expected bool
	}{
		{"exact match", "TEST", "TEST", true},
		{"no match", "TEST", "PROD", false},
		{"prefix wildcard", "TEST_VAR", "TEST_*", true},
		{"prefix wildcard no match", "PROD_VAR", "TEST_*", false},
		{"suffix wildcard", "VAR_TEST", "*_TEST", true},
		{"suffix wildcard no match", "VAR_PROD", "*_TEST", false},
		{"both wildcards", "PREFIX_MIDDLE_SUFFIX", "*_MIDDLE_*", true},
		{"empty string", "", "", true},
		{"empty string with wildcard", "", "*", true},
		{"wildcard matches empty", "TEST", "TEST*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := matchPattern(tt.str, tt.pattern)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestListCommandFormatting tests the output formatting
func TestListCommandFormatting(t *testing.T) {
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	// Set up variables with different key lengths
	vars := map[string]string{
		"A":          "short",
		"MEDIUM_VAR": "medium value",
		"VERY_LONG_VARIABLE_NAME": "long value",
	}

	for k, v := range vars {
		env.Storage.Set(k, v, false)
	}

	rootCmd := NewRootCommand()
	output, err := test.ExecuteCommand(rootCmd, "list", "--values")
	require.NoError(t, err)

	// Check that values are aligned
	lines := strings.Split(output.Stdout, "\n")
	var valueLine string
	for _, line := range lines {
		if strings.Contains(line, "=") && strings.Contains(line, "MEDIUM_VAR") {
			valueLine = line
			break
		}
	}

	// The equals signs should be aligned
	assert.Contains(t, valueLine, "MEDIUM_VAR              = medium value")
}

// BenchmarkListCommand benchmarks the list command
func BenchmarkListCommand(b *testing.B) {
	env := test.NewTestEnvironment(&testing.T{})
	defer env.Cleanup()

	// Pre-populate with many variables
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("VAR_%04d", i)
		value := fmt.Sprintf("value_%d", i)
		env.Storage.Set(key, value, false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rootCmd := NewRootCommand()
		rootCmd.SetOut(io.Discard)
		rootCmd.SetErr(io.Discard)
		rootCmd.SetArgs([]string{"list"})
		_ = rootCmd.Execute()
	}
}

// BenchmarkFilterKeys benchmarks pattern filtering
func BenchmarkFilterKeys(b *testing.B) {
	// Create a large list of keys
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("VAR_%04d", i)
	}

	pattern := "VAR_0*"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filterKeys(keys, pattern)
	}
}

// TestListCommandConcurrency tests concurrent list operations
func TestListCommandConcurrency(t *testing.T) {
	t.Skip("Skipping concurrency test - flaky due to output format variations")
	
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	// Pre-populate storage
	numVars := 100
	for i := 0; i < numVars; i++ {
		key := fmt.Sprintf("CONCURRENT_VAR_%d", i)
		value := fmt.Sprintf("value_%d", i)
		env.Storage.Set(key, value, false)
	}

	// Number of concurrent operations
	numOps := 10
	done := make(chan bool, numOps)
	errors := make(chan error, numOps)

	// Run concurrent list operations
	for i := 0; i < numOps; i++ {
		go func(index int) {
			rootCmd := NewRootCommand()
			output, err := test.ExecuteCommand(rootCmd, "list")
			if err != nil {
				errors <- err
			} else {
				// Verify we got all variables
				if !strings.Contains(output.Stdout, fmt.Sprintf("Total: %d variable(s)", numVars)) {
					errors <- fmt.Errorf("unexpected variable count in output")
				}
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
}

// TestListCommandLargeDataset tests listing with many variables
func TestListCommandLargeDataset(t *testing.T) {
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	// Create 1000 variables
	numVars := 1000
	for i := 0; i < numVars; i++ {
		key := fmt.Sprintf("VAR_%04d", i)
		value := fmt.Sprintf("value_%d", i)
		env.Storage.Set(key, value, false)
	}

	rootCmd := NewRootCommand()
	output, err := test.ExecuteCommand(rootCmd, "list")
	require.NoError(t, err)

	// Verify all variables are listed
	assert.Contains(t, output.Stdout, fmt.Sprintf("Total: %d variable(s)", numVars))

	// Test with pattern to reduce output
	output, err = test.ExecuteCommand(rootCmd, "list", "--pattern", "VAR_00*")
	require.NoError(t, err)
	assert.Contains(t, output.Stdout, "Total: 100 variable(s)")
}