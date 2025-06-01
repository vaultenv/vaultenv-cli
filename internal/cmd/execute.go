package cmd

import (
    "os"
    "runtime"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "github.com/fatih/color"

    "github.com/vaultenv/vaultenv-cli/internal/ui"
)

// BuildInfo contains version information passed from main
type BuildInfo struct {
    Version   string
    Commit    string
    BuildTime string
    BuiltBy   string
}

var (
    // Global flags that affect all commands
    cfgFile     string
    noColor     bool
    verbose     bool

    // Build information
    buildInfo   BuildInfo

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
            // Configure color output based on flags and environment
            configureColorOutput()

            // Initialize configuration
            initializeConfig()

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
        viper.AddConfigPath(".")                          // Current directory
        viper.AddConfigPath("$HOME/.vaultenv-cli")       // User config directory
        viper.AddConfigPath("/etc/vaultenv-cli")         // System config directory
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
    // TODO: Add other commands
    // rootCmd.AddCommand(newSetCommand())
    // rootCmd.AddCommand(newGetCommand())
    // rootCmd.AddCommand(newListCommand())
    // rootCmd.AddCommand(newInitCommand())
    // rootCmd.AddCommand(newCompletionCommand())
}