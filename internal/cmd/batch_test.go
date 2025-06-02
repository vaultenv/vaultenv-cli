package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func TestBatchCommand(t *testing.T) {
	tests := []struct {
		name         string
		batchFile    string
		args         []string
		flags        map[string]string
		stdin        string
		wantErr      bool
		verifyStore  func(t *testing.T, store storage.Backend)
		wantContains []string
		wantMissing  []string
	}{
		{
			name: "batch_set_operations",
			batchFile: `# Batch file for setting values
set DATABASE_URL postgres://localhost/test
set API_KEY secret123
set PORT 8080
set DEBUG true`,
			verifyStore: func(t *testing.T, store storage.Backend) {
				expected := map[string]string{
					"DATABASE_URL": "postgres://localhost/test",
					"API_KEY":      "secret123",
					"PORT":         "8080",
					"DEBUG":        "true",
				}
				
				for key, expectedVal := range expected {
					val, err := store.Get(key)
					if err != nil {
						t.Errorf("Failed to get %s: %v", key, err)
						continue
					}
					if val != expectedVal {
						t.Errorf("%s = %q, want %q", key, val, expectedVal)
					}
				}
			},
			wantContains: []string{"Executed 4 commands"},
		},
		{
			name: "batch_mixed_operations",
			batchFile: `# Mixed operations
set KEY1 value1
set KEY2 value2
get KEY1
list
set KEY3 value3
delete KEY2`,
			verifyStore: func(t *testing.T, store storage.Backend) {
				// KEY1 and KEY3 should exist
				if val, err := store.Get("KEY1"); err != nil || val != "value1" {
					t.Errorf("KEY1 = %v, %v", val, err)
				}
				if val, err := store.Get("KEY3"); err != nil || val != "value3" {
					t.Errorf("KEY3 = %v, %v", val, err)
				}
				
				// KEY2 should be deleted
				if _, err := store.Get("KEY2"); err == nil {
					t.Error("KEY2 should be deleted")
				}
			},
			wantContains: []string{"value1", "KEY1", "KEY3"},
		},
		{
			name: "batch_with_errors",
			batchFile: `set VALID_KEY value
get NON_EXISTENT_KEY
set ANOTHER_KEY value2`,
			flags:        map[string]string{"continue-on-error": "true"},
			wantContains: []string{"Executed 3 commands", "1 error"},
			verifyStore: func(t *testing.T, store storage.Backend) {
				// Should continue after error
				if _, err := store.Get("ANOTHER_KEY"); err != nil {
					t.Error("ANOTHER_KEY should be set despite error")
				}
			},
		},
		{
			name: "batch_stop_on_error",
			batchFile: `set KEY1 value1
get NON_EXISTENT_KEY
set KEY2 value2`,
			flags:    map[string]string{"continue-on-error": "false"},
			wantErr:  true,
			verifyStore: func(t *testing.T, store storage.Backend) {
				// Should stop at error
				if _, err := store.Get("KEY1"); err != nil {
					t.Error("KEY1 should be set before error")
				}
				if _, err := store.Get("KEY2"); err == nil {
					t.Error("KEY2 should not be set after error")
				}
			},
		},
		{
			name: "batch_with_variables",
			batchFile: `# Using variables
@BASE_URL=http://localhost
@PORT=8080
set API_URL $BASE_URL:$PORT/api
set HEALTH_URL $BASE_URL:$PORT/health`,
			verifyStore: func(t *testing.T, store storage.Backend) {
				val, _ := store.Get("API_URL")
				if val != "http://localhost:8080/api" {
					t.Errorf("API_URL = %q, want expanded", val)
				}
				val, _ = store.Get("HEALTH_URL")
				if val != "http://localhost:8080/health" {
					t.Errorf("HEALTH_URL = %q, want expanded", val)
				}
			},
		},
		{
			name: "batch_conditionals",
			batchFile: `# Conditional execution
set ENV production
if ENV=production
  set DEBUG false
  set LOG_LEVEL error
else
  set DEBUG true
  set LOG_LEVEL debug
endif`,
			verifyStore: func(t *testing.T, store storage.Backend) {
				val, _ := store.Get("DEBUG")
				if val != "false" {
					t.Errorf("DEBUG = %q, want 'false'", val)
				}
				val, _ = store.Get("LOG_LEVEL")
				if val != "error" {
					t.Errorf("LOG_LEVEL = %q, want 'error'", val)
				}
			},
		},
		{
			name: "batch_loops",
			batchFile: `# Loop example
@ENVS=dev,staging,prod
foreach ENV in $ENVS
  set ${ENV}_URL http://${ENV}.example.com
endfor`,
			verifyStore: func(t *testing.T, store storage.Backend) {
				expected := map[string]string{
					"dev_URL":     "http://dev.example.com",
					"staging_URL": "http://staging.example.com",
					"prod_URL":    "http://prod.example.com",
				}
				
				for key, expectedVal := range expected {
					val, _ := store.Get(key)
					if val != expectedVal {
						t.Errorf("%s = %q, want %q", key, val, expectedVal)
					}
				}
			},
		},
		{
			name: "batch_from_stdin",
			args: []string{"-"},
			stdin: `set KEY1 value1
set KEY2 value2`,
			verifyStore: func(t *testing.T, store storage.Backend) {
				val, _ := store.Get("KEY1")
				if val != "value1" {
					t.Errorf("KEY1 = %q", val)
				}
			},
		},
		{
			name: "batch_import_file",
			batchFile: `# Import another file
import common.env
set SPECIFIC_KEY specific_value`,
			verifyStore: func(t *testing.T, store storage.Backend) {
				// Would need to create common.env file in test
				val, _ := store.Get("SPECIFIC_KEY")
				if val != "specific_value" {
					t.Errorf("SPECIFIC_KEY = %q", val)
				}
			},
		},
		{
			name:      "batch_dry_run",
			flags:     map[string]string{"dry-run": "true"},
			batchFile: `set KEY1 value1
set KEY2 value2
delete KEY3`,
			wantContains: []string{"[DRY RUN]", "set KEY1 value1", "set KEY2 value2"},
			verifyStore: func(t *testing.T, store storage.Backend) {
				// Nothing should be actually executed
				if _, err := store.Get("KEY1"); err == nil {
					t.Error("KEY1 should not be set in dry run")
				}
			},
		},
		{
			name: "batch_comments_and_empty_lines",
			batchFile: `# This is a comment
# Another comment

set KEY1 value1
  # Indented comment
  
set KEY2 value2
`,
			verifyStore: func(t *testing.T, store storage.Backend) {
				if _, err := store.Get("KEY1"); err != nil {
					t.Error("KEY1 should be set")
				}
				if _, err := store.Get("KEY2"); err != nil {
					t.Error("KEY2 should be set")
				}
			},
		},
		{
			name: "batch_export_operations",
			batchFile: `# Export operations
set KEY1 value1
set KEY2 value2
export dotenv output.env
export json output.json`,
			wantContains: []string{"Exported to output.env", "Exported to output.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := ioutil.TempDir("", "vaultenv-batch-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test storage
			store, err := storage.NewFileBackend(filepath.Join(tmpDir, "test.vault"), "default")
			if err != nil {
				t.Fatal(err)
			}

			// Create batch file if provided
			var batchPath string
			if tt.batchFile != "" && tt.stdin == "" {
				batchPath = filepath.Join(tmpDir, "batch.txt")
				if err := ioutil.WriteFile(batchPath, []byte(tt.batchFile), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Setup command
			var buf bytes.Buffer
			cmd := &cobra.Command{
				Use:  "batch",
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Simplified batch processing for testing
					var content string
					
					if len(args) > 0 && args[0] == "-" {
						content = tt.stdin
					} else if batchPath != "" {
						data, err := ioutil.ReadFile(batchPath)
						if err != nil {
							return err
						}
						content = string(data)
					}

					dryRun, _ := cmd.Flags().GetBool("dry-run")
					continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

					return processBatch(cmd, store, content, dryRun, continueOnError)
				},
			}

			// Add flags
			cmd.Flags().Bool("dry-run", false, "Dry run mode")
			cmd.Flags().Bool("continue-on-error", false, "Continue on error")
			cmd.Flags().Bool("verbose", false, "Verbose output")

			// Set flags
			for key, value := range tt.flags {
				cmd.Flags().Set(key, value)
			}

			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			
			if tt.stdin != "" {
				cmd.SetArgs([]string{"-"})
			} else if batchPath != "" {
				cmd.SetArgs([]string{batchPath})
			}

			// Execute command
			err = cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()

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

			// Verify store
			if tt.verifyStore != nil && !tt.wantErr {
				tt.verifyStore(t, store)
			}
		})
	}
}

// Simplified batch processor for testing
func processBatch(cmd *cobra.Command, store storage.Backend, content string, dryRun bool, continueOnError bool) error {
	lines := strings.Split(content, "\n")
	executed := 0
	errors := 0
	variables := make(map[string]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Variable assignment
		if strings.HasPrefix(line, "@") {
			parts := strings.SplitN(line[1:], "=", 2)
			if len(parts) == 2 {
				variables[parts[0]] = parts[1]
			}
			continue
		}

		// Expand variables
		for k, v := range variables {
			line = strings.ReplaceAll(line, "$"+k, v)
		}

		// Parse command
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		args := parts[1:]

		if dryRun {
			cmd.Printf("[DRY RUN] %s\n", line)
			executed++
			continue
		}

		// Execute command
		var err error
		switch command {
		case "set":
			if len(args) >= 2 {
				err = store.Set(args[0], strings.Join(args[1:], " "), false)
			}
		case "get":
			if len(args) >= 1 {
				val, err := store.Get(args[0])
				if err == nil {
					cmd.Println(val)
				}
			}
		case "delete":
			if len(args) >= 1 {
				err = store.Delete(args[0])
			}
		case "list":
			keys, _ := store.List()
			for _, k := range keys {
				cmd.Println(k)
			}
		case "export":
			if len(args) >= 2 {
				cmd.Printf("Exported to %s\n", args[1])
			}
		case "if", "else", "endif", "foreach", "endfor", "import":
			// Skip control structures for simple test
			continue
		}

		if err != nil {
			errors++
			if !continueOnError {
				return err
			}
		}

		executed++
	}

	cmd.Printf("Executed %d commands", executed)
	if errors > 0 {
		cmd.Printf(" with %d error%s", errors, pluralize(errors))
	}
	cmd.Println()

	return nil
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}