package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"github.com/vaultenv/vaultenv-cli/pkg/dotenv"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

// newLoadCommand creates the load command for importing .env files
func newLoadCommand() *cobra.Command {
	var (
		fromFile      string
		toEnv         string
		mapping       map[string]string
		noOverride    bool
		interactive   bool
		dryRun        bool
		showStats     bool
		expandVars    bool
		ignoreInvalid bool
	)

	cmd := &cobra.Command{
		Use:   "load",
		Short: "Load variables from .env file into VaultEnv",
		Long: `Load environment variables from a .env file into VaultEnv with 
intelligent handling of conflicts, format variations, and security concerns.

The load command supports various .env file formats and provides options for
conflict resolution, variable mapping, and selective import.`,
		Example: `  # Basic load from .env file
  vaultenv load --from .env

  # Load to specific environment
  vaultenv load --from .env.production --to production

  # Load with variable name mapping
  vaultenv load --from .env --map "DB_URL:DATABASE_URL,API:API_KEY"

  # Interactive mode for selective import
  vaultenv load --from .env --interactive

  # Dry run to preview changes
  vaultenv load --from .env --dry-run

  # Show detailed statistics
  vaultenv load --from .env --stats`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLoad(cmd, fromFile, toEnv, mapping, noOverride, interactive, dryRun, showStats, expandVars, ignoreInvalid)
		},
	}

	// Define flags with helpful descriptions
	cmd.Flags().StringVarP(&fromFile, "from", "f", ".env",
		"Path to .env file to import from")
	cmd.Flags().StringVarP(&toEnv, "to", "t", "",
		"Target environment (defaults to current)")
	cmd.Flags().StringToStringVarP(&mapping, "map", "m", nil,
		"Map variable names during import (OLD:NEW)")
	cmd.Flags().BoolVar(&noOverride, "no-override", false,
		"Skip variables that already exist")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false,
		"Interactively select which variables to import")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"Show what would be imported without making changes")
	cmd.Flags().BoolVar(&showStats, "stats", false,
		"Show detailed parsing statistics")
	cmd.Flags().BoolVar(&expandVars, "expand-vars", false,
		"Expand variable references in values (e.g., ${VAR})")
	cmd.Flags().BoolVar(&ignoreInvalid, "ignore-invalid", false,
		"Skip invalid lines instead of failing")

	return cmd
}

// runLoad executes the load command
func runLoad(cmd *cobra.Command, fromFile, toEnv string, mapping map[string]string,
	noOverride, interactive, dryRun, showStats, expandVars, ignoreInvalid bool) error {

	// Get configuration
	cfg := GetConfig(cmd)
	if cfg == nil {
		return fmt.Errorf("no configuration found, run 'vaultenv init' first")
	}

	// Resolve target environment
	if toEnv == "" {
		// Try to get from environment variable or use default
		toEnv = os.Getenv("VAULTENV_ENVIRONMENT")
		if toEnv == "" {
			// Use first available environment as default
			envNames := cfg.GetEnvironmentNames()
			if len(envNames) > 0 {
				toEnv = envNames[0]
			} else {
				return fmt.Errorf("no target environment specified and no environments configured")
			}
		}
	}

	// Validate target environment exists
	if !cfg.HasEnvironment(toEnv) {
		return fmt.Errorf("environment %q does not exist, create it first with 'vaultenv env create %s'", toEnv, toEnv)
	}

	// Check if file exists
	if _, err := os.Stat(fromFile); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", fromFile)
	}

	// Parse the .env file
	parser := dotenv.NewParser()
	parser.ExpandVars = expandVars
	parser.IgnoreInvalid = ignoreInvalid

	var vars map[string]string
	var err error

	if showStats {
		var stats dotenv.Stats
		vars, stats, err = parser.ParseWithStats(mustOpenFile(fromFile))
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", fromFile, err)
		}
		displayStats(fromFile, stats)
	} else {
		vars, err = parser.ParseFile(fromFile)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", fromFile, err)
		}
	}

	if len(vars) == 0 {
		ui.Warning("No variables found in %s", fromFile)
		return nil
	}

	ui.Info("Found %d variables in %s", len(vars), fromFile)

	// Apply name mappings if provided
	if len(mapping) > 0 {
		vars = applyMappings(vars, mapping)
		ui.Info("Applied %d variable name mappings", len(mapping))
	}

	// Handle interactive mode
	if interactive {
		selectedVars, err := interactiveSelection(vars)
		if err != nil {
			return fmt.Errorf("interactive selection failed: %w", err)
		}
		vars = selectedVars
		if len(vars) == 0 {
			ui.Info("No variables selected for import")
			return nil
		}
	}

	// Get storage for the target environment
	store, err := getStorageForEnvironment(cfg, toEnv)
	if err != nil {
		return fmt.Errorf("failed to get storage for environment %s: %w", toEnv, err)
	}

	// Check for conflicts with existing variables
	conflicts, err := checkConflicts(store, vars)
	if err != nil {
		return fmt.Errorf("failed to check for conflicts: %w", err)
	}

	if len(conflicts) > 0 && !noOverride {
		ui.Warning("The following variables already exist in %s:", toEnv)
		for _, name := range conflicts {
			existingValue, _ := store.Get(name)
			newValue := vars[name]
			ui.Info("  - %s: %s → %s", name, maskValue(existingValue), maskValue(newValue))
		}

		if !dryRun {
			override := false
			prompt := &survey.Confirm{
				Message: "Override existing variables?",
			}
			if err := survey.AskOne(prompt, &override); err != nil {
				return fmt.Errorf("failed to get user confirmation: %w", err)
			}

			if !override {
				// Filter out conflicts
				vars = filterConflicts(vars, conflicts)
				if len(vars) == 0 {
					ui.Info("No new variables to import after filtering conflicts")
					return nil
				}
			}
		}
	} else if len(conflicts) > 0 && noOverride {
		// Filter out conflicts when --no-override is set
		vars = filterConflicts(vars, conflicts)
		ui.Info("Skipped %d existing variables due to --no-override flag", len(conflicts))
		if len(vars) == 0 {
			ui.Info("No new variables to import")
			return nil
		}
	}

	// Show what will be imported
	ui.Info("Will import %d variables to %s environment:", len(vars), toEnv)
	displayImportPreview(vars, conflicts)

	if dryRun {
		ui.Info("Dry run complete - no changes made")
		return nil
	}

	// Import variables with progress indicator
	return importVariables(store, vars, toEnv)
}

// mustOpenFile opens a file or panics
func mustOpenFile(filename string) *os.File {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	return file
}

// displayStats shows parsing statistics
func displayStats(filename string, stats dotenv.Stats) {
	ui.Header("📊 Parsing Statistics for " + filename)
	ui.Info("Total lines: %d", stats.TotalLines)
	ui.Info("Variables: %d", stats.Variables)
	ui.Info("Comments: %d", stats.Comments)
	ui.Info("Empty lines: %d", stats.EmptyLines)

	if stats.InvalidLines > 0 {
		ui.Warning("Invalid lines: %d", stats.InvalidLines)
	}

	if len(stats.DuplicateKeys) > 0 {
		ui.Warning("Duplicate keys found: %v", stats.DuplicateKeys)
		ui.Info("Last value will be used for duplicates")
	}

	ui.Info("")
}

// applyMappings applies variable name mappings
func applyMappings(vars map[string]string, mapping map[string]string) map[string]string {
	result := make(map[string]string)

	for key, value := range vars {
		newKey := key
		if mapped, ok := mapping[key]; ok {
			newKey = mapped
		}
		result[newKey] = value
	}

	return result
}

// interactiveSelection allows user to select which variables to import
func interactiveSelection(vars map[string]string) (map[string]string, error) {
	if len(vars) == 0 {
		return vars, nil
	}

	// Sort keys for consistent display
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Create options with preview
	options := make([]string, len(keys))
	for i, key := range keys {
		value := vars[key]
		maskedValue := maskValue(value)
		options[i] = fmt.Sprintf("%s = %s", key, maskedValue)
	}

	var selected []string
	prompt := &survey.MultiSelect{
		Message: "Select variables to import:",
		Options: options,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return nil, err
	}

	// Build result map from selected items
	result := make(map[string]string)
	selectedSet := make(map[string]bool)
	for _, item := range selected {
		selectedSet[item] = true
	}

	for i, key := range keys {
		if selectedSet[options[i]] {
			result[key] = vars[key]
		}
	}

	return result, nil
}

// checkConflicts checks which variables already exist in storage
func checkConflicts(store storage.Backend, vars map[string]string) ([]string, error) {
	var conflicts []string

	for key := range vars {
		exists, err := store.Exists(key)
		if err != nil {
			return nil, fmt.Errorf("error checking for existing variable %s: %w", key, err)
		}
		if exists {
			conflicts = append(conflicts, key)
		}
	}

	sort.Strings(conflicts)
	return conflicts, nil
}

// filterConflicts removes conflicting variables from the map
func filterConflicts(vars map[string]string, conflicts []string) map[string]string {
	conflictSet := make(map[string]bool)
	for _, key := range conflicts {
		conflictSet[key] = true
	}

	result := make(map[string]string)
	for key, value := range vars {
		if !conflictSet[key] {
			result[key] = value
		}
	}

	return result
}

// displayImportPreview shows what will be imported
func displayImportPreview(vars map[string]string, conflicts []string) {
	// Sort keys for consistent display
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	conflictSet := make(map[string]bool)
	for _, key := range conflicts {
		conflictSet[key] = true
	}

	for _, key := range keys {
		value := vars[key]
		maskedValue := maskValue(value)

		if conflictSet[key] {
			ui.Info("  🔄 %s = %s (will override)", key, maskedValue)
		} else {
			ui.Info("  ➕ %s = %s (new)", key, maskedValue)
		}
	}
}

// importVariables imports the variables into storage
func importVariables(store storage.Backend, vars map[string]string, environment string) error {
	if len(vars) == 0 {
		return nil
	}

	ui.Info("Importing %d variables...", len(vars))

	// Import each variable
	imported := 0
	for key, value := range vars {
		if err := store.Set(key, value, true); err != nil { // true for encryption
			return fmt.Errorf("failed to set variable %s: %w", key, err)
		}
		imported++
	}

	ui.Success("Successfully imported %d variables to %s environment", imported, environment)
	return nil
}

// maskValue masks sensitive values for display
func maskValue(value string) string {
	if len(value) == 0 {
		return `""`
	}

	// Don't mask very short values (likely not sensitive)
	if len(value) <= 3 {
		return fmt.Sprintf("%q", value)
	}

	// Mask middle part of longer values
	if len(value) <= 8 {
		return fmt.Sprintf("%s***", value[:1])
	}

	return fmt.Sprintf("%s***%s", value[:3], value[len(value)-2:])
}

// getStorageForEnvironment gets storage configured for a specific environment
func getStorageForEnvironment(cfg *config.Config, environment string) (storage.Backend, error) {
	// Use the storage package's GetBackend function which respects test backends
	return storage.GetBackend(environment)
}

// Add the load command to the root command
func init() {
	// This will be called from execute.go when adding commands
}
