package migration

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/keystore"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
)

// PasswordMigrationManager handles migration from single-password to per-environment passwords
type PasswordMigrationManager struct {
	keystore *keystore.Keystore
}

// NewPasswordMigrationManager creates a new migration manager
func NewPasswordMigrationManager(ks *keystore.Keystore) *PasswordMigrationManager {
	return &PasswordMigrationManager{
		keystore: ks,
	}
}

// MigrateToPerEnvironmentPasswords detects legacy projects and offers migration
func (pmm *PasswordMigrationManager) MigrateToPerEnvironmentPasswords(projectConfig *config.Config) error {
	// Check if this is a legacy project (version < 2.0.0 or no per_environment_passwords setting)
	if pmm.isLegacyProject(projectConfig) {
		ui.Info("This project uses a single password for all environments.")
		ui.Info("VaultEnv now supports per-environment passwords for better security.")
		ui.Info("This allows different team members to have access to different environments.")
		
		if promptConfirm("Would you like to migrate to per-environment passwords?") {
			return pmm.performPasswordMigration(projectConfig)
		} else {
			ui.Info("Skipping migration. You can migrate later using 'vaultenv migrate passwords'")
			return nil
		}
	}
	
	return nil
}

// ForceMigration performs migration without asking for confirmation
func (pmm *PasswordMigrationManager) ForceMigration(projectConfig *config.Config) error {
	if !pmm.isLegacyProject(projectConfig) {
		return fmt.Errorf("project is already using per-environment passwords")
	}
	
	return pmm.performPasswordMigration(projectConfig)
}

// isLegacyProject determines if a project needs migration
func (pmm *PasswordMigrationManager) isLegacyProject(cfg *config.Config) bool {
	// If project config version is less than 2.0.0, it's legacy
	if cfg.Version < "2.0.0" {
		return true
	}
	
	// If per_environment_passwords is explicitly set to false, it's legacy
	if !cfg.Security.PerEnvironmentPasswords {
		return true
	}
	
	// Check if we have any project-level keys (old format) but no environment-level keys
	hasProjectKey := pmm.hasProjectLevelKey(cfg.Project.ID)
	hasEnvironmentKeys := pmm.hasEnvironmentKeys(cfg.Project.ID)
	
	return hasProjectKey && !hasEnvironmentKeys
}

// hasProjectLevelKey checks if there's a legacy project-level key
func (pmm *PasswordMigrationManager) hasProjectLevelKey(projectID string) bool {
	_, err := pmm.keystore.GetKey(projectID)
	return err == nil
}

// hasEnvironmentKeys checks if there are any environment-specific keys
func (pmm *PasswordMigrationManager) hasEnvironmentKeys(projectID string) bool {
	environments, err := pmm.keystore.ListEnvironments(projectID)
	return err == nil && len(environments) > 0
}

// performPasswordMigration executes the migration process
func (pmm *PasswordMigrationManager) performPasswordMigration(cfg *config.Config) error {
	ui.Info("Migrating to per-environment passwords...")
	
	// Get the existing project key
	projectKey, err := pmm.keystore.GetKey(cfg.Project.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve existing project key: %w", err)
	}
	
	// Get list of environments from config
	environments := cfg.GetEnvironmentNames()
	if len(environments) == 0 {
		// Default environments if none are configured
		environments = []string{"development", "staging", "production"}
		ui.Warning("No environments found in config, using defaults: %v", environments)
	}
	
	ui.Info("Creating environment-specific keys for: %v", environments)
	
	// For each environment, create an environment-specific key with the same settings
	for _, env := range environments {
		envKey := &keystore.EnvironmentKeyEntry{
			ProjectID:        cfg.Project.ID,
			Environment:      env,
			Salt:             projectKey.Salt,
			VerificationHash: projectKey.VerificationHash,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			Algorithm:        "argon2id", // Default algorithm
			Iterations:       3,          // Default iterations
			Memory:           64 * 1024,  // Default memory (64MB)
			Parallelism:      4,          // Default parallelism
		}
		
		if err := pmm.keystore.StoreEnvironmentKey(cfg.Project.ID, env, envKey); err != nil {
			return fmt.Errorf("failed to create key for environment %s: %w", env, err)
		}
	}
	
	// Update config to enable per-environment passwords
	cfg.Version = "2.0.0"
	cfg.Security.PerEnvironmentPasswords = true
	
	// Save updated config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save updated config: %w", err)
	}
	ui.Success("Successfully migrated to per-environment passwords!")
	ui.Info("Each environment now has its own password:")
	
	for _, env := range environments {
		ui.Info("  - %s: Uses the same password as before", env)
	}
	
	ui.Info("You can now change individual environment passwords using:")
	ui.Info("  vaultenv env change-password <environment>")
	
	return nil
}

// MigrationStatus provides information about migration status
type MigrationStatus struct {
	IsLegacy            bool
	HasProjectKey       bool
	HasEnvironmentKeys  bool
	Environments        []string
	RequiresMigration   bool
}

// GetMigrationStatus returns the current migration status
func (pmm *PasswordMigrationManager) GetMigrationStatus(cfg *config.Config) (*MigrationStatus, error) {
	hasProjectKey := pmm.hasProjectLevelKey(cfg.Project.ID)
	hasEnvironmentKeys := pmm.hasEnvironmentKeys(cfg.Project.ID)
	isLegacy := pmm.isLegacyProject(cfg)
	
	environments, err := pmm.keystore.ListEnvironments(cfg.Project.ID)
	if err != nil {
		environments = []string{}
	}
	
	status := &MigrationStatus{
		IsLegacy:           isLegacy,
		HasProjectKey:      hasProjectKey,
		HasEnvironmentKeys: hasEnvironmentKeys,
		Environments:       environments,
		RequiresMigration:  isLegacy && hasProjectKey && !hasEnvironmentKeys,
	}
	
	return status, nil
}

// CleanupLegacyKeys removes old project-level keys after successful migration
func (pmm *PasswordMigrationManager) CleanupLegacyKeys(projectID string) error {
	// Only cleanup if we have environment keys (migration was successful)
	environments, err := pmm.keystore.ListEnvironments(projectID)
	if err != nil || len(environments) == 0 {
		return fmt.Errorf("cannot cleanup: no environment keys found")
	}
	
	ui.Info("Cleaning up legacy project-level key...")
	
	if err := pmm.keystore.DeleteKey(projectID); err != nil {
		// Don't fail if key doesn't exist
		if err != keystore.ErrKeyNotFound {
			return fmt.Errorf("failed to cleanup legacy key: %w", err)
		}
	}
	
	ui.Success("Legacy project key removed successfully")
	return nil
}

// ValidateMigration ensures migration was successful
func (pmm *PasswordMigrationManager) ValidateMigration(cfg *config.Config) error {
	status, err := pmm.GetMigrationStatus(cfg)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}
	
	if status.RequiresMigration {
		return fmt.Errorf("migration validation failed: project still requires migration")
	}
	
	if !status.HasEnvironmentKeys {
		return fmt.Errorf("migration validation failed: no environment keys found")
	}
	
	// Verify each environment has a valid key
	environments := cfg.GetEnvironmentNames()
	for _, env := range environments {
		_, err := pmm.keystore.GetEnvironmentKey(cfg.Project.ID, env)
		if err != nil {
			return fmt.Errorf("migration validation failed: no key found for environment %s", env)
		}
	}
	
	ui.Success("Migration validation passed: all environments have valid keys")
	return nil
}

// promptConfirm prompts the user for a yes/no confirmation
func promptConfirm(message string) bool {
	fmt.Printf("%s (y/N): ", message)
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}