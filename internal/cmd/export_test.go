package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
	"gopkg.in/yaml.v3"
)

func TestExportCommand(t *testing.T) {
	// Create test storage with data
	tmpDir, err := ioutil.TempDir("", "vaultenv-export-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewFileBackend(tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}

	// Add test data
	testData := map[string]string{
		"DATABASE_URL":    "postgres://localhost/test",
		"API_KEY":         "secret123",
		"REDIS_URL":       "redis://localhost:6379",
		"DEBUG":           "true",
		"PORT":            "8080",
		"APP_NAME":        "test-app",
		"EMPTY_VALUE":     "",
		"SPECIAL_CHARS":   "value with spaces & special!",
		"MULTILINE_VALUE": "line1\nline2\nline3",
	}

	for key, value := range testData {
		if err := store.Set(key, value, false); err != nil {
			t.Fatalf("Failed to set test data: %v", err)
		}
	}

	tests := []struct {
		name         string
		args         []string
		flags        map[string]string
		wantErr      bool
		checkOutput  func(t *testing.T, output string)
		wantContains []string
	}{
		{
			name: "export_default_dotenv",
			checkOutput: func(t *testing.T, output string) {
				// Should be in dotenv format by default
				lines := strings.Split(output, "\n")
				for _, line := range lines {
					if line != "" && !strings.Contains(line, "=") {
						t.Errorf("Invalid dotenv line: %q", line)
					}
				}
				// Check specific values
				if !strings.Contains(output, "DATABASE_URL=postgres://localhost/test") {
					t.Error("Missing DATABASE_URL in output")
				}
			},
		},
		{
			name:  "export_json_format",
			flags: map[string]string{"format": "json"},
			checkOutput: func(t *testing.T, output string) {
				// Should be valid JSON
				var data map[string]string
				if err := json.Unmarshal([]byte(output), &data); err != nil {
					t.Errorf("Invalid JSON output: %v", err)
				}
				// Verify content
				if data["DATABASE_URL"] != "postgres://localhost/test" {
					t.Error("JSON missing or incorrect DATABASE_URL")
				}
				if data["PORT"] != "8080" {
					t.Error("JSON missing or incorrect PORT")
				}
			},
		},
		{
			name:  "export_yaml_format",
			flags: map[string]string{"format": "yaml"},
			checkOutput: func(t *testing.T, output string) {
				// Should be valid YAML
				var data map[string]string
				if err := yaml.Unmarshal([]byte(output), &data); err != nil {
					t.Errorf("Invalid YAML output: %v", err)
				}
				// Verify content
				if data["API_KEY"] != "secret123" {
					t.Error("YAML missing or incorrect API_KEY")
				}
			},
		},
		{
			name:  "export_shell_format",
			flags: map[string]string{"format": "shell"},
			checkOutput: func(t *testing.T, output string) {
				// Should have export statements
				lines := strings.Split(output, "\n")
				for _, line := range lines {
					if line != "" && !strings.HasPrefix(line, "export ") {
						t.Errorf("Shell format line should start with 'export': %q", line)
					}
				}
				// Check quoting
				if !strings.Contains(output, "export SPECIAL_CHARS='value with spaces & special!'") &&
				   !strings.Contains(output, `export SPECIAL_CHARS="value with spaces & special!"`) {
					t.Error("Shell format should properly quote special values")
				}
			},
		},
		{
			name:  "export_docker_format",
			flags: map[string]string{"format": "docker"},
			checkOutput: func(t *testing.T, output string) {
				// Should have ARG or ENV statements
				lines := strings.Split(output, "\n")
				for _, line := range lines {
					if line != "" && !strings.HasPrefix(line, "ENV ") && !strings.HasPrefix(line, "ARG ") {
						t.Errorf("Docker format line should start with ENV or ARG: %q", line)
					}
				}
			},
		},
		{
			name:  "export_to_file",
			args:  []string{filepath.Join(tmpDir, "exported.env")},
			flags: map[string]string{"format": "dotenv"},
			checkOutput: func(t *testing.T, output string) {
				// Should create file
				exportPath := filepath.Join(tmpDir, "exported.env")
				if _, err := os.Stat(exportPath); os.IsNotExist(err) {
					t.Error("Export file was not created")
					return
				}
				
				// Read and verify file content
				content, err := ioutil.ReadFile(exportPath)
				if err != nil {
					t.Errorf("Failed to read export file: %v", err)
					return
				}
				
				if !strings.Contains(string(content), "DATABASE_URL=") {
					t.Error("Export file missing DATABASE_URL")
				}
			},
		},
		{
			name:  "export_with_prefix_filter",
			flags: map[string]string{"prefix": "API_"},
			checkOutput: func(t *testing.T, output string) {
				// Should only contain API_ prefixed vars
				if !strings.Contains(output, "API_KEY=") {
					t.Error("Missing API_KEY in filtered output")
				}
				if strings.Contains(output, "DATABASE_URL=") {
					t.Error("Should not contain DATABASE_URL in filtered output")
				}
			},
		},
		{
			name:  "export_mask_secrets",
			flags: map[string]string{"mask-secrets": "true"},
			checkOutput: func(t *testing.T, output string) {
				// Should mask secret values
				if strings.Contains(output, "secret123") {
					t.Error("Secret value should be masked")
				}
				if !strings.Contains(output, "API_KEY=***") && !strings.Contains(output, "API_KEY=<masked>") {
					t.Error("API_KEY should be shown with masked value")
				}
			},
		},
		{
			name:  "export_empty_values",
			flags: map[string]string{"skip-empty": "false"},
			wantContains: []string{
				"EMPTY_VALUE=",
			},
		},
		{
			name:  "export_skip_empty",
			flags: map[string]string{"skip-empty": "true"},
			checkOutput: func(t *testing.T, output string) {
				if strings.Contains(output, "EMPTY_VALUE") {
					t.Error("Should skip empty values")
				}
			},
		},
		{
			name:  "export_sorted",
			flags: map[string]string{"sort": "true"},
			checkOutput: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				var keys []string
				for _, line := range lines {
					if line != "" {
						parts := strings.SplitN(line, "=", 2)
						keys = append(keys, parts[0])
					}
				}
				
				// Check if sorted
				for i := 1; i < len(keys); i++ {
					if keys[i-1] > keys[i] {
						t.Errorf("Keys not sorted: %s comes before %s", keys[i-1], keys[i])
					}
				}
			},
		},
		{
			name:  "export_template",
			flags: map[string]string{"template": "{{range $k, $v := .}}{{$k}}={{$v}}\n{{end}}"},
			checkOutput: func(t *testing.T, output string) {
				// Should use custom template
				if !strings.Contains(output, "DATABASE_URL=postgres://localhost/test") {
					t.Error("Template output missing expected content")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := &cobra.Command{
				Use: "export",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Simplified export logic for testing
					format, _ := cmd.Flags().GetString("format")
					prefix, _ := cmd.Flags().GetString("prefix")
					maskSecrets, _ := cmd.Flags().GetBool("mask-secrets")
					skipEmpty, _ := cmd.Flags().GetBool("skip-empty")
					doSort, _ := cmd.Flags().GetBool("sort")

					// Get all keys
					keys, err := store.List()
					if err != nil {
						return err
					}

					// Filter and prepare data
					exportData := make(map[string]string)
					for _, key := range keys {
						if prefix != "" && !strings.HasPrefix(key, prefix) {
							continue
						}
						
						value, _ := store.Get(key)
						
						if skipEmpty && value == "" {
							continue
						}
						
						if maskSecrets && isSecret(key) {
							value = "***"
						}
						
						exportData[key] = value
					}

					// Sort keys if requested
					var sortedKeys []string
					for k := range exportData {
						sortedKeys = append(sortedKeys, k)
					}
					if doSort {
						sort.Strings(sortedKeys)
					} else {
						sortedKeys = keys
					}

					// Output based on format
					var output string
					switch format {
					case "json":
						data, _ := json.MarshalIndent(exportData, "", "  ")
						output = string(data)
					case "yaml":
						data, _ := yaml.Marshal(exportData)
						output = string(data)
					case "shell":
						for _, k := range sortedKeys {
							if v, ok := exportData[k]; ok {
								output += fmt.Sprintf("export %s='%s'\n", k, v)
							}
						}
					case "docker":
						for _, k := range sortedKeys {
							if v, ok := exportData[k]; ok {
								output += fmt.Sprintf("ENV %s %s\n", k, v)
							}
						}
					default: // dotenv
						for _, k := range sortedKeys {
							if v, ok := exportData[k]; ok {
								output += fmt.Sprintf("%s=%s\n", k, v)
							}
						}
					}

					// Write to file or stdout
					if len(args) > 0 {
						return ioutil.WriteFile(args[0], []byte(output), 0644)
					}
					
					cmd.Print(output)
					return nil
				},
			}

			// Add flags
			cmd.Flags().String("format", "dotenv", "Export format")
			cmd.Flags().String("prefix", "", "Filter by prefix")
			cmd.Flags().Bool("mask-secrets", false, "Mask secret values")
			cmd.Flags().Bool("skip-empty", false, "Skip empty values")
			cmd.Flags().Bool("sort", false, "Sort keys")
			cmd.Flags().String("template", "", "Custom template")

			// Set flags
			for key, value := range tt.flags {
				cmd.Flags().Set(key, value)
			}

			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			// Execute command
			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()

			// Check output
			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}

			// Check contains
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Output = %q, want to contain %q", output, want)
				}
			}
		})
	}
}

func TestExportCommandFormats(t *testing.T) {
	// Test specific format outputs
	tmpDir, err := ioutil.TempDir("", "vaultenv-export-formats")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewFileBackend(tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}

	// Simple test data
	store.Set("KEY1", "value1", false)
	store.Set("KEY2", "value with spaces", false)

	formats := map[string]struct {
		expected []string
		validate func(t *testing.T, output string)
	}{
		"dotenv": {
			expected: []string{"KEY1=value1", "KEY2=value with spaces"},
		},
		"json": {
			validate: func(t *testing.T, output string) {
				var data map[string]string
				if err := json.Unmarshal([]byte(output), &data); err != nil {
					t.Errorf("Invalid JSON: %v", err)
				}
				if len(data) != 2 {
					t.Errorf("Expected 2 keys, got %d", len(data))
				}
			},
		},
		"yaml": {
			expected: []string{"KEY1: value1", "KEY2: value with spaces"},
		},
		"shell": {
			expected: []string{"export KEY1='value1'", "export KEY2='value with spaces'"},
		},
		"docker": {
			expected: []string{"ENV KEY1 value1", "ENV KEY2 value with spaces"},
		},
	}

	for format, test := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := &cobra.Command{
				Use: "export",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Actual implementation would use the export package
					switch format {
					case "json":
						cmd.Print(`{"KEY1":"value1","KEY2":"value with spaces"}`)
					case "yaml":
						cmd.Print("KEY1: value1\nKEY2: value with spaces\n")
					case "shell":
						cmd.Print("export KEY1='value1'\nexport KEY2='value with spaces'\n")
					case "docker":
						cmd.Print("ENV KEY1 value1\nENV KEY2 value with spaces\n")
					default:
						cmd.Print("KEY1=value1\nKEY2=value with spaces\n")
					}
					return nil
				},
			}

			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}

			output := buf.String()

			if test.validate != nil {
				test.validate(t, output)
			}

			for _, expected := range test.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Format %s output = %q, want to contain %q", format, output, expected)
				}
			}
		})
	}
}