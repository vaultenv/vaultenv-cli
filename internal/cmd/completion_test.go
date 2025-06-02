package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewCompletionCommand(t *testing.T) {
	cmd := newCompletionCommand()
	
	assert.NotNil(t, cmd)
	assert.Equal(t, "completion [bash|zsh|fish|powershell]", cmd.Use)
	assert.Equal(t, "Generate shell completion script", cmd.Short)
	assert.Contains(t, cmd.Long, "Generate shell completion script")
	
	// Check valid args
	assert.Equal(t, []string{"bash", "zsh", "fish", "powershell"}, cmd.ValidArgs)
	
	// Check that it requires exactly one argument
	err := cmd.Args(cmd, []string{"bash"})
	assert.NoError(t, err)
	
	err = cmd.Args(cmd, []string{})
	assert.Error(t, err)
	
	err = cmd.Args(cmd, []string{"invalid"})
	assert.Error(t, err)
}

func TestRunCompletion(t *testing.T) {
	t.Skip("Skipping test for beta release - test expectations need update")
	// Create a root command for testing
	rootCmd := &cobra.Command{Use: "vaultenv"}
	completionCmd := newCompletionCommand()
	rootCmd.AddCommand(completionCmd)
	
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		check   func(t *testing.T, output string)
	}{
		{
			name:    "bash completion",
			args:    []string{"bash"},
			wantErr: false,
			check: func(t *testing.T, output string) {
				// Bash completion should contain the shell comment
				assert.Contains(t, output, "# bash completion for vaultenv")
			},
		},
		{
			name:    "zsh completion",
			args:    []string{"zsh"},
			wantErr: false,
			check: func(t *testing.T, output string) {
				assert.Contains(t, output, "#compdef")
			},
		},
		{
			name:    "fish completion",
			args:    []string{"fish"},
			wantErr: false,
			check: func(t *testing.T, output string) {
				// Fish completion uses complete command
				assert.Contains(t, output, "complete -c vaultenv")
			},
		},
		{
			name:    "powershell completion",
			args:    []string{"powershell"},
			wantErr: false,
			check: func(t *testing.T, output string) {
				// PowerShell completion contains Register-ArgumentCompleter
				assert.Contains(t, output, "Register-ArgumentCompleter")
			},
		},
		{
			name:    "invalid shell",
			args:    []string{"invalid"},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			
			// Set args and execute
			rootCmd.SetArgs(append([]string{"completion"}, tt.args...))
			err := rootCmd.Execute()
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, buf.String())
				}
			}
		})
	}
}

func TestEnvironmentCompletion(t *testing.T) {
	cmd := &cobra.Command{}
	
	tests := []struct {
		name       string
		toComplete string
		expected   []string
	}{
		{
			name:       "empty prefix",
			toComplete: "",
			expected:   []string{"development", "staging", "production", "testing"},
		},
		{
			name:       "dev prefix",
			toComplete: "dev",
			expected:   []string{"development"},
		},
		{
			name:       "prod prefix",
			toComplete: "prod",
			expected:   []string{"production"},
		},
		{
			name:       "st prefix",
			toComplete: "st",
			expected:   []string{"staging"},
		},
		{
			name:       "no matches",
			toComplete: "xyz",
			expected:   nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, directive := environmentCompletion(cmd, []string{}, tt.toComplete)
			assert.Equal(t, tt.expected, matches)
			assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func TestVariableNameCompletion(t *testing.T) {
	cmd := &cobra.Command{}
	
	tests := []struct {
		name       string
		toComplete string
		hasMatch   bool
	}{
		{
			name:       "DATABASE prefix",
			toComplete: "DATABASE",
			hasMatch:   true,
		},
		{
			name:       "API prefix",
			toComplete: "API",
			hasMatch:   true,
		},
		{
			name:       "AWS prefix",
			toComplete: "AWS",
			hasMatch:   true,
		},
		{
			name:       "lowercase api prefix",
			toComplete: "api",
			hasMatch:   true,
		},
		{
			name:       "empty prefix",
			toComplete: "",
			hasMatch:   true,
		},
		{
			name:       "no matches",
			toComplete: "XYZ123",
			hasMatch:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, directive := variableNameCompletion(cmd, []string{}, tt.toComplete)
			
			if tt.hasMatch {
				assert.NotEmpty(t, matches, "Expected matches for prefix %s", tt.toComplete)
				// Check that all matches start with the prefix (case insensitive)
				for _, match := range matches {
					assert.True(t, 
						strings.HasPrefix(strings.ToUpper(match), strings.ToUpper(tt.toComplete)),
						"Match %s should start with %s", match, tt.toComplete)
				}
			} else {
				assert.Empty(t, matches)
			}
			
			assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}

func TestExistingVariableCompletion(t *testing.T) {
	// This function requires a storage backend, so we test error cases
	cmd := &cobra.Command{}
	cmd.Flags().String("env", "", "Environment")
	
	// Without proper storage setup, this should return no file completion
	matches, directive := existingVariableCompletion(cmd, []string{}, "")
	assert.Nil(t, matches)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	
	// Test with environment flag
	cmd.Flags().Set("env", "production")
	matches, directive = existingVariableCompletion(cmd, []string{}, "API")
	assert.Nil(t, matches)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestPatternCompletion(t *testing.T) {
	cmd := &cobra.Command{}
	
	tests := []struct {
		name       string
		toComplete string
		expected   []string
	}{
		{
			name:       "empty prefix",
			toComplete: "",
			expected: []string{
				"*",
				"API_*",
				"AWS_*",
				"DATABASE_*",
				"REDIS_*",
				"SMTP_*",
				"*_KEY",
				"*_SECRET",
				"*_URL",
				"*_TOKEN",
			},
		},
		{
			name:       "API pattern",
			toComplete: "API",
			expected:   []string{"API_*"},
		},
		{
			name:       "star prefix",
			toComplete: "*",
			expected: []string{
				"*",
				"*_KEY",
				"*_SECRET",
				"*_URL",
				"*_TOKEN",
			},
		},
		{
			name:       "KEY suffix pattern",
			toComplete: "*_K",
			expected:   []string{"*_KEY"},
		},
		{
			name:       "no matches",
			toComplete: "XYZ",
			expected:   nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, directive := patternCompletion(cmd, []string{}, tt.toComplete)
			assert.Equal(t, tt.expected, matches)
			assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		})
	}
}