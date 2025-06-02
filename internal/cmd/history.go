package cmd

import (
	"fmt"
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