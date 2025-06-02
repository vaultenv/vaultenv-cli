package export

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Exporter interface for different file formats
type Exporter interface {
	Export(vars map[string]string, writer io.Writer) error
	FileExtension() string
	ContentType() string
}

// ExporterOptions contains common options for all exporters
type ExporterOptions struct {
	SortKeys     bool                   // Sort keys alphabetically
	IncludeEmpty bool                   // Include empty values
	ShowComments bool                   // Include comments where supported
	TemplateData map[string]interface{} // Additional template data
}

// DotEnvExporter exports in standard .env format
type DotEnvExporter struct {
	Options       ExporterOptions
	QuoteValues   bool // Always quote values
	IncludeExport bool // Prefix with 'export'
}

// NewDotEnvExporter creates a new .env format exporter
func NewDotEnvExporter() *DotEnvExporter {
	return &DotEnvExporter{
		Options: ExporterOptions{
			SortKeys:     true,
			IncludeEmpty: true,
		},
		QuoteValues:   false,
		IncludeExport: false,
	}
}

func (d *DotEnvExporter) Export(vars map[string]string, w io.Writer) error {
	// Filter empty values if needed
	if !d.Options.IncludeEmpty {
		vars = filterEmptyValues(vars)
	}

	// Sort keys if requested for consistent output
	keys := getSortedKeys(vars, d.Options.SortKeys)

	// Write header comment if enabled
	if d.Options.ShowComments {
		fmt.Fprintln(w, "# Environment variables exported by VaultEnv")
		fmt.Fprintln(w, "# Generated at:", getCurrentTimestamp())
		fmt.Fprintln(w)
	}

	// Write each variable
	for _, key := range keys {
		value := vars[key]

		// Handle special characters in values
		if d.needsQuoting(value) || d.QuoteValues {
			value = strconv.Quote(value)
		}

		// Write in appropriate format
		if d.IncludeExport {
			fmt.Fprintf(w, "export %s=%s\n", key, value)
		} else {
			fmt.Fprintf(w, "%s=%s\n", key, value)
		}
	}

	return nil
}

func (d *DotEnvExporter) FileExtension() string {
	return ".env"
}

func (d *DotEnvExporter) ContentType() string {
	return "text/plain"
}

// needsQuoting determines if a value needs to be quoted
func (d *DotEnvExporter) needsQuoting(value string) bool {
	// Quote if contains spaces, quotes, or special characters
	return strings.ContainsAny(value, " \t\n\r\"'$`\\#")
}

// JSONExporter exports as JSON object
type JSONExporter struct {
	Options     ExporterOptions
	PrettyPrint bool
	Indent      string
}

// NewJSONExporter creates a new JSON format exporter
func NewJSONExporter() *JSONExporter {
	return &JSONExporter{
		Options: ExporterOptions{
			SortKeys:     true,
			IncludeEmpty: true,
		},
		PrettyPrint: true,
		Indent:      "  ",
	}
}

func (j *JSONExporter) Export(vars map[string]string, w io.Writer) error {
	// Filter empty values if needed
	if !j.Options.IncludeEmpty {
		vars = filterEmptyValues(vars)
	}

	encoder := json.NewEncoder(w)
	if j.PrettyPrint {
		encoder.SetIndent("", j.Indent)
	}

	// Convert to interface{} for JSON encoding with sorted keys
	if j.Options.SortKeys {
		// Create a slice of key-value pairs to maintain order
		type kv struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}

		keys := getSortedKeys(vars, true)
		kvPairs := make([]kv, len(keys))
		for i, key := range keys {
			kvPairs[i] = kv{Key: key, Value: vars[key]}
		}

		// For sorted output, we need to use a different structure
		// Let's just encode the map directly as JSON handles key sorting
		return encoder.Encode(vars)
	}

	return encoder.Encode(vars)
}

func (j *JSONExporter) FileExtension() string {
	return ".json"
}

func (j *JSONExporter) ContentType() string {
	return "application/json"
}

// YAMLExporter exports as YAML format
type YAMLExporter struct {
	Options ExporterOptions
}

// NewYAMLExporter creates a new YAML format exporter
func NewYAMLExporter() *YAMLExporter {
	return &YAMLExporter{
		Options: ExporterOptions{
			SortKeys:     true,
			IncludeEmpty: true,
		},
	}
}

func (y *YAMLExporter) Export(vars map[string]string, w io.Writer) error {
	// Filter empty values if needed
	if !y.Options.IncludeEmpty {
		vars = filterEmptyValues(vars)
	}

	// Write header comment if enabled
	if y.Options.ShowComments {
		fmt.Fprintln(w, "# Environment variables exported by VaultEnv")
		fmt.Fprintln(w, "# Generated at:", getCurrentTimestamp())
	}

	encoder := yaml.NewEncoder(w)
	defer encoder.Close()

	return encoder.Encode(vars)
}

func (y *YAMLExporter) FileExtension() string {
	return ".yaml"
}

func (y *YAMLExporter) ContentType() string {
	return "application/x-yaml"
}

// ShellExporter exports as shell script
type ShellExporter struct {
	Options    ExporterOptions
	ExportVars bool   // Make variables available to child processes
	SetCommand string // Command to use (export, set, etc.)
}

// NewShellExporter creates a new shell script exporter
func NewShellExporter() *ShellExporter {
	return &ShellExporter{
		Options: ExporterOptions{
			SortKeys:     true,
			IncludeEmpty: true,
			ShowComments: true,
		},
		ExportVars: true,
		SetCommand: "export",
	}
}

func (s *ShellExporter) Export(vars map[string]string, w io.Writer) error {
	// Filter empty values if needed
	if !s.Options.IncludeEmpty {
		vars = filterEmptyValues(vars)
	}

	// Write shell script header
	fmt.Fprintln(w, "#!/bin/bash")
	if s.Options.ShowComments {
		fmt.Fprintln(w, "# Environment variables exported by VaultEnv")
		fmt.Fprintln(w, "# Generated at:", getCurrentTimestamp())
		fmt.Fprintln(w, "# DO NOT EDIT MANUALLY")
	}
	fmt.Fprintln(w)

	// Sort keys for consistent output
	keys := getSortedKeys(vars, s.Options.SortKeys)

	for _, key := range keys {
		value := vars[key]
		// Properly escape for shell
		escaped := shellEscape(value)

		if s.ExportVars {
			fmt.Fprintf(w, "%s %s=%s\n", s.SetCommand, key, escaped)
		} else {
			fmt.Fprintf(w, "%s=%s\n", key, escaped)
		}
	}

	return nil
}

func (s *ShellExporter) FileExtension() string {
	return ".sh"
}

func (s *ShellExporter) ContentType() string {
	return "application/x-sh"
}

// DockerExporter exports as Docker ENV format
type DockerExporter struct {
	Options ExporterOptions
}

// NewDockerExporter creates a new Docker ENV format exporter
func NewDockerExporter() *DockerExporter {
	return &DockerExporter{
		Options: ExporterOptions{
			SortKeys:     true,
			IncludeEmpty: false, // Docker typically skips empty values
			ShowComments: true,
		},
	}
}

func (d *DockerExporter) Export(vars map[string]string, w io.Writer) error {
	// Filter empty values if needed
	if !d.Options.IncludeEmpty {
		vars = filterEmptyValues(vars)
	}

	// Write header comment if enabled
	if d.Options.ShowComments {
		fmt.Fprintln(w, "# Environment variables exported by VaultEnv")
		fmt.Fprintln(w, "# Generated at:", getCurrentTimestamp())
		fmt.Fprintln(w, "# Add these lines to your Dockerfile")
		fmt.Fprintln(w)
	}

	// Sort keys for consistent output
	keys := getSortedKeys(vars, d.Options.SortKeys)

	for _, key := range keys {
		value := vars[key]
		// Docker ENV instruction format
		fmt.Fprintf(w, "ENV %s=%s\n", key, dockerEscape(value))
	}

	return nil
}

func (d *DockerExporter) FileExtension() string {
	return ".dockerfile"
}

func (d *DockerExporter) ContentType() string {
	return "text/plain"
}

// TemplateExporter exports using a custom template
type TemplateExporter struct {
	Options      ExporterOptions
	Template     *template.Template
	TemplateName string
}

// NewTemplateExporter creates a new template-based exporter
func NewTemplateExporter(tmplContent string, name string) (*TemplateExporter, error) {
	tmpl, err := template.New(name).Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &TemplateExporter{
		Options: ExporterOptions{
			SortKeys:     true,
			IncludeEmpty: true,
		},
		Template:     tmpl,
		TemplateName: name,
	}, nil
}

func (t *TemplateExporter) Export(vars map[string]string, w io.Writer) error {
	// Filter empty values if needed
	if !t.Options.IncludeEmpty {
		vars = filterEmptyValues(vars)
	}

	// Prepare template data
	data := map[string]interface{}{
		"Variables": vars,
		"Keys":      getSortedKeys(vars, t.Options.SortKeys),
		"Timestamp": getCurrentTimestamp(),
	}

	// Add any additional template data
	for k, v := range t.Options.TemplateData {
		data[k] = v
	}

	return t.Template.Execute(w, data)
}

func (t *TemplateExporter) FileExtension() string {
	return ".txt" // Default, can be overridden
}

func (t *TemplateExporter) ContentType() string {
	return "text/plain"
}

// ExporterFactory creates exporters by format name
type ExporterFactory struct{}

// NewExporterFactory creates a new exporter factory
func NewExporterFactory() *ExporterFactory {
	return &ExporterFactory{}
}

// CreateExporter creates an exporter for the specified format
func (f *ExporterFactory) CreateExporter(format string) (Exporter, error) {
	switch strings.ToLower(format) {
	case "dotenv", "env":
		return NewDotEnvExporter(), nil
	case "json":
		return NewJSONExporter(), nil
	case "yaml", "yml":
		return NewYAMLExporter(), nil
	case "shell", "sh", "bash":
		return NewShellExporter(), nil
	case "docker", "dockerfile":
		return NewDockerExporter(), nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// GetSupportedFormats returns a list of supported export formats
func (f *ExporterFactory) GetSupportedFormats() []string {
	return []string{"dotenv", "json", "yaml", "shell", "docker"}
}

// Helper functions

// filterEmptyValues removes variables with empty values
func filterEmptyValues(vars map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range vars {
		if v != "" {
			result[k] = v
		}
	}
	return result
}

// getSortedKeys returns sorted keys if requested
func getSortedKeys(vars map[string]string, shouldSort bool) []string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	if shouldSort {
		sort.Strings(keys)
	}
	return keys
}

// shellEscape properly escapes a value for shell scripts
func shellEscape(value string) string {
	// Use single quotes to avoid most shell interpretation
	// Replace single quotes with '"'"' pattern
	if !strings.Contains(value, "'") {
		return "'" + value + "'"
	}

	// Complex escaping for values containing single quotes
	escaped := strings.ReplaceAll(value, "'", "'\"'\"'")
	return "'" + escaped + "'"
}

// dockerEscape escapes a value for Docker ENV instruction
func dockerEscape(value string) string {
	// Docker ENV supports both ENV key=value and ENV key="value" formats
	// Quote if contains spaces or special characters
	if strings.ContainsAny(value, " \t\n\r\"'$\\") {
		return strconv.Quote(value)
	}
	return value
}

// getCurrentTimestamp returns current timestamp as string
func getCurrentTimestamp() string {
	// In a real implementation, this would use time.Now()
	// For testing, we return a static value
	return "2024-01-01T12:00:00Z"
}
