package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/internal/auth"
	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/keystore"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func newMigrateCommand() *cobra.Command {
	var (
		fromType    string
		toType      string
		environment string
		force       bool
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate storage backend",
		Long:  `Migrate variables from one storage backend to another, preserving all data and encryption.`,

		Example: `  # Migrate from file to SQLite storage
  vaultenv migrate --from file --to sqlite
  
  # Migrate specific environment
  vaultenv migrate --from file --to sqlite --env production
  
  # Dry run to see what would be migrated
  vaultenv migrate --from file --to sqlite --dry-run
  
  # Force migration without confirmation
  vaultenv migrate --from file --to sqlite --force`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrate(fromType, toType, environment, force, dryRun)
		},
	}

	cmd.Flags().StringVar(&fromType, "from", "", "source storage type (file, sqlite)")
	cmd.Flags().StringVar(&toType, "to", "", "destination storage type (file, sqlite)")
	cmd.Flags().StringVarP(&environment, "env", "e", "development", "environment to migrate")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be migrated without making changes")

	cmd.MarkFlagRequired("from")
	cmd.MarkFlagRequired("to")

	return cmd
}

func runMigrate(fromType, toType, environment string, force, dryRun bool) error {
	// Validate migration path
	if fromType == toType {
		return fmt.Errorf("source and destination storage types must be different")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Show migration plan
	ui.Header("Migration Plan")
	fmt.Printf("  From: %s storage\n", fromType)
	fmt.Printf("  To: %s storage\n", toType)
	fmt.Printf("  Environment: %s\n", environment)
	fmt.Printf("  Project: %s\n", cfg.Project.Name)

	if dryRun {
		ui.Info("\n🔍 DRY RUN MODE - No changes will be made")
	}

	// Confirm migration
	if !force && !dryRun {
		fmt.Println()
		// Simple confirmation without external dependency
		fmt.Print("This will migrate all variables. Continue? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			ui.Info("Migration cancelled")
			return nil
		}
	}

	// Check if we're in a test environment
	isTest := isTestEnvironment()

	// Create source backend
	sourceOpts := storage.BackendOptions{
		Environment: environment,
		Type:        fromType,
		BasePath:    cfg.Vault.Path,
	}

	// Handle authentication if not in test mode
	var password string
	if !isTest && cfg.Vault.IsEncrypted() {
		ks, err := keystore.NewKeystore(cfg.Vault.Path)
		if err != nil {
			return fmt.Errorf("failed to initialize keystore: %w", err)
		}
		
		pm := auth.NewPasswordManager(ks)

		// Get or create encryption key
		key, err := pm.GetOrCreateMasterKey(cfg.Project.ID)
		if err != nil {
			return fmt.Errorf("failed to get encryption key: %w", err)
		}
		
		// Convert key to string for storage options
		password = string(key)
		sourceOpts.Password = password
	}

	// Get source backend
	source, err := storage.GetBackendWithOptions(sourceOpts)
	if err != nil {
		return fmt.Errorf("failed to create source backend: %w", err)
	}
	defer source.Close()

	// List all variables from source
	ui.Info("\nReading variables from %s storage...", fromType)
	keys, err := source.List()
	if err != nil {
		return fmt.Errorf("failed to list variables: %w", err)
	}

	if len(keys) == 0 {
		ui.Warning("No variables found to migrate")
		return nil
	}

	ui.Success("Found %d variables to migrate", len(keys))

	// Read all variables
	variables := make(map[string]string)
	for _, key := range keys {
		value, err := source.Get(key)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", key, err)
		}
		variables[key] = value
	}

	if dryRun {
		// Show what would be migrated
		ui.Info("\nVariables that would be migrated:")
		for _, key := range keys {
			fmt.Printf("  • %s\n", key)
		}
		return nil
	}

	// Create destination backend
	destOpts := storage.BackendOptions{
		Environment: environment,
		Type:        toType,
		BasePath:    cfg.Vault.Path,
		Password:    password,
	}

	dest, err := storage.GetBackendWithOptions(destOpts)
	if err != nil {
		return fmt.Errorf("failed to create destination backend: %w", err)
	}
	defer dest.Close()

	// Migrate variables
	ui.Info("\nMigrating variables to %s storage...", toType)
	
	migrated := 0
	for key, value := range variables {
		fmt.Printf("  Migrating %s...\n", key)
		
		// Set in destination (encryption will be handled by the backend)
		if err := dest.Set(key, value, false); err != nil {
			return fmt.Errorf("failed to migrate %s: %w", key, err)
		}
		
		migrated++
	}

	// Update configuration to use new storage type
	cfg.Vault.Type = toType
	if err := cfg.Save(); err != nil {
		ui.Warning("Migration completed but failed to update config: %v", err)
		ui.Info("Please manually update .vaultenv/config.yaml to set vault.type: %s", toType)
	} else {
		ui.Success("Updated configuration to use %s storage", toType)
	}

	ui.Success("\n✅ Successfully migrated %d variables from %s to %s storage", 
		migrated, fromType, toType)

	// Show additional information for SQLite
	if toType == "sqlite" {
		ui.Info("\nSQLite features now available:")
		fmt.Println("  • Version history tracking")
		fmt.Println("  • Audit logging")
		fmt.Println("  • Better performance for large datasets")
		fmt.Println("\nTry these commands:")
		fmt.Printf("  vaultenv history <KEY> --env %s\n", environment)
		fmt.Printf("  vaultenv audit --env %s\n", environment)
	}

	return nil
}