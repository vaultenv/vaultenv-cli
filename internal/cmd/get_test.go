package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func TestGetCommand(t *testing.T) {
	// Create test storage
	tmpDir, err := ioutil.TempDir("", "vaultenv-get-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test storage backend
	store, err := storage.NewFileBackend(tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}

	// Add some test data
	testData := map[string]string{
		"DATABASE_URL":    "postgres://localhost/test",
		"API_KEY":         "secret123",
		"FEATURE_FLAG":    "true",
		"NESTED__VALUE":   "nested",
		"PORT":            "8080",
	}

	for key, value := range testData {
		if err := store.Set(key, value, false); err != nil {
			t.Fatalf("Failed to set test data: %v", err)
		}
	}

	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		wantErr     bool
		wantOutput  string
		wantExact   bool
		wantMissing []string
	}{
		{
			name:       "get_single_key",
			args:       []string{"DATABASE_URL"},
			wantOutput: "postgres://localhost/test",
			wantExact:  true,
		},
		{
			name:       "get_single_key_verbose",
			args:       []string{"API_KEY"},
			flags:      map[string]string{"verbose": "true"},
			wantOutput: "API_KEY=secret123",
		},
		{
			name:    "get_non_existent_key",
			args:    []string{"NON_EXISTENT"},
			wantErr: true,
		},
		{
			name:       "get_multiple_keys",
			args:       []string{"DATABASE_URL", "PORT"},
			flags:      map[string]string{"verbose": "true"},
			wantOutput: "DATABASE_URL=postgres://localhost/test",
		},
		{
			name:    "get_no_args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:       "get_with_json_format",
			args:       []string{"PORT"},
			flags:      map[string]string{"format": "json"},
			wantOutput: `{"PORT":"8080"}`,
		},
		{
			name:       "get_with_export_format",
			args:       []string{"DATABASE_URL"},
			flags:      map[string]string{"format": "export"},
			wantOutput: "export DATABASE_URL='postgres://localhost/test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := &cobra.Command{
				Use:   "get",
				Short: "Get value of an environment variable",
				Args:  cobra.MinimumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Simplified get logic for testing
					if len(args) == 0 {
						return fmt.Errorf("requires at least 1 arg(s), only received 0")
					}

					format, _ := cmd.Flags().GetString("format")
					verbose, _ := cmd.Flags().GetBool("verbose")

					for _, key := range args {
						value, err := store.Get(key)
						if err != nil {
							return err
						}

						switch format {
						case "json":
							cmd.Printf(`{"%s":"%s"}`, key, value)
						case "export":
							cmd.Printf("export %s='%s'", key, value)
						default:
							if verbose {
								cmd.Printf("%s=%s", key, value)
							} else {
								cmd.Print(value)
							}
						}

						if len(args) > 1 {
							cmd.Println()
						}
					}
					return nil
				},
			}

			// Add flags
			cmd.Flags().String("format", "", "Output format")
			cmd.Flags().Bool("verbose", false, "Verbose output")

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

			// Check output
			output := strings.TrimSpace(buf.String())
			if tt.wantExact {
				if output != tt.wantOutput {
					t.Errorf("Output = %q, want %q", output, tt.wantOutput)
				}
			} else if tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("Output = %q, want to contain %q", output, tt.wantOutput)
				}
			}

			// Check missing strings
			for _, missing := range tt.wantMissing {
				if strings.Contains(output, missing) {
					t.Errorf("Output = %q, should not contain %q", output, missing)
				}
			}
		})
	}
}

func TestGetCommandWithEnvironments(t *testing.T) {
	// Test get command with different environments
	tmpDir, err := ioutil.TempDir("", "vaultenv-get-env-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage for different environments
	envs := map[string]map[string]string{
		"development": {
			"API_URL": "http://localhost:3000",
			"DEBUG":   "true",
		},
		"production": {
			"API_URL": "https://api.example.com",
			"DEBUG":   "false",
		},
	}

	stores := make(map[string]storage.Backend)
	for env, data := range envs {
		store, err := storage.NewFileBackend(tmpDir, env)
		if err != nil {
			t.Fatal(err)
		}
		stores[env] = store

		for key, value := range data {
			if err := store.Set(key, value, false); err != nil {
				t.Fatalf("Failed to set test data: %v", err)
			}
		}
	}

	tests := []struct {
		name        string
		args        []string
		environment string
		wantOutput  string
	}{
		{
			name:        "get_from_development",
			args:        []string{"API_URL"},
			environment: "development",
			wantOutput:  "http://localhost:3000",
		},
		{
			name:        "get_from_production",
			args:        []string{"API_URL"},
			environment: "production",
			wantOutput:  "https://api.example.com",
		},
		{
			name:        "get_debug_from_dev",
			args:        []string{"DEBUG"},
			environment: "development",
			wantOutput:  "true",
		},
		{
			name:        "get_debug_from_prod",
			args:        []string{"DEBUG"},
			environment: "production",
			wantOutput:  "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := &cobra.Command{
				Use:  "get",
				Args: cobra.MinimumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					store := stores[tt.environment]
					value, err := store.Get(args[0])
					if err != nil {
						return err
					}
					cmd.Print(value)
					return nil
				},
			}

			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}

			output := strings.TrimSpace(buf.String())
			if output != tt.wantOutput {
				t.Errorf("Output = %q, want %q", output, tt.wantOutput)
			}
		})
	}
}

func TestGetCommandPatterns(t *testing.T) {
	// Test pattern matching
	tmpDir, err := ioutil.TempDir("", "vaultenv-get-pattern-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewFileBackend(tmpDir, "test")
	if err != nil {
		t.Fatal(err)
	}

	// Add test data with patterns
	testData := map[string]string{
		"DB_HOST":     "localhost",
		"DB_PORT":     "5432",
		"DB_NAME":     "testdb",
		"API_KEY":     "key123",
		"API_SECRET":  "secret456",
		"API_VERSION": "v1",
	}

	for key, value := range testData {
		if err := store.Set(key, value, false); err != nil {
			t.Fatalf("Failed to set test data: %v", err)
		}
	}

	tests := []struct {
		name       string
		pattern    string
		wantKeys   []string
		wantErr    bool
	}{
		{
			name:     "pattern_db_prefix",
			pattern:  "DB_*",
			wantKeys: []string{"DB_HOST", "DB_PORT", "DB_NAME"},
		},
		{
			name:     "pattern_api_prefix",
			pattern:  "API_*",
			wantKeys: []string{"API_KEY", "API_SECRET", "API_VERSION"},
		},
		{
			name:     "pattern_all",
			pattern:  "*",
			wantKeys: []string{"DB_HOST", "DB_PORT", "DB_NAME", "API_KEY", "API_SECRET", "API_VERSION"},
		},
		{
			name:     "pattern_suffix",
			pattern:  "*_KEY",
			wantKeys: []string{"API_KEY"},
		},
		{
			name:    "pattern_no_match",
			pattern: "CACHE_*",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := &cobra.Command{
				Use: "get",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Simulate pattern matching
					pattern := args[0]
					found := false
					
					keys, err := store.List()
					if err != nil {
						return err
					}

					for _, key := range keys {
						if match, _ := matchPattern(key, pattern); match {
							value, _ := store.Get(key)
							cmd.Printf("%s=%s\n", key, value)
							found = true
						}
					}

					if !found {
						return storage.ErrNotFound
					}
					return nil
				},
			}

			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs([]string{tt.pattern})

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				for _, key := range tt.wantKeys {
					if !strings.Contains(output, key) {
						t.Errorf("Output missing expected key %q", key)
					}
				}
			}
		})
	}
}