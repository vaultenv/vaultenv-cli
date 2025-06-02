package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/internal/auth"
	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/keystore"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func newShellCommand() *cobra.Command {
	var (
		environment string
		shell       string
	)

	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Load variables into shell environment",
		Long:  `Load environment variables into the current shell session or output shell commands to source.`,

		Example: `  # Load variables into current shell (run with eval)
  eval "$(vaultenv shell)"
  
  # Load variables for production environment
  eval "$(vaultenv shell --env production)"
  
  # Specify shell type explicitly
  eval "$(vaultenv shell --shell bash)"
  
  # For fish shell
  vaultenv shell --shell fish | source`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runShell(environment, shell)
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "development", "environment to load")
	cmd.Flags().StringVar(&shell, "shell", "", "shell type (bash, zsh, fish, powershell)")

	return cmd
}

func newRunCommand() *cobra.Command {
	var (
		environment string
	)

	cmd := &cobra.Command{
		Use:   "run -- COMMAND [ARGS...]",
		Short: "Run command with environment variables",
		Long:  `Run a command with environment variables loaded from VaultEnv.`,

		Example: `  # Run npm start with development variables
  vaultenv run -- npm start
  
  # Run command in production environment
  vaultenv run --env production -- node server.js
  
  # Run with multiple arguments
  vaultenv run -- python manage.py runserver 0.0.0.0:8000`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("no command specified. Use -- before the command")
			}
			return runWithEnv(environment, args)
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "development", "environment to use")

	return cmd
}

// Implementation functions

func runShell(environment, shellType string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if environment exists
	if !cfg.HasEnvironment(environment) {
		return fmt.Errorf("environment '%s' does not exist", environment)
	}

	// Auto-detect shell if not specified
	if shellType == "" {
		shellType = detectShell()
	}

	// Get environment variables
	vars, err := getEnvironmentVariables(cfg, environment)
	if err != nil {
		return fmt.Errorf("failed to get environment variables: %w", err)
	}

	// Generate shell commands
	commands := generateShellCommands(vars, shellType)

	// Output shell commands
	for _, cmd := range commands {
		fmt.Println(cmd)
	}

	return nil
}

func runWithEnv(environment string, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if environment exists
	if !cfg.HasEnvironment(environment) {
		return fmt.Errorf("environment '%s' does not exist", environment)
	}

	// Get environment variables
	vars, err := getEnvironmentVariables(cfg, environment)
	if err != nil {
		return fmt.Errorf("failed to get environment variables: %w", err)
	}

	// Prepare command
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	// Create command
	cmd := exec.Command(cmdName, cmdArgs...)

	// Set environment variables
	cmd.Env = os.Environ() // Start with current environment
	for key, value := range vars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Connect stdin, stdout, stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run command
	err = cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}

// Helper functions

func detectShell() string {
	// Try to detect shell from environment
	shell := os.Getenv("SHELL")
	if shell == "" {
		// Fallback based on OS
		switch runtime.GOOS {
		case "windows":
			return "powershell"
		default:
			return "bash"
		}
	}

	// Extract shell name from path
	parts := strings.Split(shell, "/")
	shellName := parts[len(parts)-1]

	// Normalize shell names
	switch shellName {
	case "bash", "sh":
		return "bash"
	case "zsh":
		return "zsh"
	case "fish":
		return "fish"
	case "powershell", "pwsh":
		return "powershell"
	default:
		return "bash" // Default fallback
	}
}

func generateShellCommands(vars map[string]string, shellType string) []string {
	var commands []string

	switch shellType {
	case "fish":
		for key, value := range vars {
			// Fish shell syntax
			escapedValue := strings.ReplaceAll(value, "'", "\\'")
			commands = append(commands, fmt.Sprintf("set -x %s '%s'", key, escapedValue))
		}
	case "powershell":
		for key, value := range vars {
			// PowerShell syntax
			escapedValue := strings.ReplaceAll(value, "'", "''")
			commands = append(commands, fmt.Sprintf("$env:%s = '%s'", key, escapedValue))
		}
	default: // bash, zsh, sh
		for key, value := range vars {
			// Bash/Zsh syntax
			escapedValue := strings.ReplaceAll(value, "'", "'\"'\"'")
			commands = append(commands, fmt.Sprintf("export %s='%s'", key, escapedValue))
		}
	}

	return commands
}

func getEnvironmentVariables(cfg *config.Config, environment string) (map[string]string, error) {
	// Check if we're in a test environment
	isTest := isTestEnvironment()

	// Create backend options
	opts := storage.BackendOptions{
		Environment: environment,
		Type:        cfg.Vault.Type,
		BasePath:    cfg.Vault.Path,
	}

	// Handle authentication if not in test mode
	if !isTest && cfg.Vault.IsEncrypted() {
		ks, err := keystore.NewKeystore(cfg.Vault.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize keystore: %w", err)
		}

		pm := auth.NewPasswordManager(ks, cfg)

		// Get or create encryption key
		key, err := pm.GetOrCreateMasterKey(cfg.Project.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get encryption key: %w", err)
		}

		// Convert key to string for storage options
		opts.Password = string(key)
	}

	// Get storage backend
	store, err := storage.GetBackendWithOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage backend: %w", err)
	}
	defer store.Close()

	// Get all variable keys
	keys, err := store.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	// Get all variable values
	vars := make(map[string]string)
	for _, key := range keys {
		value, err := store.Get(key)
		if err != nil {
			ui.Warning("Failed to get variable '%s': %v", key, err)
			continue
		}
		vars[key] = value
	}

	return vars, nil
}
