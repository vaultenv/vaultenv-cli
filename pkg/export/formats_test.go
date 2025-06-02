package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDotEnvExporter(t *testing.T) {
	vars := map[string]string{
		"API_KEY":      "secret123",
		"DATABASE_URL": "postgres://user:pass@localhost/db",
		"EMPTY_VAR":    "",
		"QUOTED_VAR":   `value with "quotes"`,
		"MULTILINE":    "line1\nline2",
		"SPECIAL":      "value with $pecial ch@rs!",
	}
	
	tests := []struct {
		name          string
		setupExporter func(*DotEnvExporter)
		checkOutput   func(t *testing.T, output string)
	}{
		{
			name: "basic_export",
			setupExporter: func(e *DotEnvExporter) {
				e.Options.SortKeys = false
			},
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "API_KEY=secret123") {
					t.Error("Missing API_KEY in output")
				}
				if !strings.Contains(output, "DATABASE_URL=postgres://user:pass@localhost/db") {
					t.Error("Missing DATABASE_URL in output")
				}
			},
		},
		{
			name: "sorted_keys",
			setupExporter: func(e *DotEnvExporter) {
				e.Options.SortKeys = true
			},
			checkOutput: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				// First non-empty line should be API_KEY (alphabetically first)
				if !strings.HasPrefix(lines[0], "API_KEY=") {
					t.Errorf("Expected API_KEY first, got %s", lines[0])
				}
			},
		},
		{
			name: "quote_values",
			setupExporter: func(e *DotEnvExporter) {
				e.QuoteValues = true
			},
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, `API_KEY="secret123"`) {
					t.Error("Values should be quoted")
				}
			},
		},
		{
			name: "export_prefix",
			setupExporter: func(e *DotEnvExporter) {
				e.IncludeExport = true
			},
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "export API_KEY=") {
					t.Error("Missing export prefix")
				}
			},
		},
		{
			name: "exclude_empty",
			setupExporter: func(e *DotEnvExporter) {
				e.Options.IncludeEmpty = false
			},
			checkOutput: func(t *testing.T, output string) {
				if strings.Contains(output, "EMPTY_VAR") {
					t.Error("Empty variables should be excluded")
				}
			},
		},
		{
			name: "with_comments",
			setupExporter: func(e *DotEnvExporter) {
				e.Options.ShowComments = true
			},
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "# Environment variables exported by VaultEnv") {
					t.Error("Missing header comment")
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := NewDotEnvExporter()
			tt.setupExporter(exp)
			
			var buf bytes.Buffer
			err := exp.Export(vars, &buf)
			if err != nil {
				t.Fatalf("Export() error = %v", err)
			}
			
			tt.checkOutput(t, buf.String())
		})
	}
}

func TestDotEnvExporter_NeedsQuoting(t *testing.T) {
	exporter := NewDotEnvExporter()
	
	tests := []struct {
		value     string
		needsQuote bool
	}{
		{"simple", false},
		{"with space", true},
		{"with\ttab", true},
		{"with\nnewline", true},
		{`with"quote`, true},
		{`with'apostrophe`, true},
		{"with$dollar", true},
		{"with`backtick", true},
		{"with\\backslash", true},
		{"with#hash", true},
		{"normal-value_123", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := exporter.needsQuoting(tt.value); got != tt.needsQuote {
				t.Errorf("needsQuoting(%q) = %v, want %v", tt.value, got, tt.needsQuote)
			}
		})
	}
}

func TestJSONExporter(t *testing.T) {
	vars := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"EMPTY": "",
	}
	
	tests := []struct {
		name          string
		setupExporter func(*JSONExporter)
		checkOutput   func(t *testing.T, output string)
	}{
		{
			name: "pretty_print",
			setupExporter: func(e *JSONExporter) {
				e.PrettyPrint = true
			},
			checkOutput: func(t *testing.T, output string) {
				// Check for indentation
				if !strings.Contains(output, "  ") {
					t.Error("Expected indented JSON")
				}
				
				// Verify valid JSON
				var data map[string]string
				if err := json.Unmarshal([]byte(output), &data); err != nil {
					t.Errorf("Invalid JSON: %v", err)
				}
			},
		},
		{
			name: "compact",
			setupExporter: func(e *JSONExporter) {
				e.PrettyPrint = false
			},
			checkOutput: func(t *testing.T, output string) {
				// Should be on one line
				if strings.Count(output, "\n") > 1 {
					t.Error("Expected compact JSON")
				}
			},
		},
		{
			name: "exclude_empty",
			setupExporter: func(e *JSONExporter) {
				e.Options.IncludeEmpty = false
			},
			checkOutput: func(t *testing.T, output string) {
				var data map[string]string
				json.Unmarshal([]byte(output), &data)
				
				if _, exists := data["EMPTY"]; exists {
					t.Error("Empty value should be excluded")
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := NewJSONExporter()
			tt.setupExporter(exp)
			
			var buf bytes.Buffer
			err := exp.Export(vars, &buf)
			if err != nil {
				t.Fatalf("Export() error = %v", err)
			}
			
			tt.checkOutput(t, buf.String())
		})
	}
}

func TestYAMLExporter(t *testing.T) {
	exporter := NewYAMLExporter()
	
	vars := map[string]string{
		"KEY1":     "value1",
		"KEY2":     "value2",
		"MULTILINE": "line1\nline2",
		"EMPTY":    "",
	}
	
	var buf bytes.Buffer
	err := exporter.Export(vars, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	
	// Verify valid YAML
	var data map[string]string
	if err := yaml.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Errorf("Invalid YAML: %v", err)
	}
	
	// Check values
	if data["KEY1"] != "value1" {
		t.Errorf("KEY1 = %v, want value1", data["KEY1"])
	}
	
	// Test with comments
	exporter.Options.ShowComments = true
	buf.Reset()
	exporter.Export(vars, &buf)
	
	output := buf.String()
	if !strings.Contains(output, "# Environment variables exported by VaultEnv") {
		t.Error("Missing header comment")
	}
}

func TestShellExporter(t *testing.T) {
	vars := map[string]string{
		"SIMPLE":      "value",
		"WITH_SPACE":  "value with space",
		"WITH_QUOTE":  "value'with'quote",
		"WITH_DOLLAR": "value$dollar",
		"EMPTY":       "",
	}
	
	tests := []struct {
		name          string
		setupExporter func(*ShellExporter)
		checkOutput   func(t *testing.T, output string)
	}{
		{
			name: "with_export",
			setupExporter: func(e *ShellExporter) {
				e.ExportVars = true
				e.SetCommand = "export"
			},
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "export SIMPLE=") {
					t.Error("Missing export command")
				}
				if !strings.Contains(output, "#!/bin/bash") {
					t.Error("Missing shebang")
				}
			},
		},
		{
			name: "without_export",
			setupExporter: func(e *ShellExporter) {
				e.ExportVars = false
			},
			checkOutput: func(t *testing.T, output string) {
				if strings.Contains(output, "export ") {
					t.Error("Should not contain export")
				}
			},
		},
		{
			name: "proper_escaping",
			setupExporter: func(e *ShellExporter) {},
			checkOutput: func(t *testing.T, output string) {
				// Check single quote escaping
				if !strings.Contains(output, `WITH_QUOTE='value'"'"'with'"'"'quote'`) {
					t.Error("Incorrect quote escaping")
				}
				// Check space handling
				if !strings.Contains(output, `WITH_SPACE='value with space'`) {
					t.Error("Incorrect space handling")
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := NewShellExporter()
			tt.setupExporter(exp)
			
			var buf bytes.Buffer
			err := exp.Export(vars, &buf)
			if err != nil {
				t.Fatalf("Export() error = %v", err)
			}
			
			tt.checkOutput(t, buf.String())
		})
	}
}

func TestDockerExporter(t *testing.T) {
	exporter := NewDockerExporter()
	
	vars := map[string]string{
		"API_KEY":     "secret123",
		"PORT":        "8080",
		"WITH_SPACE":  "value with space",
		"WITH_DOLLAR": "value$dollar",
		"EMPTY":       "",
	}
	
	var buf bytes.Buffer
	err := exporter.Export(vars, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	
	output := buf.String()
	
	// Check ENV format
	if !strings.Contains(output, "ENV API_KEY=secret123") {
		t.Error("Missing ENV instruction")
	}
	
	// Check escaping
	if !strings.Contains(output, `ENV WITH_SPACE="value with space"`) {
		t.Error("Space values should be quoted")
	}
	
	// Check empty values are excluded by default
	if strings.Contains(output, "EMPTY") {
		t.Error("Empty values should be excluded")
	}
	
	// Check comments
	if !strings.Contains(output, "# Environment variables exported by VaultEnv") {
		t.Error("Missing header comment")
	}
}

func TestTemplateExporter(t *testing.T) {
	// Test template
	tmplContent := `
Variables:
{{range .Keys -}}
{{. }} = {{index $.Variables .}}
{{end}}`
	
	exporter, err := NewTemplateExporter(tmplContent, "test")
	if err != nil {
		t.Fatalf("NewTemplateExporter() error = %v", err)
	}
	
	vars := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}
	
	var buf bytes.Buffer
	err = exporter.Export(vars, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	
	output := buf.String()
	
	// Check output contains expected values
	if !strings.Contains(output, "KEY1 = value1") {
		t.Error("Missing KEY1 in template output")
	}
	if !strings.Contains(output, "KEY2 = value2") {
		t.Error("Missing KEY2 in template output")
	}
	
	// Test with additional template data
	exporter.Options.TemplateData = map[string]interface{}{
		"AppName": "MyApp",
	}
	
	// Test invalid template
	_, err = NewTemplateExporter("{{.Invalid}", "bad")
	if err == nil {
		t.Error("Expected error for invalid template")
	}
}

func TestExporterFactory(t *testing.T) {
	factory := NewExporterFactory()
	
	tests := []struct {
		format      string
		wantType    string
		wantErr     bool
	}{
		{"dotenv", "*export.DotEnvExporter", false},
		{"env", "*export.DotEnvExporter", false},
		{"json", "*export.JSONExporter", false},
		{"yaml", "*export.YAMLExporter", false},
		{"yml", "*export.YAMLExporter", false},
		{"shell", "*export.ShellExporter", false},
		{"sh", "*export.ShellExporter", false},
		{"bash", "*export.ShellExporter", false},
		{"docker", "*export.DockerExporter", false},
		{"dockerfile", "*export.DockerExporter", false},
		{"unknown", "", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			exporter, err := factory.CreateExporter(tt.format)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateExporter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				gotType := fmt.Sprintf("%T", exporter)
				if gotType != tt.wantType {
					t.Errorf("CreateExporter() type = %v, want %v", gotType, tt.wantType)
				}
			}
		})
	}
}

func TestExporterFactory_GetSupportedFormats(t *testing.T) {
	factory := NewExporterFactory()
	formats := factory.GetSupportedFormats()
	
	expectedFormats := []string{"dotenv", "json", "yaml", "shell", "docker"}
	
	if len(formats) != len(expectedFormats) {
		t.Errorf("GetSupportedFormats() returned %d formats, want %d", len(formats), len(expectedFormats))
	}
	
	for _, expected := range expectedFormats {
		found := false
		for _, format := range formats {
			if format == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing format: %s", expected)
		}
	}
}

func TestExporterFileExtensions(t *testing.T) {
	tests := []struct {
		exporter Exporter
		wantExt  string
	}{
		{NewDotEnvExporter(), ".env"},
		{NewJSONExporter(), ".json"},
		{NewYAMLExporter(), ".yaml"},
		{NewShellExporter(), ".sh"},
		{NewDockerExporter(), ".dockerfile"},
	}
	
	for _, tt := range tests {
		t.Run(tt.wantExt, func(t *testing.T) {
			if got := tt.exporter.FileExtension(); got != tt.wantExt {
				t.Errorf("FileExtension() = %v, want %v", got, tt.wantExt)
			}
		})
	}
}

func TestExporterContentTypes(t *testing.T) {
	tests := []struct {
		exporter    Exporter
		wantType    string
	}{
		{NewDotEnvExporter(), "text/plain"},
		{NewJSONExporter(), "application/json"},
		{NewYAMLExporter(), "application/x-yaml"},
		{NewShellExporter(), "application/x-sh"},
		{NewDockerExporter(), "text/plain"},
	}
	
	for _, tt := range tests {
		t.Run(tt.wantType, func(t *testing.T) {
			if got := tt.exporter.ContentType(); got != tt.wantType {
				t.Errorf("ContentType() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test filterEmptyValues
	vars := map[string]string{
		"KEY1":  "value1",
		"EMPTY": "",
		"KEY2":  "value2",
	}
	
	filtered := filterEmptyValues(vars)
	if len(filtered) != 2 {
		t.Errorf("filterEmptyValues() returned %d items, want 2", len(filtered))
	}
	
	if _, exists := filtered["EMPTY"]; exists {
		t.Error("Empty value should be filtered out")
	}
	
	// Test getSortedKeys
	keys := getSortedKeys(vars, true)
	if len(keys) != 3 {
		t.Errorf("getSortedKeys() returned %d keys, want 3", len(keys))
	}
	
	// Verify sorted
	if keys[0] != "EMPTY" || keys[1] != "KEY1" || keys[2] != "KEY2" {
		t.Error("Keys not properly sorted")
	}
	
	// Test unsorted
	unsortedKeys := getSortedKeys(vars, false)
	if len(unsortedKeys) != 3 {
		t.Errorf("getSortedKeys() returned %d keys, want 3", len(unsortedKeys))
	}
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"with space", "'with space'"},
		{"with'quote", "'with'\"'\"'quote'"},
		{"multiple'quotes'here", "'multiple'\"'\"'quotes'\"'\"'here'"},
		{"", "''"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := shellEscape(tt.input); got != tt.want {
				t.Errorf("shellEscape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDockerEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with space", `"with space"`},
		{"with$dollar", `"with$dollar"`},
		{`with"quote`, `"with\"quote"`},
		{"normal-value_123", "normal-value_123"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := dockerEscape(tt.input); got != tt.want {
				t.Errorf("dockerEscape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func BenchmarkDotEnvExporter(b *testing.B) {
	exporter := NewDotEnvExporter()
	vars := make(map[string]string)
	for i := 0; i < 100; i++ {
		vars[fmt.Sprintf("KEY_%d", i)] = fmt.Sprintf("value_%d", i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		exporter.Export(vars, &buf)
	}
}

func BenchmarkJSONExporter(b *testing.B) {
	exporter := NewJSONExporter()
	vars := make(map[string]string)
	for i := 0; i < 100; i++ {
		vars[fmt.Sprintf("KEY_%d", i)] = fmt.Sprintf("value_%d", i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		exporter.Export(vars, &buf)
	}
}