package cmd

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"
    "github.com/AlecAivazis/survey/v2"

    "github.com/vaultenv/vaultenv-cli/internal/auth"
    "github.com/vaultenv/vaultenv-cli/internal/config"
    "github.com/vaultenv/vaultenv-cli/internal/keystore"
    "github.com/vaultenv/vaultenv-cli/internal/ui"
    "github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func newSetCommand() *cobra.Command {
    var (
        environment string
        force      bool
        encrypt    bool
    )

    cmd := &cobra.Command{
        Use:   "set KEY=VALUE [KEY=VALUE...]",
        Short: "Set one or more environment variables",
        Long: `Set environment variables in the specified environment.

Values are encrypted by default before storage. Use --no-encrypt only
for non-sensitive values to improve performance.`,

        Example: `  # Set a single variable
  vaultenv-cli set DATABASE_URL=postgres://localhost/myapp

  # Set multiple variables
  vaultenv-cli set API_KEY=secret DATABASE_URL=postgres://localhost

  # Set in specific environment
  vaultenv-cli set API_KEY=prod-secret --env production

  # Set without encryption (only for non-sensitive data)
  vaultenv-cli set LOG_LEVEL=debug --no-encrypt`,

        Args: cobra.MinimumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return runSet(args, environment, force, encrypt)
        },
    }

    // Add command-specific flags
    cmd.Flags().StringVarP(&environment, "env", "e", "development",
        "environment to set variables in")
    cmd.Flags().BoolVarP(&force, "force", "f", false,
        "overwrite existing variables without confirmation")
    cmd.Flags().BoolVar(&encrypt, "encrypt", true,
        "encrypt values before storage")

    // Register completion functions for better UX
    cmd.RegisterFlagCompletionFunc("env", environmentCompletion)
    
    // Register custom validation function that provides variable name suggestions
    cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        // If toComplete contains '=', don't provide completions (user is typing value)
        if strings.Contains(toComplete, "=") {
            return nil, cobra.ShellCompDirectiveNoFileComp
        }
        
        // Otherwise, provide variable name suggestions with '=' appended
        suggestions, directive := variableNameCompletion(cmd, args, toComplete)
        
        // Append '=' to each suggestion for convenience
        for i := range suggestions {
            suggestions[i] = suggestions[i] + "="
        }
        
        return suggestions, directive
    }

    return cmd
}

func runSet(args []string, environment string, force bool, encrypt bool) error {
    // Parse KEY=VALUE pairs
    vars, err := parseVariables(args)
    if err != nil {
        return fmt.Errorf("invalid variable format: %w", err)
    }

    // Show what we're about to do
    ui.Info("Setting %d variable(s) in %s environment", len(vars), environment)

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load configuration: %w", err)
    }

    // Initialize storage options
    storageOpts := storage.BackendOptions{
        Environment: environment,
    }

    // If encryption is enabled and not in test environment, set up authentication
    if encrypt && !isTestEnvironment() {
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

    // Get storage backend with options
    store, err := storage.GetBackendWithOptions(storageOpts)
    if err != nil {
        return fmt.Errorf("failed to initialize storage: %w", err)
    }
    defer store.Close()

    // Process each variable
    for key, value := range vars {
        // Check if variable already exists
        exists, err := store.Exists(key)
        if err != nil {
            return fmt.Errorf("failed to check variable: %w", err)
        }

        // Confirm overwrite if needed
        if exists && !force {
            overwrite := false
            prompt := &survey.Confirm{
                Message: fmt.Sprintf("Variable %s already exists. Overwrite?", key),
                Default: false,
            }

            if err := survey.AskOne(prompt, &overwrite); err != nil {
                return err
            }

            if !overwrite {
                ui.Info("Skipping %s", key)
                continue
            }
        }

        // Store the variable
        err = ui.StartProgress(fmt.Sprintf("Setting %s", key), func() error {
            return store.Set(key, value, encrypt)
        })
        if err != nil {
            return fmt.Errorf("failed to set %s: %w", key, err)
        }
    }

    ui.Success("Variables set successfully")
    return nil
}

func parseVariables(args []string) (map[string]string, error) {
    vars := make(map[string]string)

    for _, arg := range args {
        parts := strings.SplitN(arg, "=", 2)
        if len(parts) != 2 {
            return nil, fmt.Errorf("invalid format: %s (expected KEY=VALUE)", arg)
        }

        key := strings.TrimSpace(parts[0])
        value := parts[1]  // Don't trim spaces from values - they might be intentional

        // Check for spaces in variable name before validation
        if strings.Contains(key, " ") {
            return nil, fmt.Errorf("invalid format: %s (variable names cannot contain spaces)", arg)
        }

        // Validate key format
        if !isValidVariableName(key) {
            return nil, fmt.Errorf("invalid variable name: %s", key)
        }

        vars[key] = value
    }

    return vars, nil
}

func isValidVariableName(name string) bool {
    // Environment variable names should follow conventions
    // Must start with letter or underscore
    // Can contain letters, numbers, and underscores
    if len(name) == 0 {
        return false
    }

    for i, ch := range name {
        if i == 0 {
            if !isLetter(ch) && ch != '_' {
                return false
            }
        } else {
            if !isLetter(ch) && !isDigit(ch) && ch != '_' {
                return false
            }
        }
    }

    return true
}

func isLetter(ch rune) bool {
    return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func isDigit(ch rune) bool {
    return ch >= '0' && ch <= '9'
}

