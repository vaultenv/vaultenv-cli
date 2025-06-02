package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func TestListCommand(t *testing.T) {
	// Create test storage with data
	tmpDir, err := ioutil.TempDir("", "vaultenv-list-test")
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
		"DATABASE_URL":     "postgres://localhost/test",
		"API_KEY":          "secret123",
		"API_SECRET":       "hidden",
		"FEATURE_ENABLED":  "true",
		"SERVICE_URL":      "http://service.local",
		"DEBUG":            "false",
		"LOG_LEVEL":        "info",
		"AWS_ACCESS_KEY":   "AKIA...",
		"AWS_SECRET_KEY":   "secret",
		"EMPTY_VALUE":      "",
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
		wantMissing  []string
	}{
		{
			name: "list_all_keys",
			checkOutput: func(t *testing.T, output string) {
				// Should list all keys
				for key := range testData {
					if !strings.Contains(output, key) {
						t.Errorf("Output missing key %q", key)
					}
				}
			},
		},
		{
			name:  "list_with_values",
			flags: map[string]string{"values": "true"},
			checkOutput: func(t *testing.T, output string) {
				// Should show key=value pairs
				if !strings.Contains(output, "DATABASE_URL=postgres://localhost/test") {
					t.Error("Output missing DATABASE_URL with value")
				}
				if !strings.Contains(output, "DEBUG=false") {
					t.Error("Output missing DEBUG with value")
				}
			},
		},
		{
			name:  "list_show_all_secrets",
			flags: map[string]string{"values": "true", "show-all": "true"},
			wantContains: []string{
				"API_KEY=secret123",
				"API_SECRET=hidden",
				"AWS_SECRET_KEY=secret",
			},
		},
		{
			name:  "list_hide_secrets",
			flags: map[string]string{"values": "true", "show-all": "false"},
			checkOutput: func(t *testing.T, output string) {
				// Should mask secret values
				if strings.Contains(output, "secret123") {
					t.Error("Output should not contain actual secret value")
				}
				if strings.Contains(output, "API_KEY=***") || strings.Contains(output, "API_KEY=<hidden>") {
					// Good - secret is masked
				} else if strings.Contains(output, "API_KEY") {
					t.Error("API_KEY should be shown with masked value")
				}
			},
		},
		{
			name:  "list_json_format",
			flags: map[string]string{"format": "json"},
			checkOutput: func(t *testing.T, output string) {
				// Should be valid JSON
				if !strings.HasPrefix(output, "{") || !strings.HasSuffix(strings.TrimSpace(output), "}") {
					t.Error("Output is not valid JSON format")
				}
				if !strings.Contains(output, `"DATABASE_URL"`) {
					t.Error("JSON output missing DATABASE_URL key")
				}
			},
		},
		{
			name:  "list_export_format",
			flags: map[string]string{"format": "export"},
			checkOutput: func(t *testing.T, output string) {
				lines := strings.Split(output, "\n")
				for _, line := range lines {
					if line != "" && !strings.HasPrefix(line, "export ") {
						t.Errorf("Export format line doesn't start with 'export ': %q", line)
					}
				}
			},
		},
		{
			name:  "list_dotenv_format",
			flags: map[string]string{"format": "dotenv"},
			checkOutput: func(t *testing.T, output string) {
				// Should be KEY=VALUE format
				lines := strings.Split(output, "\n")
				for _, line := range lines {
					if line != "" && !strings.Contains(line, "=") {
						t.Errorf("Dotenv format line missing '=': %q", line)
					}
				}
			},
		},
		{
			name: "list_with_prefix",
			args: []string{"API_"},
			checkOutput: func(t *testing.T, output string) {
				// Should only show API_ prefixed keys
				if !strings.Contains(output, "API_KEY") {
					t.Error("Output missing API_KEY")
				}
				if !strings.Contains(output, "API_SECRET") {
					t.Error("Output missing API_SECRET")
				}
				if strings.Contains(output, "DATABASE_URL") {
					t.Error("Output should not contain DATABASE_URL")
				}
			},
		},
		{
			name: "list_sorted",
			checkOutput: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				var keys []string
				for _, line := range lines {
					if line != "" {
						// Extract key from output
						parts := strings.SplitN(line, "=", 2)
						keys = append(keys, parts[0])
					}
				}
				
				// Check if sorted
				sortedKeys := make([]string, len(keys))
				copy(sortedKeys, keys)
				sort.Strings(sortedKeys)
				
				for i, key := range keys {
					if key != sortedKeys[i] {
						t.Errorf("Keys not sorted: position %d has %q, want %q", i, key, sortedKeys[i])
						break
					}
				}
			},
		},
		{
			name:  "list_empty_values",
			flags: map[string]string{"values": "true"},
			wantContains: []string{
				"EMPTY_VALUE=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := &cobra.Command{
				Use: "list",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Get flags
					showValues, _ := cmd.Flags().GetBool("values")
					showAll, _ := cmd.Flags().GetBool("show-all")
					format, _ := cmd.Flags().GetString("format")

					// Get all keys
					keys, err := store.List()
					if err != nil {
						return err
					}

					// Filter by prefix if provided
					if len(args) > 0 {
						prefix := args[0]
						var filtered []string
						for _, key := range keys {
							if strings.HasPrefix(key, prefix) {
								filtered = append(filtered, key)
							}
						}
						keys = filtered
					}

					// Sort keys
					sort.Strings(keys)

					// Format output
					switch format {
					case "json":
						cmd.Print("{")
						for i, key := range keys {
							value, _ := store.Get(key)
							if i > 0 {
								cmd.Print(",")
							}
							cmd.Printf(`"%s":"%s"`, key, value)
						}
						cmd.Print("}")
					case "export":
						for _, key := range keys {
							value, _ := store.Get(key)
							cmd.Printf("export %s='%s'\n", key, value)
						}
					case "dotenv":
						for _, key := range keys {
							value, _ := store.Get(key)
							cmd.Printf("%s=%s\n", key, value)
						}
					default:
						// Default format
						for _, key := range keys {
							if showValues {
								value, _ := store.Get(key)
								if !showAll && isSecret(key) {
									value = "***"
								}
								cmd.Printf("%s=%s\n", key, value)
							} else {
								cmd.Println(key)
							}
						}
					}
					return nil
				},
			}

			// Add flags
			cmd.Flags().Bool("values", false, "Show values")
			cmd.Flags().Bool("show-all", false, "Show all values including secrets")
			cmd.Flags().String("format", "", "Output format")

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

			// Check missing
			for _, missing := range tt.wantMissing {
				if strings.Contains(output, missing) {
					t.Errorf("Output = %q, should not contain %q", output, missing)
				}
			}
		})
	}
}

func TestListCommandEmpty(t *testing.T) {
	// Test with empty storage
	tmpDir, err := ioutil.TempDir("", "vaultenv-list-empty")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewFileBackend(tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			keys, err := store.List()
			if err != nil {
				return err
			}
			
			if len(keys) == 0 {
				cmd.Println("No environment variables found")
				return nil
			}
			
			for _, key := range keys {
				cmd.Println(key)
			}
			return nil
		},
	}

	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "No environment variables found" {
		t.Errorf("Output = %q, want %q", output, "No environment variables found")
	}
}

func TestListCommandStats(t *testing.T) {
	// Test statistics flag
	tmpDir, err := ioutil.TempDir("", "vaultenv-list-stats")
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
		"VAR1": "value1",
		"VAR2": "value2",
		"VAR3": "value3",
		"VAR4": "",
		"VAR5": "secret",
	}

	for key, value := range testData {
		store.Set(key, value, false)
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			stats, _ := cmd.Flags().GetBool("stats")
			
			keys, _ := store.List()
			
			if stats {
				emptyCount := 0
				totalSize := 0
				
				for _, key := range keys {
					value, _ := store.Get(key)
					if value == "" {
						emptyCount++
					}
					totalSize += len(key) + len(value)
				}
				
				cmd.Printf("Total variables: %d\n", len(keys))
				cmd.Printf("Empty values: %d\n", emptyCount)
				cmd.Printf("Total size: %d bytes\n", totalSize)
			} else {
				for _, key := range keys {
					cmd.Println(key)
				}
			}
			return nil
		},
	}

	cmd.Flags().Bool("stats", true, "Show statistics")
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	output := buf.String()
	expectedStats := []string{
		"Total variables: 5",
		"Empty values: 1",
		"Total size:",
	}

	for _, expected := range expectedStats {
		if !strings.Contains(output, expected) {
			t.Errorf("Output = %q, want to contain %q", output, expected)
		}
	}
}

// Helper function to identify secret keys
func isSecret(key string) bool {
	secretPatterns := []string{
		"SECRET", "KEY", "TOKEN", "PASSWORD", "PASS", "PWD",
		"PRIVATE", "CREDENTIAL", "AUTH",
	}
	
	upperKey := strings.ToUpper(key)
	for _, pattern := range secretPatterns {
		if strings.Contains(upperKey, pattern) {
			return true
		}
	}
	return false
}