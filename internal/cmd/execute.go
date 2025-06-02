package cmd

import (
	"context"
	"os"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
)

// BuildInfo contains version information passed from main
type BuildInfo struct {
	Version   string
	Commit    string
	BuildTime string
	BuiltBy   string
}

// ConfigKey is the context key for storing configuration
type configKey struct{}

var (
	// Global flags that affect all commands
	cfgFile string
	noColor bool
	verbose bool

	// Build information
	buildInfo BuildInfo

	// Global configuration instance
	globalConfig *config.Config

	// Root command definition
	rootCmd = &cobra.Command{
		Use:   "vaultenv-cli",
		Short: "🔐 Secure environment variable management for modern teams",

		// Long description shows when users run 'vaultenv-cli help'
		Long: `vaultenv-cli - Enterprise-grade environment variable management

vaultenv-cli makes managing environment variables across different environments
as simple as a single command, while maintaining bank-level security.

Perfect for teams who are tired of:
  • Manually syncing .env files
  • Sharing secrets through Slack
  • Breaking production with wrong configs
  • Not knowing who changed what and when`,

		// PersistentPreRun executes before any subcommand
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set UI output to command's output
			ui.SetOutput(cmd.OutOrStdout(), cmd.ErrOrStderr())

			// Configure color output based on flags and environment
			configureColorOutput()

			// Initialize configuration
			initializeConfig()

			// Load configuration unless this is the init command
			if cmd.Name() != "init" && !isInitCommand(cmd) {
				cfg, err := loadProjectConfig()
				if err != nil {
					ui.Error("Failed to load configuration: %v", err)
					ui.Info("Run 'vaultenv-cli init' to initialize a new project")
					os.Exit(1)
				}
				globalConfig = cfg

				// Store config in command context
				ctx := context.WithValue(cmd.Context(), configKey{}, cfg)
				cmd.SetContext(ctx)

				if verbose {
					ui.Debug("Loaded configuration for project: %s", cfg.Project.Name)
				}
			}

			// Show version info in verbose mode
			if verbose {
				ui.Debug("vaultenv-cli %s (commit: %s, built: %s)",
					buildInfo.Version, buildInfo.Commit, buildInfo.BuildTime)
			}
		},

		// Don't show errors twice
		SilenceErrors: true,

		// Don't show usage on errors automatically
		SilenceUsage: true,
	}
)

// Execute runs the root command
func Execute(info BuildInfo) error {
	buildInfo = info

	// Add all subcommands
	addCommands()

	// Execute the command tree
	if err := rootCmd.Execute(); err != nil {
		// Handle errors with helpful messages
		handleError(err)
		return err
	}

	return nil
}

// NewRootCommand creates a new root command for testing
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vaultenv-cli",
		Short: "🔐 Secure environment variable management for modern teams",
		Long:  rootCmd.Long,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set UI output to command's output
			ui.SetOutput(cmd.OutOrStdout(), cmd.ErrOrStderr())

			// Configure color output based on flags and environment
			configureColorOutput()

			// Initialize configuration
			initializeConfig()

			// Load configuration unless this is the init command
			if cmd.Name() != "init" && !isInitCommand(cmd) {
				cfg, err := loadProjectConfig()
				if err != nil {
					ui.Error("Failed to load configuration: %v", err)
					ui.Info("Run 'vaultenv-cli init' to initialize a new project")
					os.Exit(1)
				}
				globalConfig = cfg

				// Store config in command context
				ctx := context.WithValue(cmd.Context(), configKey{}, cfg)
				cmd.SetContext(ctx)

				if verbose {
					ui.Debug("Loaded configuration for project: %s", cfg.Project.Name)
				}
			}

			// Show version info in verbose mode
			if verbose {
				ui.Debug("vaultenv-cli %s (commit: %s, built: %s)",
					buildInfo.Version, buildInfo.Commit, buildInfo.BuildTime)
			}
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Define global flags
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default: $HOME/.vaultenv-cli/config.yaml)")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false,
		"disable colored output")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"enable verbose output")

	// Add all subcommands
	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newSetCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newCompletionCommand())
	cmd.AddCommand(newHistoryCommand())
	cmd.AddCommand(newRestoreCommand())
	cmd.AddCommand(newAuditCommand())
	cmd.AddCommand(newMigrateCommand())
	cmd.AddCommand(newGitCommand())
	cmd.AddCommand(newEnvCommand())
	cmd.AddCommand(newLoadCommand())
	cmd.AddCommand(newExportCommand())
	cmd.AddCommand(newBatchCommand())
	cmd.AddCommand(newConfigCommand())
	cmd.AddCommand(newSecurityCommand())
	cmd.AddCommand(newShellCommand())
	cmd.AddCommand(newRunCommand())

	// Add command aliases for better UX
	addAliases(cmd)

	// Add alias help
	AddShortHelp(cmd)

	return cmd
}

func init() {
	// Define global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default: $HOME/.vaultenv-cli/config.yaml)")

	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false,
		"disable colored output")

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"enable verbose output")

	// Bind flags to viper for configuration management
	viper.BindPFlag("no_color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func configureColorOutput() {
	// Respect user preferences and environment
	if noColor || os.Getenv("NO_COLOR") != "" {
		color.NoColor = true
		return
	}

	// Detect if we're in a CI environment
	if os.Getenv("CI") != "" {
		color.NoColor = true
		return
	}

	// Windows requires special handling for color support
	if runtime.GOOS == "windows" {
		// Color package handles Windows automatically
		return
	}
}

func initializeConfig() {
	if cfgFile != "" {
		// Use config file from flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in standard locations
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")

		// Add search paths in order of precedence
		viper.AddConfigPath(".")                   // Current directory
		viper.AddConfigPath("$HOME/.vaultenv-cli") // User config directory
		viper.AddConfigPath("/etc/vaultenv-cli")   // System config directory
	}

	// Environment variables override config file
	viper.SetEnvPrefix("vaultenv-cli")
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil && verbose {
		ui.Debug("Using config file: %s", viper.ConfigFileUsed())
	}
}

func handleError(err error) {
	// This is where we make errors helpful, not frustrating
	ui.HandleError(err)
}

func addCommands() {
	// Add all subcommands
	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newSetCommand())
	rootCmd.AddCommand(newGetCommand())
	rootCmd.AddCommand(newListCommand())
	rootCmd.AddCommand(newInitCommand())
	rootCmd.AddCommand(newCompletionCommand())
	rootCmd.AddCommand(newHistoryCommand())
	rootCmd.AddCommand(newRestoreCommand())
	rootCmd.AddCommand(newAuditCommand())
	rootCmd.AddCommand(newMigrateCommand())
	rootCmd.AddCommand(newGitCommand())
	rootCmd.AddCommand(newEnvCommand())
	rootCmd.AddCommand(newLoadCommand())
	rootCmd.AddCommand(newExportCommand())
	rootCmd.AddCommand(newBatchCommand())
	rootCmd.AddCommand(newConfigCommand())
	rootCmd.AddCommand(newSecurityCommand())
	rootCmd.AddCommand(newShellCommand())
	rootCmd.AddCommand(newRunCommand())

	// Add command aliases for better UX
	addAliases(rootCmd)

	// Add alias help
	AddShortHelp(rootCmd)
}

// loadProjectConfig loads the project configuration
func loadProjectConfig() (*config.Config, error) {
	// Try to load from specified config file first
	if cfgFile != "" {
		return config.LoadFromFile(cfgFile)
	}

	// Otherwise, use the default loading mechanism which walks up the directory tree
	return config.Load()
}

// isInitCommand checks if the current command is init or a parent of init
func isInitCommand(cmd *cobra.Command) bool {
	// Check if any parent command is init
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "init" {
			return true
		}
	}

	// Check if any of the command line args indicate init command
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if arg == "init" {
				return true
			}
			// Stop checking at first non-flag argument that isn't init
			if !strings.HasPrefix(arg, "-") && arg != "init" {
				break
			}
		}
	}

	return false
}

// GetConfig retrieves the configuration from the command context
func GetConfig(cmd *cobra.Command) *config.Config {
	if cmd.Context() == nil {
		return globalConfig
	}

	cfg, ok := cmd.Context().Value(configKey{}).(*config.Config)
	if !ok || cfg == nil {
		return globalConfig
	}

	return cfg
}

// MustGetConfig retrieves the configuration or panics if not found
func MustGetConfig(cmd *cobra.Command) *config.Config {
	cfg := GetConfig(cmd)
	if cfg == nil {
		panic("configuration not loaded")
	}
	return cfg
}
