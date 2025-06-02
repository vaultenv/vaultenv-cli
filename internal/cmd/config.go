package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"gopkg.in/yaml.v3"
)

// newConfigCommand creates the config command
func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage VaultEnv configuration",
		Long: `View and modify VaultEnv configuration settings.

The config command allows you to view, get, and set configuration values.
Configuration is stored in .vaultenv/config.yaml and can be edited manually
or through this command.`,
		Example: `  # View complete configuration
  vaultenv config

  # Get a specific value
  vaultenv config get git.auto_commit
  vaultenv config get security.per_environment_passwords

  # Set a value
  vaultenv config set git.auto_commit true
  vaultenv config set export.default_format json

  # Reset to defaults
  vaultenv config reset

  # Edit configuration in editor
  vaultenv config edit`,
	}

	cmd.AddCommand(
		newConfigGetCommand(),
		newConfigSetCommand(),
		newConfigResetCommand(),
		newConfigEditCommand(),
		newConfigMigrateCommand(),
	)

	// Default action is to display config
	cmd.RunE = runConfigShow

	return cmd
}

// runConfigShow displays the current configuration
func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Marshal to YAML for display
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	ui.Header("Current Configuration")
	fmt.Println(string(data))

	return nil
}

// newConfigGetCommand creates the config get subcommand
func newConfigGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a specific configuration value using dot notation.

Examples:
  vaultenv config get git.auto_commit
  vaultenv config get security.password_policy.min_length
  vaultenv config get environments.production.password_policy.min_length`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigGet,
	}
}

// runConfigGet gets a specific configuration value
func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Convert to map for easy access
	var configMap map[string]interface{}
	data, _ := yaml.Marshal(cfg)
	yaml.Unmarshal(data, &configMap)

	value, err := getNestedValue(configMap, key)
	if err != nil {
		return fmt.Errorf("key not found: %s", key)
	}

	// Display the value
	switch v := value.(type) {
	case string, int, bool, float64:
		fmt.Println(v)
	default:
		// For complex types, show as YAML
		output, _ := yaml.Marshal(v)
		fmt.Print(string(output))
	}

	return nil
}

// newConfigSetCommand creates the config set subcommand
func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a specific configuration value using dot notation.

Examples:
  vaultenv config set git.auto_commit true
  vaultenv config set security.password_policy.min_length 16
  vaultenv config set export.default_format json`,
		Args: cobra.ExactArgs(2),
		RunE: runConfigSet,
	}
}

// runConfigSet sets a specific configuration value
func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Convert to map for manipulation
	var configMap map[string]interface{}
	data, _ := yaml.Marshal(cfg)
	yaml.Unmarshal(data, &configMap)

	// Set the value
	if err := setNestedValue(configMap, key, value); err != nil {
		return fmt.Errorf("failed to set value: %w", err)
	}

	// Convert back to Config struct
	updatedData, _ := yaml.Marshal(configMap)
	var updatedConfig config.Config
	if err := yaml.Unmarshal(updatedData, &updatedConfig); err != nil {
		return fmt.Errorf("failed to parse updated config: %w", err)
	}

	// Validate before saving
	if err := updatedConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save the configuration
	if err := updatedConfig.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	ui.Success("Configuration updated: %s = %s", key, value)
	return nil
}

// newConfigResetCommand creates the config reset subcommand
func newConfigResetCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration to defaults",
		Long: `Reset the configuration to default values.

This will overwrite your current configuration with the default values.
Use --force to skip confirmation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				if !ui.Confirm("Are you sure you want to reset configuration to defaults?") {
					return fmt.Errorf("reset cancelled")
				}
			}

			// Create default config
			defaultConfig := config.DefaultConfig()

			// Save it
			if err := defaultConfig.Save(); err != nil {
				return fmt.Errorf("failed to save default config: %w", err)
			}

			ui.Success("Configuration reset to defaults")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}

// newConfigEditCommand creates the config edit subcommand
func newConfigEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit configuration in your editor",
		Long: `Open the configuration file in your default editor.

The editor is determined by the EDITOR environment variable.
If not set, it will try common editors (vim, nano, code).`,
		RunE: runConfigEdit,
	}
}

// runConfigEdit opens the config file in an editor
func runConfigEdit(cmd *cobra.Command, args []string) error {
	configPath := ".vaultenv/config.yaml"

	// Ensure config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config if it doesn't exist
		defaultConfig := config.DefaultConfig()
		if err := defaultConfig.Save(); err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
	}

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Try common editors
		editors := []string{"vim", "vi", "nano", "code", "subl", "atom"}
		for _, e := range editors {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}

	if editor == "" {
		return fmt.Errorf("no editor found. Set EDITOR environment variable")
	}

	// Open editor
	editorCmd := exec.Command(editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	// Validate the edited config
	_, err := config.LoadFromFile(configPath)
	if err != nil {
		ui.Error("Configuration is invalid after edit: %v", err)
		if ui.Confirm("Would you like to edit again?") {
			return runConfigEdit(cmd, args)
		}
		return err
	}

	ui.Success("Configuration updated and validated")
	return nil
}

// newConfigMigrateCommand creates the config migrate subcommand
func newConfigMigrateCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate configuration to latest version",
		Long: `Migrate your configuration to the latest version.

This command will update your configuration format to the latest version,
applying any necessary transformations and adding new default values.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if !config.NeedsMigration(cfg) {
				ui.Info("Configuration is already at the latest version (%s)", cfg.Version)
				return nil
			}

			ui.Info("Current version: %s", cfg.Version)
			ui.Info("Target version: %s", config.CurrentVersion)

			if !force {
				if !ui.Confirm("Migrate configuration to latest version?") {
					return fmt.Errorf("migration cancelled")
				}
			}

			// Backup current config
			backupPath := fmt.Sprintf(".vaultenv/config.yaml.backup-%s", time.Now().Format("20060102-150405"))
			if err := copyFile(".vaultenv/config.yaml", backupPath); err != nil {
				ui.Warning("Failed to create backup: %v", err)
			} else {
				ui.Info("Created backup at %s", backupPath)
			}

			// Migrate
			migratedConfig, err := config.MigrateConfig(cfg)
			if err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			// Save migrated config
			if err := migratedConfig.Save(); err != nil {
				return fmt.Errorf("failed to save migrated config: %w", err)
			}

			ui.Success("Configuration migrated to version %s", config.CurrentVersion)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}

// Helper functions for nested key access

func getNestedValue(m map[string]interface{}, key string) (interface{}, error) {
	parts := strings.Split(key, ".")
	current := m

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, return the value
			if val, ok := current[part]; ok {
				return val, nil
			}
			return nil, fmt.Errorf("key not found")
		}

		// Navigate deeper
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return nil, fmt.Errorf("invalid path")
		}
	}

	return nil, fmt.Errorf("key not found")
}

func setNestedValue(m map[string]interface{}, key string, value string) error {
	parts := strings.Split(key, ".")
	current := m

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, set the value
			// Try to parse the value to appropriate type
			if value == "true" {
				current[part] = true
			} else if value == "false" {
				current[part] = false
			} else if num, err := strconv.Atoi(value); err == nil {
				current[part] = num
			} else {
				current[part] = value
			}
			return nil
		}

		// Navigate deeper, creating maps as needed
		if _, ok := current[part]; !ok {
			current[part] = make(map[string]interface{})
		}

		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return fmt.Errorf("cannot set value: path exists with non-map value")
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0600)
}
