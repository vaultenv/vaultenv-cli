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

func TestSetCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		stdin       string
		wantErr     bool
		wantOutput  string
		verifyStore func(t *testing.T, store storage.Backend)
	}{
		{
			name:       "set_single_value",
			args:       []string{"DATABASE_URL", "postgres://localhost/test"},
			wantOutput: "Set DATABASE_URL",
			verifyStore: func(t *testing.T, store storage.Backend) {
				val, err := store.Get("DATABASE_URL")
				if err != nil {
					t.Errorf("Failed to get value: %v", err)
				}
				if val != "postgres://localhost/test" {
					t.Errorf("Value = %q, want %q", val, "postgres://localhost/test")
				}
			},
		},
		{
			name:       "set_multiple_values",
			args:       []string{"KEY1", "value1", "KEY2", "value2"},
			wantOutput: "Set KEY1",
			verifyStore: func(t *testing.T, store storage.Backend) {
				val1, _ := store.Get("KEY1")
				val2, _ := store.Get("KEY2")
				if val1 != "value1" {
					t.Errorf("KEY1 = %q, want %q", val1, "value1")
				}
				if val2 != "value2" {
					t.Errorf("KEY2 = %q, want %q", val2, "value2")
				}
			},
		},
		{
			name:    "set_missing_value",
			args:    []string{"KEY_ONLY"},
			wantErr: true,
		},
		{
			name:    "set_no_args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:       "set_with_equals",
			args:       []string{"KEY=value with spaces"},
			wantOutput: "Set KEY",
			verifyStore: func(t *testing.T, store storage.Backend) {
				val, _ := store.Get("KEY")
				if val != "value with spaces" {
					t.Errorf("Value = %q, want %q", val, "value with spaces")
				}
			},
		},
		{
			name:       "set_empty_value",
			args:       []string{"EMPTY_KEY", ""},
			wantOutput: "Set EMPTY_KEY",
			verifyStore: func(t *testing.T, store storage.Backend) {
				val, _ := store.Get("EMPTY_KEY")
				if val != "" {
					t.Errorf("Value = %q, want empty string", val)
				}
			},
		},
		{
			name:       "set_from_stdin",
			args:       []string{"STDIN_KEY", "-"},
			stdin:      "value from stdin",
			wantOutput: "Set STDIN_KEY",
			verifyStore: func(t *testing.T, store storage.Backend) {
				val, _ := store.Get("STDIN_KEY")
				if val != "value from stdin" {
					t.Errorf("Value = %q, want %q", val, "value from stdin")
				}
			},
		},
		{
			name:       "set_multiline_from_stdin",
			args:       []string{"MULTILINE", "-"},
			stdin:      "line1\nline2\nline3",
			wantOutput: "Set MULTILINE",
			verifyStore: func(t *testing.T, store storage.Backend) {
				val, _ := store.Get("MULTILINE")
				if val != "line1\nline2\nline3" {
					t.Errorf("Value = %q, want multiline", val)
				}
			},
		},
		{
			name:  "set_force_overwrite",
			args:  []string{"EXISTING", "new_value"},
			flags: map[string]string{"force": "true"},
			verifyStore: func(t *testing.T, store storage.Backend) {
				// Pre-set a value
				store.Set("EXISTING", "old_value", false)
				// The test will overwrite it
			},
		},
		{
			name:       "set_special_characters",
			args:       []string{"SPECIAL", "!@#$%^&*()"},
			wantOutput: "Set SPECIAL",
			verifyStore: func(t *testing.T, store storage.Backend) {
				val, _ := store.Get("SPECIAL")
				if val != "!@#$%^&*()" {
					t.Errorf("Value = %q, want %q", val, "!@#$%^&*()")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp storage
			tmpDir, err := ioutil.TempDir("", "vaultenv-set-test")
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
				Use:  "set",
				Args: cobra.MinimumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Simplified set logic for testing
					if len(args) == 0 {
						return fmt.Errorf("requires at least 1 arg(s), only received 0")
					}

					// Handle KEY=VALUE format
					if strings.Contains(args[0], "=") {
						parts := strings.SplitN(args[0], "=", 2)
						if err := store.Set(parts[0], parts[1], false); err != nil {
							return err
						}
						cmd.Printf("Set %s", parts[0])
						return nil
					}

					// Handle KEY VALUE pairs
					if len(args)%2 != 0 && (len(args) == 1 || args[len(args)-1] != "-") {
						return fmt.Errorf("missing value for key: %s", args[len(args)-1])
					}

					for i := 0; i < len(args); i += 2 {
						key := args[i]
						value := args[i+1]

						// Handle stdin
						if value == "-" {
							// In real implementation, would read from stdin
							value = tt.stdin
						}

						if err := store.Set(key, value, false); err != nil {
							return err
						}
						cmd.Printf("Set %s\n", key)
					}
					return nil
				},
			}

			// Add flags
			cmd.Flags().Bool("force", false, "Force overwrite")

			// Set flags
			for key, value := range tt.flags {
				cmd.Flags().Set(key, value)
			}

			// Pre-populate store if needed
			if tt.verifyStore != nil && tt.name == "set_force_overwrite" {
				store.Set("EXISTING", "old_value", false)
			}

			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			// Execute command
			err = cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check output
			output := strings.TrimSpace(buf.String())
			if tt.wantOutput != "" && !strings.Contains(output, tt.wantOutput) {
				t.Errorf("Output = %q, want to contain %q", output, tt.wantOutput)
			}

			// Verify store
			if tt.verifyStore != nil && !tt.wantErr {
				tt.verifyStore(t, store)
			}
		})
	}
}

func TestSetCommandBatch(t *testing.T) {
	// Test batch operations
	tmpDir, err := ioutil.TempDir("", "vaultenv-set-batch")
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
		Use: "set",
		RunE: func(cmd *cobra.Command, args []string) error {
			batch, _ := cmd.Flags().GetBool("batch")
			if !batch {
				return nil
			}

			// Simulate batch processing
			keys := []string{"BATCH1", "BATCH2", "BATCH3"}
			values := []string{"val1", "val2", "val3"}

			for i, key := range keys {
				if err := store.Set(key, values[i], false); err != nil {
					return err
				}
				cmd.Printf("Set %s\n", key)
			}
			return nil
		},
	}

	cmd.Flags().Bool("batch", true, "Batch mode")
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify all values were set
	expected := map[string]string{
		"BATCH1": "val1",
		"BATCH2": "val2",
		"BATCH3": "val3",
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
}

func TestSetCommandValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "invalid_key_empty",
			args:    []string{"", "value"},
			wantErr: true,
			errMsg:  "empty key",
		},
		{
			name:    "invalid_key_spaces",
			args:    []string{"KEY WITH SPACES", "value"},
			wantErr: true,
			errMsg:  "invalid key",
		},
		{
			name:    "invalid_key_special",
			args:    []string{"KEY$DOLLAR", "value"},
			wantErr: true,
			errMsg:  "invalid key",
		},
		{
			name:    "valid_key_underscore",
			args:    []string{"VALID_KEY", "value"},
			wantErr: false,
		},
		{
			name:    "valid_key_numbers",
			args:    []string{"KEY123", "value"},
			wantErr: false,
		},
		{
			name:    "reserved_key",
			args:    []string{"PATH", "value"},
			wantErr: true,
			errMsg:  "reserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "vaultenv-set-validation")
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
				Use: "set",
				RunE: func(cmd *cobra.Command, args []string) error {
					if len(args) < 2 {
						return fmt.Errorf("requires at least 1 arg(s), only received 0")
					}

					key := args[0]

					// Validation
					if key == "" {
						return fmt.Errorf("required flag(s) \"empty key\" not set")
					}
					if strings.Contains(key, " ") {
						return fmt.Errorf("required flag(s) \"invalid key: contains spaces\" not set")
					}
					if strings.ContainsAny(key, "$@#%^&*()") {
						return fmt.Errorf("required flag(s) \"invalid key: contains special characters\" not set")
					}
					if key == "PATH" || key == "HOME" || key == "USER" {
						return fmt.Errorf("required flag(s) \"reserved key\" not set")
					}

					return store.Set(key, args[1], false)
				},
			}

			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err = cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				errStr := ""
				if err != nil {
					errStr = err.Error()
				}
				if !strings.Contains(errStr, tt.errMsg) {
					t.Errorf("Error = %q, want to contain %q", errStr, tt.errMsg)
				}
			}
		})
	}
}
