package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddAliases(t *testing.T) {
	rootCmd := &cobra.Command{Use: "vaultenv"}

	// Add original commands
	originalCommands := map[string]*cobra.Command{
		"set":     {Use: "set", Short: "Set a variable"},
		"get":     {Use: "get", Short: "Get a variable"},
		"list":    {Use: "list", Short: "List variables"},
		"env":     {Use: "env", Short: "Environment management"},
		"export":  {Use: "export", Short: "Export variables"},
		"import":  {Use: "import", Short: "Import variables"},
		"push":    {Use: "push", Short: "Push to git"},
		"pull":    {Use: "pull", Short: "Pull from git"},
		"history": {Use: "history", Short: "Show history"},
		"audit":   {Use: "audit", Short: "Show audit log"},
		"config":  {Use: "config", Short: "Configuration management"},
	}

	for _, cmd := range originalCommands {
		rootCmd.AddCommand(cmd)
	}

	// Add aliases
	addAliases(rootCmd)

	// Expected aliases
	expectedAliases := map[string]string{
		"s":  "set",
		"g":  "get",
		"l":  "list",
		"e":  "env",
		"x":  "export",
		"i":  "import",
		"p":  "push",
		"pl": "pull",
		"h":  "history",
		"a":  "audit",
		"c":  "config",
	}

	// Check that all aliases were created
	for alias, original := range expectedAliases {
		t.Run("alias_"+alias, func(t *testing.T) {
			aliasCmd := findCommand(rootCmd, alias)
			require.NotNil(t, aliasCmd, "Alias %s should exist", alias)

			originalCmd := findCommand(rootCmd, original)
			require.NotNil(t, originalCmd, "Original command %s should exist", original)

			// Check that alias has same short description
			assert.Equal(t, originalCmd.Short, aliasCmd.Short)

			// Check that alias is hidden
			assert.True(t, aliasCmd.Hidden, "Alias %s should be hidden", alias)
		})
	}

	// Check workflow aliases were added
	workflowAliases := []string{"sw", "ls", "ld"}
	for _, alias := range workflowAliases {
		t.Run("workflow_alias_"+alias, func(t *testing.T) {
			cmd := findCommand(rootCmd, alias)
			assert.NotNil(t, cmd, "Workflow alias %s should exist", alias)
			assert.True(t, cmd.Hidden, "Workflow alias %s should be hidden", alias)
		})
	}
}

func TestAddWorkflowAliases(t *testing.T) {
	rootCmd := &cobra.Command{Use: "vaultenv"}

	// Add commands that workflow aliases depend on
	envCmd := &cobra.Command{Use: "env", Short: "Environment management"}
	switchCmd := &cobra.Command{
		Use: "switch",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	envCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(envCmd)

	listCmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	rootCmd.AddCommand(listCmd)

	loadCmd := &cobra.Command{
		Use: "load",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	loadCmd.Flags().String("from", "", "File to load from")
	rootCmd.AddCommand(loadCmd)

	// Add workflow aliases
	addWorkflowAliases(rootCmd)

	tests := []struct {
		name          string
		alias         string
		expectedArgs  cobra.PositionalArgs
		expectedShort string
	}{
		{
			name:          "sw alias",
			alias:         "sw",
			expectedShort: "Switch to a different environment",
		},
		{
			name:          "ls alias",
			alias:         "ls",
			expectedShort: "List all variables in current environment",
		},
		{
			name:          "ld alias",
			alias:         "ld",
			expectedShort: "Load variables from .env file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findCommand(rootCmd, tt.alias)
			require.NotNil(t, cmd, "Alias %s should exist", tt.alias)
			assert.Equal(t, tt.expectedShort, cmd.Short)
			assert.True(t, cmd.Hidden)

			// Test that RunE is set
			assert.NotNil(t, cmd.RunE)
		})
	}

	// Test sw command execution
	t.Run("sw command execution", func(t *testing.T) {
		swCmd := findCommand(rootCmd, "sw")
		require.NotNil(t, swCmd)

		// Should accept exactly one argument
		err := swCmd.Args(swCmd, []string{"production"})
		assert.NoError(t, err)

		err = swCmd.Args(swCmd, []string{})
		assert.Error(t, err)
	})

	// Test ld command execution
	t.Run("ld command execution", func(t *testing.T) {
		ldCmd := findCommand(rootCmd, "ld")
		require.NotNil(t, ldCmd)

		// Should accept 0 or 1 argument
		err := ldCmd.Args(ldCmd, []string{})
		assert.NoError(t, err)

		err = ldCmd.Args(ldCmd, []string{".env"})
		assert.NoError(t, err)

		err = ldCmd.Args(ldCmd, []string{".env", "extra"})
		assert.Error(t, err)
	})
}

func TestFindCommand(t *testing.T) {
	rootCmd := &cobra.Command{Use: "vaultenv"}

	// Add some test commands
	cmd1 := &cobra.Command{Use: "test1"}
	cmd2 := &cobra.Command{Use: "test2"}
	rootCmd.AddCommand(cmd1)
	rootCmd.AddCommand(cmd2)

	tests := []struct {
		name     string
		search   string
		expected *cobra.Command
	}{
		{
			name:     "find existing command",
			search:   "test1",
			expected: cmd1,
		},
		{
			name:     "find another existing command",
			search:   "test2",
			expected: cmd2,
		},
		{
			name:     "command not found",
			search:   "nonexistent",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findCommand(rootCmd, tt.search)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddShortHelp(t *testing.T) {
	rootCmd := &cobra.Command{Use: "vaultenv"}

	// Add short help
	AddShortHelp(rootCmd)

	// Find the aliases command
	aliasCmd := findCommand(rootCmd, "aliases")
	require.NotNil(t, aliasCmd, "aliases command should exist")

	// Check properties
	assert.Equal(t, "aliases", aliasCmd.Use)
	assert.Equal(t, "List all available command aliases", aliasCmd.Short)
	assert.Contains(t, aliasCmd.Long, "VaultEnv supports short aliases")
	assert.Contains(t, aliasCmd.Long, "Available aliases:")
	assert.Contains(t, aliasCmd.Long, "Workflow aliases:")
	assert.Contains(t, aliasCmd.Long, "Examples:")

	// Check that Run is set
	assert.NotNil(t, aliasCmd.Run)
}
