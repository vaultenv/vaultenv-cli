package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/internal/auth"
	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/keystore"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func newHistoryCommand() *cobra.Command {
	var (
		environment string
		limit       int
	)

	cmd := &cobra.Command{
		Use:   "history KEY",
		Short: "Show history of changes for a variable",
		Long:  `Display the change history for a specific variable including previous values and who made changes.`,

		Example: `  # Show history for DATABASE_URL
  vaultenv history DATABASE_URL
  
  # Show last 5 changes
  vaultenv history API_KEY --limit 5
  
  # Show history in production
  vaultenv history DATABASE_URL --env production`,

		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistory(args[0], environment, limit)
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "development", "environment to use")
	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "number of history entries to show")

	return cmd
}

func newRestoreCommand() *cobra.Command {
	var (
		environment string
		version     int
		timestamp   string
		force       bool
	)

	cmd := &cobra.Command{
		Use:   "restore KEY",
		Short: "Restore a variable from history",
		Long:  `Restore a variable to a previous value from its history.`,

		Example: `  # Restore to version 3
  vaultenv restore DATABASE_URL --version 3
  
  # Restore to specific timestamp
  vaultenv restore API_KEY --timestamp "2024-01-20 10:30:00"
  
  # Force restore without confirmation
  vaultenv restore DATABASE_URL --version 2 --force
  
  # Restore in production environment
  vaultenv restore DATABASE_URL --version 1 --env production`,

		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(args[0], environment, version, timestamp, force)
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "development", "environment to use")
	cmd.Flags().IntVarP(&version, "version", "v", 0, "version number to restore to")
	cmd.Flags().StringVarP(&timestamp, "timestamp", "t", "", "timestamp to restore to (YYYY-MM-DD HH:MM:SS)")
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")

	return cmd
}

func runHistory(key, environment string, limit int) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

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
			return fmt.Errorf("failed to initialize keystore: %w", err)
		}

		pm := auth.NewPasswordManager(ks, cfg)

		// Get or create encryption key
		key, err := pm.GetOrCreateMasterKey(cfg.Project.ID)
		if err != nil {
			return fmt.Errorf("failed to get encryption key: %w", err)
		}

		// Convert key to string for storage options
		opts.Password = string(key)
	}

	// Get storage backend
	store, err := storage.GetBackendWithOptions(opts)
	if err != nil {
		return fmt.Errorf("failed to get storage backend: %w", err)
	}
	defer store.Close()

	// Check if backend supports history
	historyBackend, ok := store.(storage.HistoryBackend)
	if !ok {
		return fmt.Errorf("current storage backend (%s) does not support history", cfg.Vault.Type)
	}

	// Get history
	history, err := historyBackend.GetHistory(key, limit)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	if len(history) == 0 {
		ui.Info("No history found for %s", key)
		return nil
	}

	// Display history
	ui.Header(fmt.Sprintf("History for %s in %s", key, environment))

	for _, h := range history {
		fmt.Printf("\n● Version %d\n", h.Version)
		fmt.Printf("  Changed: %s\n", h.ChangedAt.Format(time.RFC3339))
		fmt.Printf("  By: %s\n", h.ChangedBy)
		fmt.Printf("  Action: %s\n", h.ChangeType)

		if h.ChangeType != "DELETE" {
			// Show value preview (truncated for security)
			preview := h.Value
			if len(preview) > 50 {
				preview = preview[:47] + "..."
			}
			fmt.Printf("  Value: %s\n", preview)
		}
	}

	return nil
}

func newAuditCommand() *cobra.Command {
	var (
		environment string
		limit       int
		user        string
		action      string
	)

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Show audit log of operations",
		Long:  `Display the audit log showing all operations performed on variables including who made changes and when.`,

		Example: `  # Show recent audit entries
  vaultenv audit
  
  # Show last 20 audit entries
  vaultenv audit --limit 20
  
  # Show audit for production
  vaultenv audit --env production
  
  # Filter by user
  vaultenv audit --user john
  
  # Filter by action type
  vaultenv audit --action SET`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runAudit(environment, limit, user, action)
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "development", "environment to use")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "number of audit entries to show")
	cmd.Flags().StringVarP(&user, "user", "u", "", "filter by user")
	cmd.Flags().StringVarP(&action, "action", "a", "", "filter by action (GET, SET, DELETE)")

	return cmd
}

func runAudit(environment string, limit int, filterUser, filterAction string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

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
			return fmt.Errorf("failed to initialize keystore: %w", err)
		}

		pm := auth.NewPasswordManager(ks, cfg)

		// Get or create encryption key
		key, err := pm.GetOrCreateMasterKey(cfg.Project.ID)
		if err != nil {
			return fmt.Errorf("failed to get encryption key: %w", err)
		}

		// Convert key to string for storage options
		opts.Password = string(key)
	}

	// Get storage backend
	store, err := storage.GetBackendWithOptions(opts)
	if err != nil {
		return fmt.Errorf("failed to get storage backend: %w", err)
	}
	defer store.Close()

	// Check if backend supports audit
	historyBackend, ok := store.(storage.HistoryBackend)
	if !ok {
		return fmt.Errorf("current storage backend (%s) does not support audit logging", cfg.Vault.Type)
	}

	// Get audit log
	entries, err := historyBackend.GetAuditLog(limit)
	if err != nil {
		return fmt.Errorf("failed to get audit log: %w", err)
	}

	// Filter entries if requested
	var filtered []storage.AuditEntry
	for _, entry := range entries {
		if filterUser != "" && entry.User != filterUser {
			continue
		}
		if filterAction != "" && entry.Action != filterAction {
			continue
		}
		filtered = append(filtered, entry)
	}

	if len(filtered) == 0 {
		ui.Info("No audit entries found matching filters")
		return nil
	}

	// Display audit log
	ui.Header(fmt.Sprintf("Audit Log for %s", environment))

	for _, entry := range filtered {
		// Simple display without color codes for action
		fmt.Printf("\n● %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Action: %s\n", entry.Action)
		if entry.Key != "" {
			fmt.Printf("  Key: %s\n", entry.Key)
		}
		fmt.Printf("  User: %s\n", entry.User)

		if !entry.Success {
			fmt.Printf("  Status: Failed\n")
			if entry.ErrorMessage != "" {
				fmt.Printf("  Error: %s\n", entry.ErrorMessage)
			}
		}
	}

	return nil
}

func runRestore(key, environment string, version int, timestamp string, force bool) error {
	// Validate inputs
	if version == 0 && timestamp == "" {
		return fmt.Errorf("either --version or --timestamp must be specified")
	}
	if version != 0 && timestamp != "" {
		return fmt.Errorf("cannot specify both --version and --timestamp")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

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
			return fmt.Errorf("failed to initialize keystore: %w", err)
		}

		pm := auth.NewPasswordManager(ks, cfg)

		// Get or create encryption key
		encKey, err := pm.GetOrCreateMasterKey(cfg.Project.ID)
		if err != nil {
			return fmt.Errorf("failed to get encryption key: %w", err)
		}

		// Convert key to string for storage options
		opts.Password = string(encKey)
	}

	// Get storage backend
	store, err := storage.GetBackendWithOptions(opts)
	if err != nil {
		return fmt.Errorf("failed to get storage backend: %w", err)
	}
	defer store.Close()

	// Check if backend supports history
	historyBackend, ok := store.(storage.HistoryBackend)
	if !ok {
		return fmt.Errorf("current storage backend (%s) does not support history", cfg.Vault.Type)
	}

	// Get history for the key
	history, err := historyBackend.GetHistory(key, 100) // Get more history to find the right version
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	if len(history) == 0 {
		return fmt.Errorf("no history found for key '%s'", key)
	}

	// Find the target history entry
	var targetEntry *storage.SecretHistory
	if version != 0 {
		// Find by version
		for _, h := range history {
			if h.Version == version {
				targetEntry = &h
				break
			}
		}
		if targetEntry == nil {
			return fmt.Errorf("version %d not found in history for key '%s'", version, key)
		}
	} else {
		// Find by timestamp
		targetTime, err := time.Parse("2006-01-02 15:04:05", timestamp)
		if err != nil {
			return fmt.Errorf("invalid timestamp format. Use YYYY-MM-DD HH:MM:SS")
		}

		// Find the closest entry at or before the target time
		for _, h := range history {
			if h.ChangedAt.Before(targetTime) || h.ChangedAt.Equal(targetTime) {
				targetEntry = &h
				break
			}
		}
		if targetEntry == nil {
			return fmt.Errorf("no history entry found at or before timestamp '%s'", timestamp)
		}
	}

	// Check if the target entry is a DELETE operation
	if targetEntry.ChangeType == "DELETE" {
		return fmt.Errorf("cannot restore to a DELETE operation (version %d)", targetEntry.Version)
	}

	// Get current value for comparison
	currentValue, err := store.Get(key)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("failed to get current value: %w", err)
	}

	// Check if value would actually change
	if currentValue == targetEntry.Value {
		ui.Info("Variable '%s' already has the target value", key)
		return nil
	}

	// Show preview of what will be restored
	ui.Header(fmt.Sprintf("Restore Preview for '%s'", key))
	fmt.Printf("Environment: %s\n", environment)
	fmt.Printf("Target Version: %d\n", targetEntry.Version)
	fmt.Printf("Target Timestamp: %s\n", targetEntry.ChangedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Changed By: %s\n", targetEntry.ChangedBy)
	fmt.Println()

	// Show value preview (truncated for security)
	preview := targetEntry.Value
	if len(preview) > 100 {
		preview = preview[:97] + "..."
	}
	fmt.Printf("Value to restore: %s\n", preview)

	// Confirm restore if not forced
	if !force {
		fmt.Printf("\nAre you sure you want to restore '%s' to version %d? [y/N] ", key, targetEntry.Version)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			ui.Info("Restore cancelled")
			return nil
		}
	}

	// Perform the restore
	err = store.Set(key, targetEntry.Value, cfg.Vault.IsEncrypted())
	if err != nil {
		return fmt.Errorf("failed to restore variable: %w", err)
	}

	ui.Success("Successfully restored '%s' to version %d", key, targetEntry.Version)
	ui.Info("Original timestamp: %s", targetEntry.ChangedAt.Format("2006-01-02 15:04:05"))

	return nil
}
