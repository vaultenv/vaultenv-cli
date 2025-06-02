package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"github.com/vaultenv/vaultenv-cli/pkg/dotenv"
	"github.com/vaultenv/vaultenv-cli/pkg/export"
)

// newBatchCommand creates the batch command for bulk operations
func newBatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Perform batch operations on variables",
		Long: `Execute batch operations across multiple environments or variables.
		
Batch operations allow you to efficiently manage variables across multiple
environments, export all environments at once, or perform bulk imports.`,
		Example: `  # Export all environments
  vaultenv batch export-all --to-dir ./backups/

  # Import multiple .env files
  vaultenv batch import-all --from-dir ./configs/

  # Copy environment variables
  vaultenv batch copy --from development --to staging`,
	}

	cmd.AddCommand(
		newBatchExportCommand(),
		newBatchImportCommand(),
		newBatchCopyCommand(),
	)

	return cmd
}

// newBatchExportCommand creates the batch export-all command
func newBatchExportCommand() *cobra.Command {
	var (
		toDir        string
		format       string
		timestamp    bool
		envs         []string
		includeEmpty bool
		overwrite    bool
		dryRun       bool
	)

	cmd := &cobra.Command{
		Use:   "export-all",
		Short: "Export all environments to separate files",
		Long: `Export all environments to separate files in a directory.
		
Each environment will be exported to a separate file named after the environment.
The export format and various options can be customized.`,
		Example: `  # Export all environments
  vaultenv batch export-all --to-dir ./backups/

  # Export with timestamp
  vaultenv batch export-all --to-dir ./backups/ --timestamp

  # Export specific environments
  vaultenv batch export-all --envs "dev,staging,prod" --to-dir ./deploy/

  # Export as YAML format
  vaultenv batch export-all --to-dir ./configs/ --format yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatchExport(cmd, toDir, format, timestamp, envs, includeEmpty, overwrite, dryRun)
		},
	}

	cmd.Flags().StringVar(&toDir, "to-dir", "",
		"Output directory for exported files")
	cmd.Flags().StringVar(&format, "format", "dotenv",
		"Export format: dotenv, json, yaml, shell, docker")
	cmd.Flags().BoolVar(&timestamp, "timestamp", false,
		"Include timestamp in filenames")
	cmd.Flags().StringSliceVar(&envs, "envs", nil,
		"Specific environments to export (comma-separated)")
	cmd.Flags().BoolVar(&includeEmpty, "include-empty", true,
		"Include variables with empty values")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false,
		"Overwrite existing files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"Show what would be exported without creating files")

	cmd.MarkFlagRequired("to-dir")

	return cmd
}

// runBatchExport executes the batch export command
func runBatchExport(cmd *cobra.Command, toDir, format string, timestamp bool,
	envs []string, includeEmpty, overwrite, dryRun bool) error {

	// Get configuration
	cfg := GetConfig(cmd)
	if cfg == nil {
		return fmt.Errorf("no configuration found, run 'vaultenv init' first")
	}

	// Create output directory if not in dry-run mode
	if !dryRun {
		if err := os.MkdirAll(toDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Get list of environments to export
	if len(envs) == 0 {
		envs = cfg.GetEnvironmentNames()
	}

	if len(envs) == 0 {
		ui.Warning("No environments found to export")
		return nil
	}

	// Validate that all specified environments exist
	for _, env := range envs {
		if !cfg.HasEnvironment(env) {
			return fmt.Errorf("environment %q does not exist", env)
		}
	}

	ui.Info("Exporting %d environments to %s", len(envs), toDir)

	// Create exporter factory
	factory := export.NewExporterFactory()

	// Export each environment
	var exportedCount int
	var errors []error

	for _, env := range envs {
		if err := exportSingleEnvironment(cfg, env, toDir, format, timestamp,
			includeEmpty, overwrite, dryRun, factory); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", env, err))
			ui.Error("Failed to export %s: %v", env, err)
		} else {
			exportedCount++
			ui.Success("Exported %s", env)
		}
	}

	// Report results
	if dryRun {
		ui.Info("Dry run complete - would export %d environments", len(envs))
	} else {
		ui.Success("Successfully exported %d/%d environments", exportedCount, len(envs))
	}

	if len(errors) > 0 {
		ui.Error("Failed to export %d environments:", len(errors))
		for _, err := range errors {
			ui.Error("  - %v", err)
		}
		return fmt.Errorf("batch export partially failed")
	}

	return nil
}

// exportSingleEnvironment exports a single environment to a file
func exportSingleEnvironment(cfg *config.Config, env, toDir, format string,
	timestamp, includeEmpty, overwrite, dryRun bool, factory *export.ExporterFactory) error {

	// Get storage for the environment
	store, err := getStorageForEnvironment(cfg, env)
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}

	// Get all variables
	vars, err := getAllVariables(store)
	if err != nil {
		return fmt.Errorf("failed to get variables: %w", err)
	}

	if len(vars) == 0 {
		ui.Debug("No variables found in environment %s", env)
		return nil
	}

	// Create exporter
	exporter, err := factory.CreateExporter(format)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	// Configure exporter
	configureExporter(exporter, true, includeEmpty, false)

	// Generate filename
	filename := fmt.Sprintf("%s%s", env, exporter.FileExtension())
	if timestamp {
		timeStr := time.Now().Format("20060102-150405")
		filename = fmt.Sprintf("%s-%s%s", env, timeStr, exporter.FileExtension())
	}

	filepath := filepath.Join(toDir, filename)

	if dryRun {
		ui.Info("Would export %d variables from %s to %s", len(vars), env, filepath)
		return nil
	}

	// Check if file exists
	if !overwrite {
		if _, err := os.Stat(filepath); err == nil {
			return fmt.Errorf("file %s already exists (use --overwrite to replace)", filepath)
		}
	}

	// Export to file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := exporter.Export(vars, file); err != nil {
		return fmt.Errorf("failed to export: %w", err)
	}

	return nil
}

// newBatchImportCommand creates the batch import-all command
func newBatchImportCommand() *cobra.Command {
	var (
		fromDir       string
		pattern       string
		toEnv         string
		createEnvs    bool
		noOverride    bool
		dryRun        bool
		expandVars    bool
		ignoreInvalid bool
	)

	cmd := &cobra.Command{
		Use:   "import-all",
		Short: "Import multiple .env files from a directory",
		Long: `Import multiple .env files from a directory into VaultEnv.
		
Files are matched by pattern and imported into environments based on their names.
New environments can be created automatically if they don't exist.`,
		Example: `  # Import all .env files from directory
  vaultenv batch import-all --from-dir ./configs/

  # Import with specific pattern
  vaultenv batch import-all --from-dir ./configs/ --pattern "*.env"

  # Import and create environments automatically
  vaultenv batch import-all --from-dir ./configs/ --create-envs

  # Import all files to specific environment
  vaultenv batch import-all --from-dir ./configs/ --to-env production`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatchImport(cmd, fromDir, pattern, toEnv, createEnvs, noOverride, dryRun, expandVars, ignoreInvalid)
		},
	}

	cmd.Flags().StringVar(&fromDir, "from-dir", "",
		"Directory containing .env files to import")
	cmd.Flags().StringVar(&pattern, "pattern", "*.env",
		"File pattern to match (e.g., *.env, *.config)")
	cmd.Flags().StringVar(&toEnv, "to-env", "",
		"Target environment for all imports (overrides filename-based detection)")
	cmd.Flags().BoolVar(&createEnvs, "create-envs", false,
		"Create environments automatically if they don't exist")
	cmd.Flags().BoolVar(&noOverride, "no-override", false,
		"Skip variables that already exist")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"Show what would be imported without making changes")
	cmd.Flags().BoolVar(&expandVars, "expand-vars", false,
		"Expand variable references in values")
	cmd.Flags().BoolVar(&ignoreInvalid, "ignore-invalid", false,
		"Skip invalid lines instead of failing")

	cmd.MarkFlagRequired("from-dir")

	return cmd
}

// runBatchImport executes the batch import command
func runBatchImport(cmd *cobra.Command, fromDir, pattern, toEnv string,
	createEnvs, noOverride, dryRun, expandVars, ignoreInvalid bool) error {

	// Get configuration
	cfg := GetConfig(cmd)
	if cfg == nil {
		return fmt.Errorf("no configuration found, run 'vaultenv init' first")
	}

	// Find matching files
	files, err := findMatchingFiles(fromDir, pattern)
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	if len(files) == 0 {
		ui.Warning("No files found matching pattern %s in %s", pattern, fromDir)
		return nil
	}

	ui.Info("Found %d files to import", len(files))

	// Process each file
	var importedCount int
	var errors []error

	for _, file := range files {
		env := toEnv
		if env == "" {
			// Derive environment name from filename
			env = deriveEnvironmentName(file)
		}

		if err := importSingleFile(cfg, file, env, createEnvs, noOverride,
			dryRun, expandVars, ignoreInvalid); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", filepath.Base(file), err))
			ui.Error("Failed to import %s: %v", filepath.Base(file), err)
		} else {
			importedCount++
			ui.Success("Imported %s to %s", filepath.Base(file), env)
		}
	}

	// Report results
	if dryRun {
		ui.Info("Dry run complete - would import %d files", len(files))
	} else {
		ui.Success("Successfully imported %d/%d files", importedCount, len(files))
	}

	if len(errors) > 0 {
		ui.Error("Failed to import %d files:", len(errors))
		for _, err := range errors {
			ui.Error("  - %v", err)
		}
		return fmt.Errorf("batch import partially failed")
	}

	return nil
}

// importSingleFile imports a single .env file
func importSingleFile(cfg *config.Config, filename, env string, createEnvs, noOverride,
	dryRun, expandVars, ignoreInvalid bool) error {

	// Check if environment exists
	if !cfg.HasEnvironment(env) {
		if createEnvs {
			if !dryRun {
				cfg.SetEnvironmentConfig(env, config.EnvironmentConfig{
					Description: fmt.Sprintf("Auto-created from %s", filepath.Base(filename)),
				})
				if err := cfg.Save(); err != nil {
					return fmt.Errorf("failed to save config after creating environment: %w", err)
				}
			}
			ui.Info("Would create environment: %s", env)
		} else {
			return fmt.Errorf("environment %s does not exist (use --create-envs to create)", env)
		}
	}

	// Parse the file
	parser := dotenv.NewParser()
	parser.ExpandVars = expandVars
	parser.IgnoreInvalid = ignoreInvalid

	vars, err := parser.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	if len(vars) == 0 {
		ui.Debug("No variables found in %s", filename)
		return nil
	}

	if dryRun {
		ui.Info("Would import %d variables from %s to %s", len(vars), filepath.Base(filename), env)
		return nil
	}

	// Get storage and import
	store, err := getStorageForEnvironment(cfg, env)
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}

	return importVariables(store, vars, env)
}

// findMatchingFiles finds files matching a pattern in a directory
func findMatchingFiles(dir, pattern string) ([]string, error) {
	fullPattern := filepath.Join(dir, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, err
	}

	// Filter out directories
	var files []string
	for _, match := range matches {
		if info, err := os.Stat(match); err == nil && !info.IsDir() {
			files = append(files, match)
		}
	}

	sort.Strings(files)
	return files, nil
}

// deriveEnvironmentName derives an environment name from a filename
func deriveEnvironmentName(filename string) string {
	base := filepath.Base(filename)

	// Remove common extensions
	extensions := []string{".env", ".config", ".txt"}
	for _, ext := range extensions {
		if strings.HasSuffix(base, ext) {
			base = strings.TrimSuffix(base, ext)
			break
		}
	}

	// Handle common patterns
	if base == ".env" {
		return "development"
	}

	// Remove .env prefix if present (e.g., .env.production -> production)
	if strings.HasPrefix(base, ".env.") {
		return strings.TrimPrefix(base, ".env.")
	}

	// Remove env prefix if present (e.g., env.production -> production)
	if strings.HasPrefix(base, "env.") {
		return strings.TrimPrefix(base, "env.")
	}

	return base
}

// newBatchCopyCommand creates the batch copy command
func newBatchCopyCommand() *cobra.Command {
	var (
		fromEnv    string
		toEnv      string
		filter     []string
		noOverride bool
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "copy",
		Short: "Copy variables between environments",
		Long: `Copy environment variables from one environment to another.
		
Variables can be filtered and existing variables can be preserved or overwritten.
This is useful for setting up new environments based on existing ones.`,
		Example: `  # Copy all variables from dev to staging
  vaultenv batch copy --from development --to staging

  # Copy specific variables
  vaultenv batch copy --from development --to staging --filter "API_*,DB_*"

  # Copy without overriding existing variables
  vaultenv batch copy --from development --to staging --no-override`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatchCopy(cmd, fromEnv, toEnv, filter, noOverride, dryRun)
		},
	}

	cmd.Flags().StringVar(&fromEnv, "from", "",
		"Source environment")
	cmd.Flags().StringVar(&toEnv, "to", "",
		"Target environment")
	cmd.Flags().StringSliceVar(&filter, "filter", nil,
		"Filter variables by pattern")
	cmd.Flags().BoolVar(&noOverride, "no-override", false,
		"Skip variables that already exist in target")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"Show what would be copied without making changes")

	cmd.MarkFlagRequired("from")
	cmd.MarkFlagRequired("to")

	return cmd
}

// runBatchCopy executes the batch copy command
func runBatchCopy(cmd *cobra.Command, fromEnv, toEnv string, filter []string, noOverride, dryRun bool) error {
	// Get configuration
	cfg := GetConfig(cmd)
	if cfg == nil {
		return fmt.Errorf("no configuration found, run 'vaultenv init' first")
	}

	// Validate environments exist
	if !cfg.HasEnvironment(fromEnv) {
		return fmt.Errorf("source environment %q does not exist", fromEnv)
	}
	if !cfg.HasEnvironment(toEnv) {
		return fmt.Errorf("target environment %q does not exist", toEnv)
	}

	// Get storage for both environments
	sourceStore, err := getStorageForEnvironment(cfg, fromEnv)
	if err != nil {
		return fmt.Errorf("failed to get source storage: %w", err)
	}

	targetStore, err := getStorageForEnvironment(cfg, toEnv)
	if err != nil {
		return fmt.Errorf("failed to get target storage: %w", err)
	}

	// Get source variables
	sourceVars, err := getAllVariables(sourceStore)
	if err != nil {
		return fmt.Errorf("failed to get source variables: %w", err)
	}

	if len(sourceVars) == 0 {
		ui.Warning("No variables found in source environment %s", fromEnv)
		return nil
	}

	// Apply filters
	vars := sourceVars
	if len(filter) > 0 {
		vars = applyFilters(vars, filter)
		ui.Info("Filtered to %d variables matching patterns: %v", len(vars), filter)
	}

	if len(vars) == 0 {
		ui.Warning("No variables to copy after filtering")
		return nil
	}

	// Check for conflicts
	conflicts, err := checkConflicts(targetStore, vars)
	if err != nil {
		return fmt.Errorf("failed to check for conflicts: %w", err)
	}

	if len(conflicts) > 0 && noOverride {
		vars = filterConflicts(vars, conflicts)
		ui.Info("Skipped %d existing variables due to --no-override", len(conflicts))
	} else if len(conflicts) > 0 {
		ui.Warning("Will override %d existing variables in %s", len(conflicts), toEnv)
		for _, name := range conflicts {
			ui.Info("  - %s", name)
		}

		if !dryRun {
			confirm := false
			prompt := &survey.Confirm{
				Message: "Continue with copy operation?",
			}
			if err := survey.AskOne(prompt, &confirm); err != nil {
				return fmt.Errorf("failed to get confirmation: %w", err)
			}
			if !confirm {
				ui.Info("Copy operation cancelled")
				return nil
			}
		}
	}

	if len(vars) == 0 {
		ui.Info("No variables to copy")
		return nil
	}

	ui.Info("Copying %d variables from %s to %s", len(vars), fromEnv, toEnv)

	if dryRun {
		ui.Info("Dry run complete - would copy %d variables", len(vars))
		return nil
	}

	// Copy variables
	return importVariables(targetStore, vars, toEnv)
}
