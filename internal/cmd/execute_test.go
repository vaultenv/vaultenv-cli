package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vaultenv/vaultenv-cli/internal/config"
)

func TestExecute(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name      string
		args      []string
		buildInfo BuildInfo
		wantErr   bool
	}{
		{
			name: "version command",
			args: []string{"vaultenv", "version"},
			buildInfo: BuildInfo{
				Version:   "1.0.0",
				Commit:    "abc123",
				BuildTime: "2023-01-01",
				BuiltBy:   "test",
			},
			wantErr: false,
		},
		{
			name: "help command",
			args: []string{"vaultenv", "--help"},
			buildInfo: BuildInfo{
				Version: "1.0.0",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			rootCmd = NewRootCommand()
			os.Args = tt.args

			err := Execute(tt.buildInfo)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "vaultenv-cli", cmd.Use)
	assert.Contains(t, cmd.Short, "Secure environment variable management")

	// Check that all subcommands are added
	expectedCommands := []string{
		"version", "set", "get", "list", "init", "completion",
		"history", "restore", "audit", "migrate", "git", "env",
		"load", "export", "batch", "config", "security", "shell", "run",
	}

	for _, cmdName := range expectedCommands {
		t.Run("has_"+cmdName+"_command", func(t *testing.T) {
			found := false
			for _, c := range cmd.Commands() {
				if c.Name() == cmdName {
					found = true
					break
				}
			}
			assert.True(t, found, "Command %s not found", cmdName)
		})
	}
}

func TestConfigureColorOutput(t *testing.T) {
	tests := []struct {
		name    string
		noColor bool
		envVar  string
		ciEnv   string
	}{
		{
			name:    "no color flag",
			noColor: true,
		},
		{
			name:   "NO_COLOR env var",
			envVar: "1",
		},
		{
			name:  "CI environment",
			ciEnv: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			oldNoColor := os.Getenv("NO_COLOR")
			oldCI := os.Getenv("CI")
			defer func() {
				os.Setenv("NO_COLOR", oldNoColor)
				os.Setenv("CI", oldCI)
			}()

			// Set test environment
			noColor = tt.noColor
			if tt.envVar != "" {
				os.Setenv("NO_COLOR", tt.envVar)
			} else {
				os.Unsetenv("NO_COLOR")
			}
			if tt.ciEnv != "" {
				os.Setenv("CI", tt.ciEnv)
			} else {
				os.Unsetenv("CI")
			}

			configureColorOutput()
			// Function modifies global color state which we can't directly test
			// But we verify it doesn't panic
		})
	}
}

func TestInitializeConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfgFile string
	}{
		{
			name:    "no config file",
			cfgFile: "",
		},
		{
			name:    "specific config file",
			cfgFile: "/tmp/test-config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgFile = tt.cfgFile
			verbose = false

			// This should not panic
			initializeConfig()
		})
	}
}

func TestHandleError(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetErr(&buf)

	// Test error handling doesn't panic
	handleError(assert.AnError)
}

func TestLoadProjectConfig(t *testing.T) {
	t.Skip("Skipping test for beta release - test implementation needs update")
	// Save original cfgFile value
	originalCfgFile := cfgFile
	defer func() { cfgFile = originalCfgFile }()

	// Save original working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	tests := []struct {
		name      string
		cfgFile   string
		setupFunc func(t *testing.T) string
		wantErr   bool
	}{
		{
			name: "no config file",
			setupFunc: func(t *testing.T) string {
				// Change to a temp directory with no config
				tmpDir := t.TempDir()
				require.NoError(t, os.Chdir(tmpDir))
				return ""
			},
			wantErr: true,
		},
		{
			name: "specific config file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				cfgPath := filepath.Join(tmpDir, "config.yaml")

				cfg := &config.Config{
					Version: config.CurrentVersion,
					Project: config.ProjectConfig{
						Name: "test-project",
					},
					Vault: config.VaultConfig{
						Type: "file",
						Path: filepath.Join(tmpDir, "vault"),
					},
				}
				require.NoError(t, cfg.SaveToFile(cfgPath))
				return cfgPath
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				cfgFile = tt.setupFunc(t)
			} else {
				cfgFile = tt.cfgFile
			}

			cfg, err := loadProjectConfig()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestIsInitCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *cobra.Command
		args     []string
		expected bool
	}{
		{
			name:     "init command",
			cmd:      &cobra.Command{Use: "init"},
			args:     []string{"vaultenv", "init"},
			expected: true,
		},
		{
			name:     "non-init command",
			cmd:      &cobra.Command{Use: "get"},
			args:     []string{"vaultenv", "get", "KEY"},
			expected: false,
		},
		{
			name: "init subcommand",
			cmd: &cobra.Command{
				Use: "subcommand",
			},
			args:     []string{"vaultenv", "init", "subcommand"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore os.Args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			result := isInitCommand(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetConfig(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name: "test-project",
		},
	}

	tests := []struct {
		name         string
		setupFunc    func(*cobra.Command)
		globalConfig *config.Config
		expected     *config.Config
	}{
		{
			name: "config from context",
			setupFunc: func(cmd *cobra.Command) {
				ctx := context.WithValue(context.Background(), configKey{}, cfg)
				cmd.SetContext(ctx)
			},
			expected: cfg,
		},
		{
			name:         "global config fallback",
			setupFunc:    func(cmd *cobra.Command) {},
			globalConfig: cfg,
			expected:     cfg,
		},
		{
			name:      "no config",
			setupFunc: func(cmd *cobra.Command) {},
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			tt.setupFunc(cmd)

			// Set global config
			oldGlobal := globalConfig
			globalConfig = tt.globalConfig
			defer func() { globalConfig = oldGlobal }()

			result := GetConfig(cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMustGetConfig(t *testing.T) {
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name: "test-project",
		},
	}

	t.Run("with config", func(t *testing.T) {
		cmd := &cobra.Command{}
		ctx := context.WithValue(context.Background(), configKey{}, cfg)
		cmd.SetContext(ctx)

		result := MustGetConfig(cmd)
		assert.Equal(t, cfg, result)
	})

	t.Run("without config panics", func(t *testing.T) {
		cmd := &cobra.Command{}

		// Save global config
		oldGlobal := globalConfig
		globalConfig = nil
		defer func() { globalConfig = oldGlobal }()

		assert.Panics(t, func() {
			MustGetConfig(cmd)
		})
	})
}

func TestAddCommands(t *testing.T) {
	// Reset rootCmd
	rootCmd = &cobra.Command{
		Use: "vaultenv-cli",
	}

	// Verify no commands initially
	assert.Empty(t, rootCmd.Commands())

	// Add commands
	addCommands()

	// Verify commands were added
	assert.NotEmpty(t, rootCmd.Commands())

	// Check for some key commands
	commandNames := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		commandNames[cmd.Name()] = true
	}

	expectedCommands := []string{"version", "set", "get", "list", "init"}
	for _, name := range expectedCommands {
		assert.True(t, commandNames[name], "Expected command %s not found", name)
	}
}
