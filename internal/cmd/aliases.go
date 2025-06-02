package cmd

import (
	"github.com/spf13/cobra"
)

// addAliases adds short aliases for common commands
func addAliases(rootCmd *cobra.Command) {
	// Create short aliases for common commands
	aliasMap := map[string]string{
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

	for alias, original := range aliasMap {
		if cmd := findCommand(rootCmd, original); cmd != nil {
			// Create a copy of the command with a new name
			aliasCmd := &cobra.Command{
				Use:    alias,
				Short:  cmd.Short,
				Long:   cmd.Long,
				Hidden: true, // Hide from help to avoid clutter
				RunE:   cmd.RunE,
				Run:    cmd.Run,
				Args:   cmd.Args,
				// Inherit all the flags from the original command
				PreRunE:          cmd.PreRunE,
				PostRunE:         cmd.PostRunE,
				PersistentPreRunE: cmd.PersistentPreRunE,
			}

			// Copy flags from original command
			aliasCmd.Flags().AddFlagSet(cmd.Flags())
			aliasCmd.PersistentFlags().AddFlagSet(cmd.PersistentFlags())

			// Add any subcommands if the original has them
			for _, subCmd := range cmd.Commands() {
				aliasCmd.AddCommand(subCmd)
			}

			rootCmd.AddCommand(aliasCmd)
		}
	}

	// Add some composite aliases for common workflows
	addWorkflowAliases(rootCmd)
}

// addWorkflowAliases adds aliases for common workflows
func addWorkflowAliases(rootCmd *cobra.Command) {
	// Quick environment switch
	quickSwitch := &cobra.Command{
		Use:    "sw <environment>",
		Short:  "Switch to a different environment",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// This is equivalent to: vaultenv env switch <environment>
			if envCmd := findCommand(rootCmd, "env"); envCmd != nil {
				for _, subCmd := range envCmd.Commands() {
					if subCmd.Name() == "switch" {
						return subCmd.RunE(cmd, args)
					}
				}
			}
			return nil
		},
	}
	rootCmd.AddCommand(quickSwitch)

	// Quick list for current environment
	quickList := &cobra.Command{
		Use:    "ls",
		Short:  "List all variables in current environment",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// This is equivalent to: vaultenv list
			if listCmd := findCommand(rootCmd, "list"); listCmd != nil {
				return listCmd.RunE(cmd, args)
			}
			return nil
		},
	}
	rootCmd.AddCommand(quickList)

	// Quick load from .env file
	quickLoad := &cobra.Command{
		Use:    "ld [file]",
		Short:  "Load variables from .env file",
		Hidden: true,
		Args:   cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// This is equivalent to: vaultenv load --from <file>
			if loadCmd := findCommand(rootCmd, "load"); loadCmd != nil {
				// Set default file if not provided
				if len(args) == 0 {
					args = append(args, ".env")
				}
				// Set the --from flag
				loadCmd.Flags().Set("from", args[0])
				return loadCmd.RunE(cmd, []string{})
			}
			return nil
		},
	}
	rootCmd.AddCommand(quickLoad)
}

// findCommand recursively searches for a command by name
func findCommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, subCmd := range cmd.Commands() {
		if subCmd.Name() == name {
			return subCmd
		}
	}
	return nil
}

// AddShortHelp adds a help topic for aliases
func AddShortHelp(rootCmd *cobra.Command) {
	aliasHelp := &cobra.Command{
		Use:   "aliases",
		Short: "List all available command aliases",
		Long: `VaultEnv supports short aliases for common commands to improve your workflow.

Available aliases:
  s   → set        Set a variable
  g   → get        Get a variable
  l   → list       List variables
  e   → env        Environment management
  x   → export     Export variables
  i   → import     Import variables (via load command)
  p   → push       Push to git (git push)
  pl  → pull       Pull from git (git pull)
  h   → history    Show history
  a   → audit      Show audit log
  c   → config     Configuration management
  
Workflow aliases:
  sw  → env switch    Switch environment
  ls  → list          List variables
  ld  → load          Load from .env file

Examples:
  vaultenv s API_KEY abc123           # Same as: vaultenv set API_KEY abc123
  vaultenv g API_KEY                  # Same as: vaultenv get API_KEY
  vaultenv l                          # Same as: vaultenv list
  vaultenv sw production              # Same as: vaultenv env switch production
  vaultenv ld .env.local              # Same as: vaultenv load --from .env.local`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.AddCommand(aliasHelp)
}