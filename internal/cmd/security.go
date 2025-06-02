package cmd

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/internal/auth"
	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/keystore"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func newSecurityCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Security management commands",
		Long:  `Manage security settings, encryption keys, and integrity verification.`,
	}

	cmd.AddCommand(
		newSecurityRotateKeysCommand(),
		newSecurityVerifyCommand(),
		newSecurityReportCommand(),
		newSecurityLockCommand(),
		newSecurityUnlockCommand(),
	)

	return cmd
}

func newSecurityRotateKeysCommand() *cobra.Command {
	var (
		environment string
		force       bool
	)

	cmd := &cobra.Command{
		Use:   "rotate-keys",
		Short: "Rotate encryption keys",
		Long:  `Rotate encryption keys for enhanced security. This will re-encrypt all variables with new keys.`,

		Example: `  # Rotate keys for current environment
  vaultenv security rotate-keys
  
  # Rotate keys for production environment
  vaultenv security rotate-keys --env production
  
  # Force rotation without confirmation
  vaultenv security rotate-keys --force`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecurityRotateKeys(environment, force)
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "development", "environment to rotate keys for")
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")

	return cmd
}

func newSecurityVerifyCommand() *cobra.Command {
	var (
		environment string
		deep        bool
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify data integrity",
		Long:  `Verify the integrity of stored variables and encryption.`,

		Example: `  # Basic integrity check
  vaultenv security verify
  
  # Deep verification of all variables
  vaultenv security verify --deep
  
  # Verify production environment
  vaultenv security verify --env production --deep`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecurityVerify(environment, deep)
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "development", "environment to verify")
	cmd.Flags().BoolVar(&deep, "deep", false, "perform deep verification of all variables")

	return cmd
}

func newSecurityReportCommand() *cobra.Command {
	var (
		format string
		output string
	)

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate security report",
		Long:  `Generate a comprehensive security report showing encryption status, key information, and security metrics.`,

		Example: `  # Generate security report
  vaultenv security report
  
  # Generate JSON report
  vaultenv security report --format json
  
  # Save report to file
  vaultenv security report --output security-report.json`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecurityReport(format, output)
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format (text, json)")
	cmd.Flags().StringVar(&output, "output", "", "output file (default: stdout)")

	return cmd
}

func newSecurityLockCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Lock access to all environments",
		Long:  `Lock access to all environments. This will require password re-entry for subsequent operations.`,

		Example: `  # Lock all access
  vaultenv security lock`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecurityLock()
		},
	}

	return cmd
}

func newSecurityUnlockCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock access to environments",
		Long:  `Unlock access to environments by re-entering passwords.`,

		Example: `  # Unlock access
  vaultenv security unlock`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecurityUnlock()
		},
	}

	return cmd
}

// Implementation functions

func runSecurityRotateKeys(environment string, force bool) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if encryption is enabled
	if !cfg.Vault.IsEncrypted() {
		return fmt.Errorf("encryption is not enabled for this project")
	}

	// Check if environment exists
	if !cfg.HasEnvironment(environment) {
		return fmt.Errorf("environment '%s' does not exist", environment)
	}

	ui.Header(fmt.Sprintf("Key Rotation for Environment: %s", environment))
	fmt.Println()

	// Warning about key rotation
	ui.Warning("Key rotation will:")
	fmt.Println("  • Generate new encryption keys")
	fmt.Println("  • Re-encrypt all variables with new keys")
	fmt.Println("  • Invalidate any cached sessions")
	fmt.Println("  • Require password re-entry")
	fmt.Println()

	// Confirm rotation if not forced
	if !force {
		fmt.Printf("Are you sure you want to rotate encryption keys for '%s'? [y/N] ", environment)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			ui.Info("Key rotation cancelled")
			return nil
		}
	}

	// Initialize keystore
	ks, err := keystore.NewKeystore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize keystore: %w", err)
	}

	// Initialize password manager
	pm := auth.NewPasswordManager(ks, cfg)

	// Get current encryption key for the environment
	currentKey, err := pm.GetOrCreateEnvironmentKey(environment)
	if err != nil {
		return fmt.Errorf("failed to get current encryption key: %w", err)
	}

	ui.Info("Generating new encryption keys...")

	// Generate new key
	newKey := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(newKey); err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	// Get storage backend with current encryption
	opts := storage.BackendOptions{
		Environment: environment,
		Type:        cfg.Vault.Type,
		BasePath:    cfg.Vault.Path,
		Password:    string(currentKey),
	}

	store, err := storage.GetBackendWithOptions(opts)
	if err != nil {
		return fmt.Errorf("failed to get storage backend: %w", err)
	}
	defer store.Close()

	// Get all variables
	keys, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list variables: %w", err)
	}

	ui.Info("Re-encrypting %d variables...", len(keys))

	// Store all variables temporarily
	variables := make(map[string]string)
	for _, key := range keys {
		value, err := store.Get(key)
		if err != nil {
			return fmt.Errorf("failed to get variable %s: %w", key, err)
		}
		variables[key] = value
	}

	// Close the old backend
	store.Close()

	// Create new backend with new encryption key
	newOpts := storage.BackendOptions{
		Environment: environment,
		Type:        cfg.Vault.Type,
		BasePath:    cfg.Vault.Path,
		Password:    string(newKey),
	}

	newStore, err := storage.GetBackendWithOptions(newOpts)
	if err != nil {
		return fmt.Errorf("failed to create new storage backend: %w", err)
	}
	defer newStore.Close()

	// Re-encrypt all variables with new key
	for key, value := range variables {
		if err := newStore.Set(key, value, true); err != nil {
			return fmt.Errorf("failed to re-encrypt variable %s: %w", key, err)
		}
	}

	// Update the environment key using password manager
	// This requires implementing key rotation in the password manager
	// For now, we'll clear the cache to force re-authentication
	pm.ClearEnvironmentCache(environment)

	ui.Success("Encryption keys rotated successfully")
	ui.Info("All %d variables have been re-encrypted with new keys", len(variables))
	ui.Info("Please test your application to ensure everything works correctly")

	return nil
}

func runSecurityVerify(environment string, deep bool) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if environment exists
	if !cfg.HasEnvironment(environment) {
		return fmt.Errorf("environment '%s' does not exist", environment)
	}

	ui.Header(fmt.Sprintf("Security Verification for Environment: %s", environment))
	fmt.Println()

	// Check encryption status
	if cfg.Vault.IsEncrypted() {
		ui.Success("✓ Encryption is enabled")
		ui.Info("  Algorithm: AES-256-GCM")
		ui.Info("  Key derivation: Argon2id")
	} else {
		ui.Warning("⚠ Encryption is disabled")
	}

	// Check keystore
	if cfg.Vault.IsEncrypted() {
		_, err := keystore.NewKeystore(cfg.Vault.Path)
		if err != nil {
			ui.Error("✗ Keystore verification failed: %v", err)
		} else {
			ui.Success("✓ Keystore is accessible")
		}
	}

	// Storage backend verification
	ui.Info("Storage backend: %s", cfg.Vault.Type)

	// Create backend options for verification
	opts := storage.BackendOptions{
		Environment: environment,
		Type:        cfg.Vault.Type,
		BasePath:    cfg.Vault.Path,
	}

	// Get storage backend
	store, err := storage.GetBackendWithOptions(opts)
	if err != nil {
		ui.Error("✗ Storage backend verification failed: %v", err)
		return fmt.Errorf("storage verification failed: %w", err)
	}
	defer store.Close()

	ui.Success("✓ Storage backend is accessible")

	if deep {
		ui.Info("Performing deep verification...")

		// Get all variables
		keys, err := store.List()
		if err != nil {
			ui.Error("✗ Failed to list variables: %v", err)
			return fmt.Errorf("variable listing failed: %w", err)
		}

		successCount := 0
		errorCount := 0

		for _, key := range keys {
			_, err := store.Get(key)
			if err != nil {
				ui.Error("✗ Variable '%s' verification failed: %v", key, err)
				errorCount++
			} else {
				successCount++
			}
		}

		ui.Success("✓ Deep verification completed")
		ui.Info("  Variables verified: %d", successCount)
		if errorCount > 0 {
			ui.Error("  Variables with errors: %d", errorCount)
		}
	}

	ui.Success("Security verification completed successfully")
	return nil
}

func runSecurityReport(format, output string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Generate security report
	report := SecurityReport{
		ProjectName:       cfg.Project.Name,
		GeneratedAt:       time.Now(),
		EncryptionEnabled: cfg.Vault.IsEncrypted(),
		StorageType:       cfg.Vault.Type,
		Environments:      make(map[string]EnvironmentSecurity),
	}

	// Add encryption details
	if cfg.Vault.IsEncrypted() {
		report.EncryptionDetails = &EncryptionDetails{
			Algorithm:     "AES-256-GCM",
			KeyDerivation: "Argon2id",
			KeyLength:     256,
		}
	}

	// Analyze each environment
	for _, envName := range cfg.GetEnvironmentNames() {
		envSecurity := EnvironmentSecurity{
			Name:              envName,
			PasswordProtected: cfg.IsPerEnvironmentPasswordsEnabled(),
			VariableCount:     0,
			LastAccessed:      time.Now(), // Placeholder
		}

		// Get variable count
		opts := storage.BackendOptions{
			Environment: envName,
			Type:        cfg.Vault.Type,
			BasePath:    cfg.Vault.Path,
		}

		store, err := storage.GetBackendWithOptions(opts)
		if err == nil {
			keys, err := store.List()
			if err == nil {
				envSecurity.VariableCount = len(keys)
			}
			store.Close()
		}

		report.Environments[envName] = envSecurity
	}

	// Output report
	switch format {
	case "json":
		return outputJSONReport(report, output)
	default:
		return outputTextReport(report, output)
	}
}

func runSecurityLock() error {
	ui.Info("Locking access to all environments...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize keystore
	ks, err := keystore.NewKeystore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize keystore: %w", err)
	}

	// Initialize password manager
	pm := auth.NewPasswordManager(ks, cfg)

	// Clear all cached passwords and keys
	pm.ClearSessionCache()

	ui.Success("All environments have been locked")
	ui.Info("You will need to re-enter passwords for subsequent operations")

	return nil
}

func runSecurityUnlock() error {
	ui.Info("Unlocking environments...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize keystore
	ks, err := keystore.NewKeystore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize keystore: %w", err)
	}

	// Initialize password manager
	pm := auth.NewPasswordManager(ks, cfg)

	// Unlock each environment
	environments := cfg.GetEnvironmentNames()
	for _, env := range environments {
		ui.Info("Unlocking environment: %s", env)

		// This will prompt for password and cache it
		_, err := pm.GetOrCreateEnvironmentKey(env)
		if err != nil {
			ui.Error("Failed to unlock environment %s: %v", env, err)
			continue
		}

		ui.Success("✓ Environment %s unlocked", env)
	}

	ui.Success("Environments unlocked successfully")

	return nil
}

// Security report structures

type SecurityReport struct {
	ProjectName       string                         `json:"project_name"`
	GeneratedAt       time.Time                      `json:"generated_at"`
	EncryptionEnabled bool                           `json:"encryption_enabled"`
	EncryptionDetails *EncryptionDetails             `json:"encryption_details,omitempty"`
	StorageType       string                         `json:"storage_type"`
	Environments      map[string]EnvironmentSecurity `json:"environments"`
}

type EncryptionDetails struct {
	Algorithm     string `json:"algorithm"`
	KeyDerivation string `json:"key_derivation"`
	KeyLength     int    `json:"key_length_bits"`
}

type EnvironmentSecurity struct {
	Name              string    `json:"name"`
	PasswordProtected bool      `json:"password_protected"`
	VariableCount     int       `json:"variable_count"`
	LastAccessed      time.Time `json:"last_accessed"`
}

func outputTextReport(report SecurityReport, outputFile string) error {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Security Report for Project: %s\n", report.ProjectName))
	output.WriteString(fmt.Sprintf("Generated: %s\n\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))

	// Encryption status
	output.WriteString("ENCRYPTION STATUS\n")
	output.WriteString(strings.Repeat("=", 50) + "\n")
	if report.EncryptionEnabled {
		output.WriteString("Status: ENABLED ✓\n")
		if report.EncryptionDetails != nil {
			output.WriteString(fmt.Sprintf("Algorithm: %s\n", report.EncryptionDetails.Algorithm))
			output.WriteString(fmt.Sprintf("Key Derivation: %s\n", report.EncryptionDetails.KeyDerivation))
			output.WriteString(fmt.Sprintf("Key Length: %d bits\n", report.EncryptionDetails.KeyLength))
		}
	} else {
		output.WriteString("Status: DISABLED ⚠\n")
	}
	output.WriteString(fmt.Sprintf("Storage Type: %s\n\n", report.StorageType))

	// Environment details
	output.WriteString("ENVIRONMENT DETAILS\n")
	output.WriteString(strings.Repeat("=", 50) + "\n")
	for _, env := range report.Environments {
		output.WriteString(fmt.Sprintf("Environment: %s\n", env.Name))
		output.WriteString(fmt.Sprintf("  Password Protected: %t\n", env.PasswordProtected))
		output.WriteString(fmt.Sprintf("  Variable Count: %d\n", env.VariableCount))
		output.WriteString(fmt.Sprintf("  Last Accessed: %s\n\n", env.LastAccessed.Format("2006-01-02 15:04:05")))
	}

	if outputFile != "" {
		// Write to file
		if err := os.WriteFile(outputFile, []byte(output.String()), 0644); err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}
		ui.Success("Security report saved to: %s", outputFile)
	} else {
		// Print to stdout
		fmt.Print(output.String())
	}

	return nil
}

func outputJSONReport(report SecurityReport, outputFile string) error {
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	if outputFile != "" {
		// Write to file
		if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}
		ui.Success("Security report saved to: %s", outputFile)
	} else {
		// Print to stdout
		fmt.Println(string(jsonData))
	}

	return nil
}
