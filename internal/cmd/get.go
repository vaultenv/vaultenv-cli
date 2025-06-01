package cmd

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"

    "github.com/vaultenv/vaultenv-cli/internal/auth"
    "github.com/vaultenv/vaultenv-cli/internal/config"
    "github.com/vaultenv/vaultenv-cli/internal/keystore"
    "github.com/vaultenv/vaultenv-cli/internal/ui"
    "github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func newGetCommand() *cobra.Command {
    var (
        environment string
        export      bool
        quiet       bool
    )

    cmd := &cobra.Command{
        Use:   "get KEY [KEY...]",
        Short: "Get environment variable values",
        Long: `Get the values of one or more environment variables.

By default, prints the variable in KEY=VALUE format.
Use --quiet to print only the value (useful for scripts).
Use --export to print in shell export format.`,

        Example: `  # Get a single variable
  vaultenv-cli get DATABASE_URL

  # Get multiple variables
  vaultenv-cli get API_KEY DATABASE_URL

  # Get from specific environment
  vaultenv-cli get API_KEY --env production

  # Get value only (for scripts)
  vaultenv-cli get API_KEY --quiet

  # Export format
  vaultenv-cli get API_KEY --export`,

        Args: cobra.MinimumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return runGet(cmd, args, environment, export, quiet)
        },
    }

    // Add command-specific flags
    cmd.Flags().StringVarP(&environment, "env", "e", "development",
        "environment to get variables from")
    cmd.Flags().BoolVarP(&export, "export", "x", false,
        "output in shell export format")
    cmd.Flags().BoolVarP(&quiet, "quiet", "q", false,
        "output only values (no keys)")

    // Register completion functions
    cmd.RegisterFlagCompletionFunc("env", environmentCompletion)
    
    // Register positional argument completion for existing variables
    cmd.ValidArgsFunction = existingVariableCompletion

    return cmd
}

func runGet(cmd *cobra.Command, keys []string, environment string, export, quiet bool) error {
    // Initialize storage options
    storageOpts := storage.BackendOptions{
        Environment: environment,
    }

    // Check if we're using a test backend (for unit tests)
    if !isTestEnvironment() {
        // Load configuration
        cfg, err := config.Load()
        if err != nil {
            return fmt.Errorf("failed to load configuration: %w", err)
        }

        // Check if the vault exists and might be encrypted
        vaultPath := filepath.Join(".vaultenv", environment+".env")
        if _, err := os.Stat(vaultPath); err == nil {
            // Vault exists, check if it's encrypted by trying to read it
            tempStore, err := storage.GetBackend(environment)
            if err != nil {
                return fmt.Errorf("failed to initialize storage: %w", err)
            }
            
            // Try to list variables to check if encrypted
            _, listErr := tempStore.List()
            tempStore.Close()
            
            // If we get an error that looks like encryption-related, set up authentication
            if listErr != nil {
                // Get project ID from config or use default
                projectID := cfg.Project.ID
                if projectID == "" {
                    projectID = cfg.Project.Name
                }

                // Initialize keystore
                homeDir, err := os.UserHomeDir()
                if err != nil {
                    return fmt.Errorf("failed to get home directory: %w", err)
                }
                dataDir := filepath.Join(homeDir, ".vaultenv", "data")

                ks, err := keystore.NewKeystore(dataDir)
                if err != nil {
                    return fmt.Errorf("failed to initialize keystore: %w", err)
                }

                // Create password manager
                pm := auth.NewPasswordManager(ks)

                // Get or create encryption key
                key, err := pm.GetOrCreateMasterKey(projectID)
                if err != nil {
                    return fmt.Errorf("failed to get encryption key: %w", err)
                }

                // Convert key to string for storage options
                storageOpts.Password = string(key)
            }
        }
    }

    // Get storage backend with options
    store, err := storage.GetBackendWithOptions(storageOpts)
    if err != nil {
        return fmt.Errorf("failed to initialize storage: %w", err)
    }
    defer store.Close()

    // Track if we found any variables
    found := false

    // Get each requested variable
    for _, key := range keys {
        value, err := store.Get(key)
        if err != nil {
            if err == storage.ErrNotFound {
                ui.Warning("Variable %s not found", key)
                continue
            }
            return fmt.Errorf("failed to get %s: %w", key, err)
        }

        found = true

        // Format output based on flags
        if quiet {
            fmt.Fprintln(cmd.OutOrStdout(), value)
        } else if export {
            fmt.Fprintf(cmd.OutOrStdout(), "export %s=\"%s\"\n", key, escapeShellValue(value))
        } else {
            fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", key, value)
        }
    }

    if !found {
        return fmt.Errorf("no variables found")
    }

    return nil
}

// escapeShellValue escapes a value for shell export
func escapeShellValue(value string) string {
    // Escape backslashes first (must be done before other escapes)
    value = strings.ReplaceAll(value, "\\", "\\\\")
    // Escape double quotes
    value = strings.ReplaceAll(value, "\"", "\\\"")
    // Escape dollar signs
    value = strings.ReplaceAll(value, "$", "\\$")
    // Escape backticks
    value = strings.ReplaceAll(value, "`", "\\`")
    return value
}