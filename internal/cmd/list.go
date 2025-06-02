package cmd

import (
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "github.com/spf13/cobra"

    "github.com/vaultenv/vaultenv-cli/internal/auth"
    "github.com/vaultenv/vaultenv-cli/internal/config"
    "github.com/vaultenv/vaultenv-cli/internal/keystore"
    "github.com/vaultenv/vaultenv-cli/internal/ui"
    "github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func newListCommand() *cobra.Command {
    var (
        environment string
        showValues  bool
        pattern     string
    )

    cmd := &cobra.Command{
        Use:   "list",
        Short: "List environment variables",
        Long: `List all environment variables in the specified environment.

By default, only variable names are shown for security.
Use --values to also display the values.`,

        Example: `  # List all variables
  vaultenv-cli list

  # List with values
  vaultenv-cli list --values

  # List from specific environment
  vaultenv-cli list --env production

  # Filter by pattern
  vaultenv-cli list --pattern "API_*"`,

        RunE: func(cmd *cobra.Command, args []string) error {
            return runList(cmd, environment, showValues, pattern)
        },
    }

    // Add command-specific flags
    cmd.Flags().StringVarP(&environment, "env", "e", "development",
        "environment to list variables from")
    cmd.Flags().BoolVar(&showValues, "values", false,
        "show variable values (use with caution)")
    cmd.Flags().StringVarP(&pattern, "pattern", "p", "",
        "filter variables by pattern (supports wildcards)")

    // Register completion functions
    cmd.RegisterFlagCompletionFunc("env", environmentCompletion)
    cmd.RegisterFlagCompletionFunc("pattern", patternCompletion)

    return cmd
}

func runList(cmd *cobra.Command, environment string, showValues bool, pattern string) error {
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
                pm := auth.NewPasswordManager(ks, cfg)

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

    // Get all variable names
    keys, err := store.List()
    if err != nil {
        return fmt.Errorf("failed to list variables: %w", err)
    }

    if len(keys) == 0 {
        ui.Info("No variables found in %s environment", environment)
        return nil
    }

    // Filter by pattern if provided
    if pattern != "" {
        keys = filterKeys(keys, pattern)
        if len(keys) == 0 {
            ui.Info("No variables matching pattern '%s'", pattern)
            return nil
        }
    }

    // Sort keys for consistent output
    sort.Strings(keys)

    // Display header
    ui.Header(fmt.Sprintf("Environment: %s", environment))
    fmt.Fprintln(cmd.OutOrStdout())

    // Display variables
    if showValues {
        // Show as table with values
        maxKeyLen := 0
        for _, key := range keys {
            if len(key) > maxKeyLen {
                maxKeyLen = len(key)
            }
        }

        for _, key := range keys {
            value, err := store.Get(key)
            if err != nil {
                ui.Warning("Failed to get %s: %v", key, err)
                continue
            }

            // Truncate long values for display
            displayValue := value
            if len(displayValue) > 50 {
                displayValue = displayValue[:47] + "..."
            }

            fmt.Fprintf(cmd.OutOrStdout(), "%-*s = %s\n", maxKeyLen, key, displayValue)
        }
    } else {
        // Show only keys
        for _, key := range keys {
            fmt.Fprintln(cmd.OutOrStdout(), key)
        }
    }

    fmt.Fprintln(cmd.OutOrStdout())
    ui.Info("Total: %d variable(s)", len(keys))

    return nil
}

// filterKeys filters keys by pattern (supports * wildcard)
func filterKeys(keys []string, pattern string) []string {
    filtered := []string{}  // Initialize as empty slice, not nil
    for _, key := range keys {
        if matched, _ := matchPattern(key, pattern); matched {
            filtered = append(filtered, key)
        }
    }
    return filtered
}

// matchPattern is a simple pattern matcher
func matchPattern(s, pattern string) (bool, error) {
    // Handle exact match
    if !strings.Contains(pattern, "*") {
        return s == pattern, nil
    }

    // Convert pattern to a simple glob pattern
    // For now, we'll use simple string matching instead of regex
    
    // Use simple string matching for common patterns
    if pattern == "*" {
        return true, nil
    }
    
    if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
        // *substring*
        substring := pattern[1:len(pattern)-1]
        return strings.Contains(s, substring), nil
    }
    
    if strings.HasPrefix(pattern, "*") {
        // *suffix
        suffix := pattern[1:]
        return strings.HasSuffix(s, suffix), nil
    }
    
    if strings.HasSuffix(pattern, "*") {
        // prefix*
        prefix := pattern[:len(pattern)-1]
        return strings.HasPrefix(s, prefix), nil
    }
    
    // For more complex patterns, use simple matching
    parts := strings.Split(pattern, "*")
    
    // Check if string starts with the first part
    if parts[0] != "" && !strings.HasPrefix(s, parts[0]) {
        return false, nil
    }
    
    // Check if string ends with the last part
    if parts[len(parts)-1] != "" && !strings.HasSuffix(s, parts[len(parts)-1]) {
        return false, nil
    }
    
    // For patterns like "prefix*middle*suffix", we need more complex matching
    // For now, return true if all parts exist in order
    remaining := s
    for i, part := range parts {
        if part == "" {
            continue
        }
        
        idx := strings.Index(remaining, part)
        if idx == -1 {
            return false, nil
        }
        
        // For the first part, it must be at the beginning
        if i == 0 && idx != 0 {
            return false, nil
        }
        
        remaining = remaining[idx+len(part):]
    }
    
    return true, nil
}