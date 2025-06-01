package cmd

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vaultenv/vaultenv-cli/internal/test"
)

func TestGetCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(t *testing.T, env *test.TestEnvironment)
		wantError bool
		errorMsg  string
		verify    func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput)
	}{
		{
			name: "get single variable",
			args: []string{"get", "TEST_VAR"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("TEST_VAR", "test_value", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "TEST_VAR=test_value")
			},
		},
		{
			name: "get multiple variables",
			args: []string{"get", "VAR1", "VAR2"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("VAR1", "value1", false)
				env.Storage.Set("VAR2", "value2", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "VAR1=value1")
				assert.Contains(t, output.Stdout, "VAR2=value2")
			},
		},
		{
			name:      "get non-existent variable",
			args:      []string{"get", "NON_EXISTENT"},
			wantError: true,
			errorMsg:  "no variables found",
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "Variable NON_EXISTENT not found")
			},
		},
		{
			name: "get with quiet flag",
			args: []string{"get", "TEST_VAR", "--quiet"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("TEST_VAR", "test_value", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Equal(t, "test_value\n", output.Stdout)
			},
		},
		{
			name: "get with export flag",
			args: []string{"get", "TEST_VAR", "--export"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("TEST_VAR", "test_value", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Equal(t, "export TEST_VAR=\"test_value\"\n", output.Stdout)
			},
		},
		{
			name: "get with specific environment",
			args: []string{"get", "API_KEY", "--env", "production"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("API_KEY", "prod-key", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "API_KEY=prod-key")
			},
		},
		{
			name:      "no arguments provided",
			args:      []string{"get"},
			wantError: true,
			errorMsg:  "requires at least 1 arg(s), only received 0",
		},
		{
			name: "mix of existing and non-existing variables",
			args: []string{"get", "EXISTING", "NON_EXISTING", "ANOTHER_EXISTING"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("EXISTING", "value1", false)
				env.Storage.Set("ANOTHER_EXISTING", "value2", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "EXISTING=value1")
				assert.Contains(t, output.Stdout, "Variable NON_EXISTING not found")
				assert.Contains(t, output.Stdout, "ANOTHER_EXISTING=value2")
			},
		},
		{
			name: "get value with special characters",
			args: []string{"get", "SPECIAL_VAR"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("SPECIAL_VAR", "value with $pecial \"characters\" and `backticks`", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "SPECIAL_VAR=value with $pecial \"characters\" and `backticks`")
			},
		},
		{
			name: "export format with special characters",
			args: []string{"get", "SPECIAL_VAR", "--export"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("SPECIAL_VAR", "value with $pecial \"characters\" and `backticks`", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "export SPECIAL_VAR=\"value with \\$pecial \\\"characters\\\" and \\`backticks\\`\"")
			},
		},
		{
			name: "get empty value",
			args: []string{"get", "EMPTY_VAR"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("EMPTY_VAR", "", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Equal(t, "EMPTY_VAR=\n", output.Stdout)
			},
		},
		{
			name: "get multiline value",
			args: []string{"get", "MULTILINE"},
			setup: func(t *testing.T, env *test.TestEnvironment) {
				env.Storage.Set("MULTILINE", "line1\nline2\nline3", false)
			},
			wantError: false,
			verify: func(t *testing.T, env *test.TestEnvironment, output *test.CommandOutput) {
				assert.Contains(t, output.Stdout, "MULTILINE=line1\nline2\nline3")
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

func TestEscapeShellValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple value",
			input:    "simple",
			expected: "simple",
		},
		{
			name:     "with double quotes",
			input:    `value "with" quotes`,
			expected: `value \"with\" quotes`,
		},
		{
			name:     "with dollar sign",
			input:    "value with $VAR",
			expected: `value with \$VAR`,
		},
		{
			name:     "with backticks",
			input:    "value with `command`",
			expected: `value with \` + "`command\\`",
		},
		{
			name:     "with backslashes",
			input:    `value\with\backslashes`,
			expected: `value\\with\\backslashes`,
		},
		{
			name:     "complex value",
			input:    `complex "value" with $VAR and ` + "`command`" + ` and \backslash`,
			expected: `complex \"value\" with \$VAR and \` + "`command\\`" + ` and \\backslash`,
		},
		{
			name:     "empty value",
			input:    "",
			expected: "",
		},
		{
			name:     "newlines",
			input:    "line1\nline2",
			expected: "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeShellValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetCommandOutput tests different output formats
func TestGetCommandOutput(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		vars   map[string]string
		verify func(t *testing.T, output string)
	}{
		{
			name: "default format multiple variables",
			args: []string{"get", "VAR1", "VAR2", "VAR3"},
			vars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
				"VAR3": "value3",
			},
			verify: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				assert.Len(t, lines, 3)
				assert.Contains(t, output, "VAR1=value1")
				assert.Contains(t, output, "VAR2=value2")
				assert.Contains(t, output, "VAR3=value3")
			},
		},
		{
			name: "quiet format multiple variables",
			args: []string{"get", "VAR1", "VAR2", "--quiet"},
			vars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
			verify: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				assert.Len(t, lines, 2)
				assert.Equal(t, "value1", lines[0])
				assert.Equal(t, "value2", lines[1])
			},
		},
		{
			name: "export format multiple variables",
			args: []string{"get", "VAR1", "VAR2", "--export"},
			vars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
			verify: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				assert.Len(t, lines, 2)
				assert.Equal(t, `export VAR1="value1"`, lines[0])
				assert.Equal(t, `export VAR2="value2"`, lines[1])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := test.NewTestEnvironment(t)
			defer env.Cleanup()

			// Setup variables
			for key, value := range tt.vars {
				env.Storage.Set(key, value, false)
			}

			rootCmd := NewRootCommand()
			output, err := test.ExecuteCommand(rootCmd, tt.args...)
			require.NoError(t, err)

			tt.verify(t, output.Stdout)
		})
	}
}

// TestGetCommandPerformance tests performance with large values
func TestGetCommandPerformance(t *testing.T) {
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	// Create a large value (1MB)
	largeValue := strings.Repeat("x", 1024*1024)
	env.Storage.Set("LARGE_VAR", largeValue, false)

	rootCmd := NewRootCommand()
	output, err := test.ExecuteCommand(rootCmd, "get", "LARGE_VAR", "--quiet")

	require.NoError(t, err)
	assert.Equal(t, largeValue+"\n", output.Stdout)
}

// BenchmarkGetCommand benchmarks the get command
func BenchmarkGetCommand(b *testing.B) {
	env := test.NewTestEnvironment(&testing.T{})
	defer env.Cleanup()

	// Pre-populate with test data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("VAR_%d", i)
		value := fmt.Sprintf("value_%d", i)
		env.Storage.Set(key, value, false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rootCmd := NewRootCommand()
		rootCmd.SetOut(io.Discard)
		rootCmd.SetErr(io.Discard)
		rootCmd.SetArgs([]string{"get", "VAR_50", "--quiet"})
		_ = rootCmd.Execute()
	}
}

// BenchmarkGetMultipleVariables benchmarks getting multiple variables
func BenchmarkGetMultipleVariables(b *testing.B) {
	env := test.NewTestEnvironment(&testing.T{})
	defer env.Cleanup()

	// Pre-populate with test data
	vars := []string{"VAR_1", "VAR_2", "VAR_3", "VAR_4", "VAR_5"}
	for _, v := range vars {
		env.Storage.Set(v, "test_value", false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rootCmd := NewRootCommand()
		rootCmd.SetOut(io.Discard)
		rootCmd.SetErr(io.Discard)
		args := append([]string{"get"}, vars...)
		rootCmd.SetArgs(args)
		_ = rootCmd.Execute()
	}
}

// BenchmarkEscapeShellValue benchmarks shell value escaping
func BenchmarkEscapeShellValue(b *testing.B) {
	testValue := `complex "value" with $VAR and ` + "`command`" + ` and \backslash`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = escapeShellValue(testValue)
	}
}

// TestGetCommandConcurrency tests concurrent get operations
func TestGetCommandConcurrency(t *testing.T) {
	env := test.NewTestEnvironment(t)
	defer env.Cleanup()

	// Pre-populate storage
	numVars := 50
	for i := 0; i < numVars; i++ {
		key := fmt.Sprintf("CONCURRENT_VAR_%d", i)
		value := fmt.Sprintf("value_%d", i)
		env.Storage.Set(key, value, false)
	}

	// Number of concurrent operations
	numOps := 20
	done := make(chan bool, numOps)
	errors := make(chan error, numOps)

	// Run concurrent get operations
	for i := 0; i < numOps; i++ {
		go func(index int) {
			rootCmd := NewRootCommand()
			varName := fmt.Sprintf("CONCURRENT_VAR_%d", index%numVars)

			output, err := test.ExecuteCommand(rootCmd, "get", varName, "--quiet")
			if err != nil {
				errors <- err
			} else {
				expectedValue := fmt.Sprintf("value_%d\n", index%numVars)
				if output.Stdout != expectedValue {
					errors <- fmt.Errorf("unexpected value: got %q, want %q", output.Stdout, expectedValue)
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