package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"github.com/vaultenv/vaultenv-cli/pkg/export"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

// newExportCommand creates the export command for exporting variables to files
func newExportCommand() *cobra.Command {
	var (
		fromEnv      string
		toFile       string
		format       string
		filter       []string
		showValues   bool
		template     string
		overwrite    bool
		dryRun       bool
		sortKeys     bool
		includeEmpty bool
		showComments bool
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export variables from VaultEnv to files",
		Long: `Export environment variables from VaultEnv to various file formats.
Supports filtering, templating, and multiple output formats for different use cases.

The export command supports multiple output formats including .env, JSON, YAML,
shell scripts, and Docker ENV instructions. Variables can be filtered, and
output can be customized with various options.`,
		Example: `  # Export to .env file
  vaultenv export --to .env.local

  # Export specific environment
  vaultenv export --from production --to .env.prod

  # Export with filtering
  vaultenv export --filter "API_*,DATABASE_*" --to api.env

  # Export as JSON
  vaultenv export --format json --to config.json

  # Export using custom template
  vaultenv export --template export.tmpl --to config.sh

  # Export to stdout
  vaultenv export --format yaml

  # Dry run to preview output
  vaultenv export --to config.json --dry-run`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(cmd, fromEnv, toFile, format, filter, showValues, template,
				overwrite, dryRun, sortKeys, includeEmpty, showComments)
		},
	}

	// Configure flags
	cmd.Flags().StringVarP(&fromEnv, "from", "f", "",
		"Source environment (defaults to current)")
	cmd.Flags().StringVarP(&toFile, "to", "t", "",
		"Output file path (defaults to stdout)")
	cmd.Flags().StringVar(&format, "format", "dotenv",
		"Output format: dotenv, json, yaml, shell, docker")
	cmd.Flags().StringSliceVar(&filter, "filter", nil,
		"Filter variables by pattern (supports wildcards)")
	cmd.Flags().BoolVar(&showValues, "show-values", true,
		"Include actual values (false for templates)")
	cmd.Flags().StringVar(&template, "template", "",
		"Use custom template file")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false,
		"Overwrite existing output file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"Show what would be exported without writing to file")
	cmd.Flags().BoolVar(&sortKeys, "sort", true,
		"Sort variables alphabetically")
	cmd.Flags().BoolVar(&includeEmpty, "include-empty", true,
		"Include variables with empty values")
	cmd.Flags().BoolVar(&showComments, "comments", false,
		"Include comments in output (where supported)")

	return cmd
}

// runExport executes the export command
func runExport(cmd *cobra.Command, fromEnv, toFile, format string, filter []string,
	showValues bool, template string, overwrite, dryRun, sortKeys, includeEmpty, showComments bool) error {

	// Get configuration
	cfg := GetConfig(cmd)
	if cfg == nil {
		return fmt.Errorf("no configuration found, run 'vaultenv init' first")
	}

	// Resolve source environment
	if fromEnv == "" {
		// Try to get from environment variable or use default
		fromEnv = os.Getenv("VAULTENV_ENVIRONMENT")
		if fromEnv == "" {
			// Use first available environment as default
			envNames := cfg.GetEnvironmentNames()
			if len(envNames) > 0 {
				fromEnv = envNames[0]
			} else {
				return fmt.Errorf("no source environment specified and no environments configured")
			}
		}
	}

	// Validate source environment exists
	if !cfg.HasEnvironment(fromEnv) {
		return fmt.Errorf("environment %q does not exist", fromEnv)
	}

	// Get storage for the source environment
	store, err := getStorageForEnvironment(cfg, fromEnv)
	if err != nil {
		return fmt.Errorf("failed to get storage for environment %s: %w", fromEnv, err)
	}

	// Get all variables from storage
	allVars, err := getAllVariables(store)
	if err != nil {
		return fmt.Errorf("failed to retrieve variables from %s: %w", fromEnv, err)
	}

	if len(allVars) == 0 {
		ui.Warning("No variables found in %s environment", fromEnv)
		return nil
	}

	ui.Info("Found %d variables in %s environment", len(allVars), fromEnv)

	// Apply filters if provided
	vars := allVars
	if len(filter) > 0 {
		vars = applyFilters(vars, filter)
		ui.Info("Filtered to %d variables matching patterns: %v", len(vars), filter)

		if len(vars) == 0 {
			ui.Warning("No variables match the specified filters")
			return nil
		}
	}

	// Mask values if requested
	if !showValues {
		vars = maskAllValues(vars)
	}

	// Create exporter
	var exporter export.Exporter
	if template != "" {
		// Use custom template
		templateContent, err := os.ReadFile(template)
		if err != nil {
			return fmt.Errorf("failed to read template file %s: %w", template, err)
		}

		exporter, err = export.NewTemplateExporter(string(templateContent), filepath.Base(template))
		if err != nil {
			return fmt.Errorf("failed to create template exporter: %w", err)
		}
	} else {
		// Use format-based exporter
		factory := export.NewExporterFactory()
		exporter, err = factory.CreateExporter(format)
		if err != nil {
			return fmt.Errorf("failed to create exporter: %w", err)
		}
	}

	// Configure exporter options
	configureExporter(exporter, sortKeys, includeEmpty, showComments)

	// Determine output destination
	var outputFile *os.File
	var shouldCloseFile bool

	if toFile == "" {
		// Output to stdout
		outputFile = os.Stdout
		shouldCloseFile = false
	} else {
		// Check if file exists and handle overwrite
		if !overwrite && !dryRun {
			if _, err := os.Stat(toFile); err == nil {
				confirmOverwrite := false
				prompt := &survey.Confirm{
					Message: fmt.Sprintf("File %s already exists. Overwrite?", toFile),
				}
				if err := survey.AskOne(prompt, &confirmOverwrite); err != nil {
					return fmt.Errorf("failed to get user confirmation: %w", err)
				}
				if !confirmOverwrite {
					ui.Info("Export cancelled")
					return nil
				}
			}
		}

		if dryRun {
			ui.Info("Would write to file: %s", toFile)
			outputFile = os.Stdout
			shouldCloseFile = false
		} else {
			// Create output directory if needed
			if dir := filepath.Dir(toFile); dir != "." {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("failed to create output directory: %w", err)
				}
			}

			// Open output file
			outputFile, err = os.Create(toFile)
			if err != nil {
				return fmt.Errorf("failed to create output file %s: %w", toFile, err)
			}
			shouldCloseFile = true
		}
	}

	if shouldCloseFile {
		defer outputFile.Close()
	}

	// Show export preview
	if toFile != "" {
		ui.Info("Exporting %d variables to %s (format: %s)", len(vars), toFile, format)
	}

	// Export variables
	if err := exporter.Export(vars, outputFile); err != nil {
		return fmt.Errorf("failed to export variables: %w", err)
	}

	if dryRun {
		ui.Info("Dry run complete - no file was written")
	} else if toFile != "" {
		ui.Success("Successfully exported %d variables to %s", len(vars), toFile)
	}

	return nil
}

// getAllVariables retrieves all variables from storage
func getAllVariables(store storage.Backend) (map[string]string, error) {
	// Get list of all variables
	keys, err := store.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	// Retrieve all values
	result := make(map[string]string)
	for _, key := range keys {
		value, err := store.Get(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get variable %s: %w", key, err)
		}
		result[key] = value
	}

	return result, nil
}

// applyFilters filters variables based on patterns
func applyFilters(vars map[string]string, patterns []string) map[string]string {
	if len(patterns) == 0 {
		return vars
	}

	result := make(map[string]string)

	for key, value := range vars {
		for _, pattern := range patterns {
			if matchesPattern(key, pattern) {
				result[key] = value
				break
			}
		}
	}

	return result
}

// matchesPattern checks if a key matches a wildcard pattern
func matchesPattern(key, pattern string) bool {
	// Simple wildcard matching
	// Supports * for multiple characters and ? for single character

	// If no wildcards, do exact match
	if !strings.Contains(pattern, "*") && !strings.Contains(pattern, "?") {
		return key == pattern
	}

	// Convert pattern to Go regexp (simplified)
	// * becomes .*
	// ? becomes .
	regexPattern := strings.ReplaceAll(pattern, "*", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "?", ".")
	regexPattern = "^" + regexPattern + "$"

	// Use simple string matching for common cases
	if strings.HasSuffix(pattern, "*") && !strings.Contains(pattern[:len(pattern)-1], "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(key, prefix)
	}

	if strings.HasPrefix(pattern, "*") && !strings.Contains(pattern[1:], "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(key, suffix)
	}

	// For complex patterns, we'd need regexp package
	// For now, just do prefix matching for patterns ending with *
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(key, prefix)
	}

	return key == pattern
}

// maskAllValues masks all variable values for display
func maskAllValues(vars map[string]string) map[string]string {
	result := make(map[string]string)
	for key, value := range vars {
		result[key] = maskValueForExport(value)
	}
	return result
}

// maskValueForExport masks a single value for export
func maskValueForExport(value string) string {
	if len(value) == 0 {
		return ""
	}
	if len(value) <= 4 {
		return "***"
	}
	return value[:2] + "***" + value[len(value)-2:]
}

// configureExporter sets common options on an exporter
func configureExporter(exporter export.Exporter, sortKeys, includeEmpty, showComments bool) {
	// Use type assertion to configure specific exporter types
	switch e := exporter.(type) {
	case *export.DotEnvExporter:
		e.Options.SortKeys = sortKeys
		e.Options.IncludeEmpty = includeEmpty
		e.Options.ShowComments = showComments
	case *export.JSONExporter:
		e.Options.SortKeys = sortKeys
		e.Options.IncludeEmpty = includeEmpty
	case *export.YAMLExporter:
		e.Options.SortKeys = sortKeys
		e.Options.IncludeEmpty = includeEmpty
		e.Options.ShowComments = showComments
	case *export.ShellExporter:
		e.Options.SortKeys = sortKeys
		e.Options.IncludeEmpty = includeEmpty
		e.Options.ShowComments = showComments
	case *export.DockerExporter:
		e.Options.SortKeys = sortKeys
		e.Options.IncludeEmpty = includeEmpty
		e.Options.ShowComments = showComments
	case *export.TemplateExporter:
		e.Options.SortKeys = sortKeys
		e.Options.IncludeEmpty = includeEmpty
	}
}

// displayExportPreview shows what will be exported
func displayExportPreview(vars map[string]string, format string) {
	ui.Info("Export preview (%s format):", format)

	// Sort keys for consistent display
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Show first few variables as preview
	maxPreview := 5
	for i, key := range keys {
		if i >= maxPreview {
			ui.Info("  ... and %d more variables", len(keys)-maxPreview)
			break
		}

		value := vars[key]
		maskedValue := maskValueForExport(value)
		ui.Info("  %s = %s", key, maskedValue)
	}
}
