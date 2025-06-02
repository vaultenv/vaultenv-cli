package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/sync"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

// Change represents a detected change in vault files
type VaultChange struct {
	Type        string // "added", "modified", "deleted"
	Environment string
	Variable    string
	FilePath    string
}

func newGitPushCommand() *cobra.Command {
	var (
		message     string
		force       bool
		environment string
		autoCommit  bool
	)

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push encrypted variables to git repository",
		Long: `Push your local VaultEnv changes to the git repository. Variables are
encrypted before committing, ensuring secrets remain secure in version control.`,
		Example: `  # Push current environment
  vaultenv git push

  # Push with custom message
  vaultenv git push --message "Add API keys for payment service"

  # Push specific environment
  vaultenv git push --env production

  # Force push (overwrites remote changes)
  vaultenv git push --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(message, force, environment, autoCommit)
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "",
		"Commit message (auto-generated if not provided)")
	cmd.Flags().BoolVarP(&force, "force", "f", false,
		"Force push, overwriting remote changes")
	cmd.Flags().StringVarP(&environment, "env", "e", "",
		"Push specific environment only")
	cmd.Flags().BoolVar(&autoCommit, "auto-commit", true,
		"Automatically commit changes before pushing")

	return cmd
}

func newGitPullCommand() *cobra.Command {
	var (
		autoMerge   bool
		strategy    string
		environment string
	)

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull encrypted variables from git repository",
		Long: `Pull the latest VaultEnv changes from the git repository. Automatically
handles decryption and merging of changes.`,
		Example: `  # Pull latest changes
  vaultenv git pull

  # Pull and auto-merge conflicts
  vaultenv git pull --auto-merge

  # Pull with specific merge strategy
  vaultenv git pull --strategy=theirs

  # Pull specific environment
  vaultenv git pull --env production`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(autoMerge, strategy, environment)
		},
	}

	cmd.Flags().BoolVar(&autoMerge, "auto-merge", false,
		"Automatically resolve conflicts")
	cmd.Flags().StringVar(&strategy, "strategy", "prompt",
		"Merge strategy: prompt, ours, theirs, newest")
	cmd.Flags().StringVarP(&environment, "env", "e", "",
		"Pull specific environment only")

	return cmd
}

func runPush(message string, force bool, environment string, autoCommit bool) error {
	// Check if we're in a git repository
	if err := checkGitRepo(); err != nil {
		return fmt.Errorf("not in a git repository, run 'vaultenv git init' first")
	}

	// Get current configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check for uncommitted changes in vault files
	changes, err := detectVaultChanges()
	if err != nil {
		return fmt.Errorf("failed to detect changes: %w", err)
	}

	if len(changes) == 0 {
		ui.Info("No changes to push")
		return nil
	}

	// Show what will be pushed
	ui.Info("Changes to push:")
	for _, change := range changes {
		ui.Info("  %s %s in %s", change.Type, change.Variable, change.Environment)
	}

	// Check for remote changes first (unless forcing)
	if !force {
		if hasRemoteChanges() {
			ui.Error("Remote changes detected. Pull first or use --force")
			return fmt.Errorf("remote changes exist")
		}
	}

	// If autoCommit is enabled, stage and commit changes
	if autoCommit {
		// Stage changes
		if err := ui.StartProgress("Staging changes", func() error {
			return stageVaultChanges(changes)
		}); err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}

		// Generate commit message if not provided
		if message == "" {
			message = generateCommitMessage(changes)
		}

		// Commit changes
		if err := gitCommit(message); err != nil {
			return fmt.Errorf("failed to commit: %w", err)
		}
	}

	// Push to remote
	if err := ui.StartProgress("Pushing to remote", func() error {
		return gitPush(force)
	}); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	ui.Success("Successfully pushed %d changes", len(changes))

	// Update config with sync timestamp
	if cfg.Git.AutoPush {
		// Mark sync timestamp
		updateSyncTimestamp()
	}

	return nil
}

func runPull(autoMerge bool, strategy string, environment string) error {
	// Check if we're in a git repository
	if err := checkGitRepo(); err != nil {
		return fmt.Errorf("not in a git repository")
	}

	// Fetch latest changes from remote
	if err := ui.StartProgress("Fetching remote changes", gitFetch); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// Check for local uncommitted changes
	if hasLocalChanges() {
		ui.Warning("You have uncommitted local changes")
		if !ui.Confirm("Continue with pull? (changes may conflict)") {
			return fmt.Errorf("aborted by user")
		}
	}

	// Pull changes
	if err := ui.StartProgress("Pulling changes", gitPull); err != nil {
		// Check if it's a merge conflict
		if strings.Contains(err.Error(), "conflict") {
			return handleMergeConflicts(autoMerge, strategy, environment)
		}
		return fmt.Errorf("failed to pull: %w", err)
	}

	// Decrypt and update local vault
	if err := ui.StartProgress("Updating local vault", func() error {
		return updateLocalVault(environment)
	}); err != nil {
		return fmt.Errorf("failed to update local vault: %w", err)
	}

	ui.Success("Successfully pulled latest changes")
	return nil
}

// detectVaultChanges scans for changes in vault files
func detectVaultChanges() ([]VaultChange, error) {
	var changes []VaultChange

	// Run git status on .vaultenv directory
	cmd := exec.Command("git", "status", "--porcelain", ".vaultenv/")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse git status line
		if len(line) < 3 {
			continue
		}

		status := line[:2]
		filePath := strings.TrimSpace(line[3:])

		// Determine change type
		changeType := ""
		switch {
		case strings.Contains(status, "A") || strings.Contains(status, "?"):
			changeType = "added"
		case strings.Contains(status, "M"):
			changeType = "modified"
		case strings.Contains(status, "D"):
			changeType = "deleted"
		default:
			continue
		}

		// Extract environment and variable from path
		// Expected format: .vaultenv/git/<environment>/<namespace>/<variable>.env
		parts := strings.Split(filePath, "/")
		if len(parts) >= 4 && parts[0] == ".vaultenv" && parts[1] == "git" {
			environment := parts[2]
			variable := strings.TrimSuffix(filepath.Base(filePath), ".env")

			changes = append(changes, VaultChange{
				Type:        changeType,
				Environment: environment,
				Variable:    variable,
				FilePath:    filePath,
			})
		}
	}

	return changes, nil
}

// hasRemoteChanges checks if there are unpulled changes
func hasRemoteChanges() bool {
	// Check if local branch is behind remote
	cmd := exec.Command("git", "rev-list", "--count", "HEAD..@{u}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	count := strings.TrimSpace(string(output))
	return count != "0"
}

// hasLocalChanges checks for uncommitted local changes
func hasLocalChanges() bool {
	cmd := exec.Command("git", "status", "--porcelain", ".vaultenv/")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}

// stageVaultChanges stages all vault file changes
func stageVaultChanges(changes []VaultChange) error {
	for _, change := range changes {
		cmd := exec.Command("git", "add", change.FilePath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stage %s: %w", change.FilePath, err)
		}
	}
	return nil
}

// generateCommitMessage creates an automatic commit message
func generateCommitMessage(changes []VaultChange) string {
	// Count changes by type
	added, modified, deleted := 0, 0, 0
	environments := make(map[string]bool)

	for _, change := range changes {
		environments[change.Environment] = true
		switch change.Type {
		case "added":
			added++
		case "modified":
			modified++
		case "deleted":
			deleted++
		}
	}

	// Build message
	parts := []string{}
	if added > 0 {
		parts = append(parts, fmt.Sprintf("add %d", added))
	}
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("update %d", modified))
	}
	if deleted > 0 {
		parts = append(parts, fmt.Sprintf("remove %d", deleted))
	}

	envList := []string{}
	for env := range environments {
		envList = append(envList, env)
	}

	return fmt.Sprintf("vault: %s variables in %s",
		strings.Join(parts, ", "),
		strings.Join(envList, ", "))
}

// Git operations
func gitCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	return cmd.Run()
}

func gitPush(force bool) error {
	args := []string{"push"}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.Command("git", args...)
	return cmd.Run()
}

func gitFetch() error {
	cmd := exec.Command("git", "fetch")
	return cmd.Run()
}

func gitPull() error {
	cmd := exec.Command("git", "pull")
	return cmd.Run()
}

// updateLocalVault decrypts pulled files and updates local vault
func updateLocalVault(environment string) error {
	// TODO: Implement decryption and update logic
	// This will decrypt the pulled files and update the local storage
	return nil
}

// handleMergeConflicts handles git merge conflicts
func handleMergeConflicts(autoMerge bool, strategy string, environment string) error {
	ui.Warning("Merge conflicts detected")
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// Create storage backend
	stor, err := storage.GetBackendWithOptions(storage.BackendOptions{
		Type:        cfg.Vault.Type,
		BasePath:    cfg.Vault.Path,
		Environment: environment,
	})
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	
	// Create conflict detector
	detector := sync.NewGitConflictDetector(stor, cfg)
	
	// Detect conflicts
	conflicts, err := detector.DetectConflicts()
	if err != nil {
		return fmt.Errorf("failed to detect conflicts: %w", err)
	}
	
	if len(conflicts) == 0 {
		ui.Info("No conflicts found")
		return nil
	}
	
	ui.Info("Found %d conflicts", len(conflicts))
	
	// Filter by environment if specified
	if environment != "" {
		var filtered []sync.Conflict
		for _, c := range conflicts {
			if c.Environment == environment {
				filtered = append(filtered, c)
			}
		}
		conflicts = filtered
		ui.Info("Filtered to %d conflicts in %s environment", len(conflicts), environment)
	}
	
	// Determine strategy
	conflictStrategy := sync.ConflictStrategy(strategy)
	if !autoMerge {
		conflictStrategy = sync.StrategyPrompt
	}
	
	// Create conflict set
	conflictSet := &sync.ConflictSet{
		Conflicts: conflicts,
		Strategy:  conflictStrategy,
	}
	
	// Resolve conflicts
	resolutions, err := conflictSet.ResolveAll()
	if err != nil {
		return fmt.Errorf("failed to resolve conflicts: %w", err)
	}
	
	// Apply resolutions
	for key, resolution := range resolutions {
		parts := strings.Split(key, "/")
		if len(parts) != 2 {
			continue
		}
		
		env := parts[0]
		variable := parts[1]
		
		// Build file path
		filePath := filepath.Join(".vaultenv", "git", env, variable+".env")
		
		// Write resolved value
		if err := detector.ResolveConflictFile(filePath, resolution.Value); err != nil {
			ui.Error("Failed to write resolution for %s: %v", key, err)
			continue
		}
		
		// Stage the resolved file
		cmd := exec.Command("git", "add", filePath)
		if err := cmd.Run(); err != nil {
			ui.Error("Failed to stage resolved file %s: %v", filePath, err)
		}
	}
	
	// Show summary
	ui.Success(conflictSet.Summary(resolutions))
	
	// If all conflicts resolved, suggest committing
	if len(resolutions) == len(conflicts) {
		ui.Info("All conflicts resolved. Run 'git commit' to complete the merge.")
	}
	
	return nil
}

// updateSyncTimestamp updates the last sync timestamp
func updateSyncTimestamp() {
	// TODO: Store sync timestamp in config or metadata
}