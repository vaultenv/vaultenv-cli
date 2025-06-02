package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	// Test basic fields
	if cfg.Version != CurrentVersion {
		t.Errorf("Version = %v, want %v", cfg.Version, CurrentVersion)
	}
	
	if cfg.Project.Name != "default" {
		t.Errorf("Project.Name = %v, want 'default'", cfg.Project.Name)
	}
	
	// Test vault defaults
	if cfg.Vault.Path != ".vaultenv" {
		t.Errorf("Vault.Path = %v, want '.vaultenv'", cfg.Vault.Path)
	}
	
	if cfg.Vault.Type != "file" {
		t.Errorf("Vault.Type = %v, want 'file'", cfg.Vault.Type)
	}
	
	if cfg.Vault.EncryptionAlgo != "aes-256-gcm" {
		t.Errorf("Vault.EncryptionAlgo = %v, want 'aes-256-gcm'", cfg.Vault.EncryptionAlgo)
	}
	
	// Test KDF defaults
	if cfg.Vault.KeyDerivation.Algorithm != "argon2id" {
		t.Errorf("KeyDerivation.Algorithm = %v, want 'argon2id'", cfg.Vault.KeyDerivation.Algorithm)
	}
	
	// Test security defaults
	if cfg.Security.PasswordPolicy.MinLength != 12 {
		t.Errorf("PasswordPolicy.MinLength = %v, want 12", cfg.Security.PasswordPolicy.MinLength)
	}
	
	// Test environments
	if len(cfg.Environments) != 3 {
		t.Errorf("Environments count = %v, want 3", len(cfg.Environments))
	}
	
	// Verify default environments exist
	for _, env := range []string{"development", "staging", "production"} {
		if _, exists := cfg.Environments[env]; !exists {
			t.Errorf("Missing default environment: %s", env)
		}
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid_default",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "missing_version",
			modify: func(c *Config) {
				c.Version = ""
			},
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "missing_project_name",
			modify: func(c *Config) {
				c.Project.Name = ""
			},
			wantErr: true,
			errMsg:  "project name is required",
		},
		{
			name: "invalid_vault_type",
			modify: func(c *Config) {
				c.Vault.Type = "invalid"
			},
			wantErr: true,
			errMsg:  "vault type must be",
		},
		{
			name: "invalid_encryption_algo",
			modify: func(c *Config) {
				c.Vault.EncryptionAlgo = "rot13"
			},
			wantErr: true,
			errMsg:  "unsupported encryption algorithm",
		},
		{
			name: "invalid_kdf_algorithm",
			modify: func(c *Config) {
				c.Vault.KeyDerivation.Algorithm = "md5"
			},
			wantErr: true,
			errMsg:  "unsupported KDF algorithm",
		},
		{
			name: "invalid_sync_conflict_mode",
			modify: func(c *Config) {
				c.Sync.Enabled = true
				c.Sync.ConflictMode = "invalid"
			},
			wantErr: true,
			errMsg:  "invalid sync conflict mode",
		},
		{
			name: "invalid_ui_theme",
			modify: func(c *Config) {
				c.UI.Theme = "neon"
			},
			wantErr: true,
			errMsg:  "invalid UI theme",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)
			
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create test config
	cfg := DefaultConfig()
	cfg.Project.Name = "test-project"
	cfg.Project.ID = "test-123"
	cfg.Project.Description = "Test project"
	
	// Save to file
	configPath := filepath.Join(tmpDir, "config.yaml")
	err = cfg.SaveToFile(configPath)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}
	
	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
	
	// Load from file
	loaded, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	
	// Verify loaded config
	if loaded.Project.Name != cfg.Project.Name {
		t.Errorf("Loaded Project.Name = %v, want %v", loaded.Project.Name, cfg.Project.Name)
	}
	
	if loaded.Project.ID != cfg.Project.ID {
		t.Errorf("Loaded Project.ID = %v, want %v", loaded.Project.ID, cfg.Project.ID)
	}
}

func TestConfig_SaveToWriter(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Project.Name = "writer-test"
	
	var buf bytes.Buffer
	err := cfg.SaveToWriter(&buf)
	if err != nil {
		t.Fatalf("SaveToWriter() error = %v", err)
	}
	
	// Verify YAML content
	content := buf.String()
	if !strings.Contains(content, "version:") {
		t.Error("YAML missing version field")
	}
	if !strings.Contains(content, "writer-test") {
		t.Error("YAML missing project name")
	}
}

func TestLoadFromReader(t *testing.T) {
	yamlContent := `
version: "1.0.0"
project:
  name: reader-test
  id: reader-123
vault:
  path: .vault
  type: sqlite
  encryption_algo: aes-256-gcm
  key_derivation:
    algorithm: argon2id
security:
  password_policy:
    min_length: 16
ui:
  theme: light
`
	
	reader := strings.NewReader(yamlContent)
	cfg, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("LoadFromReader() error = %v", err)
	}
	
	// Verify loaded values
	if cfg.Project.Name != "reader-test" {
		t.Errorf("Project.Name = %v, want 'reader-test'", cfg.Project.Name)
	}
	
	if cfg.Vault.Type != "sqlite" {
		t.Errorf("Vault.Type = %v, want 'sqlite'", cfg.Vault.Type)
	}
	
	if cfg.Security.PasswordPolicy.MinLength != 16 {
		t.Errorf("PasswordPolicy.MinLength = %v, want 16", cfg.Security.PasswordPolicy.MinLength)
	}
	
	if cfg.UI.Theme != "light" {
		t.Errorf("UI.Theme = %v, want 'light'", cfg.UI.Theme)
	}
}

func TestConfig_GetVaultPath(t *testing.T) {
	tests := []struct {
		name      string
		vaultPath string
		want      string
	}{
		{
			name:      "relative_path",
			vaultPath: ".vaultenv",
			want:      filepath.Join(".vaultenv", ".vaultenv"),
		},
		{
			name:      "absolute_path",
			vaultPath: "/absolute/path/vault",
			want:      "/absolute/path/vault",
		},
		{
			name:      "relative_with_dir",
			vaultPath: "data/vault",
			want:      filepath.Join("data", "vault"),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Vault.Path = tt.vaultPath
			
			got := cfg.GetVaultPath()
			if got != tt.want {
				t.Errorf("GetVaultPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_IsLocked(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Vault.AutoLock = true
	cfg.Vault.LockTimeout = 5 * time.Minute
	
	tests := []struct {
		name         string
		lastActivity time.Time
		want         bool
	}{
		{
			name:         "not_expired",
			lastActivity: time.Now().Add(-2 * time.Minute),
			want:         false,
		},
		{
			name:         "expired",
			lastActivity: time.Now().Add(-10 * time.Minute),
			want:         true,
		},
		{
			name:         "just_expired",
			lastActivity: time.Now().Add(-5*time.Minute - 1*time.Second),
			want:         true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.IsLocked(tt.lastActivity); got != tt.want {
				t.Errorf("IsLocked() = %v, want %v", got, tt.want)
			}
		})
	}
	
	// Test with AutoLock disabled
	cfg.Vault.AutoLock = false
	if cfg.IsLocked(time.Now().Add(-1 * time.Hour)) {
		t.Error("IsLocked() should return false when AutoLock is disabled")
	}
}

func TestVaultConfig_IsEncrypted(t *testing.T) {
	tests := []struct {
		name   string
		algo   string
		want   bool
	}{
		{"aes", "aes-256-gcm", true},
		{"chacha", "chacha20-poly1305", true},
		{"none", "none", false},
		{"empty", "", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vault := VaultConfig{
				EncryptionAlgo: tt.algo,
			}
			
			if got := vault.IsEncrypted(); got != tt.want {
				t.Errorf("IsEncrypted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_EnvironmentMethods(t *testing.T) {
	cfg := DefaultConfig()
	
	// Test GetEnvironmentNames
	names := cfg.GetEnvironmentNames()
	if len(names) != 3 {
		t.Errorf("GetEnvironmentNames() returned %d names, want 3", len(names))
	}
	
	// Test HasEnvironment
	if !cfg.HasEnvironment("development") {
		t.Error("HasEnvironment('development') = false, want true")
	}
	if cfg.HasEnvironment("nonexistent") {
		t.Error("HasEnvironment('nonexistent') = true, want false")
	}
	
	// Test GetEnvironmentConfig
	devConfig, exists := cfg.GetEnvironmentConfig("development")
	if !exists {
		t.Error("GetEnvironmentConfig('development') not found")
	}
	if devConfig.Description != "Development environment" {
		t.Errorf("Development description = %v", devConfig.Description)
	}
	
	// Test SetEnvironmentConfig
	newEnv := EnvironmentConfig{
		Description:       "Test environment",
		PasswordProtected: true,
	}
	cfg.SetEnvironmentConfig("test", newEnv)
	
	if !cfg.HasEnvironment("test") {
		t.Error("SetEnvironmentConfig() did not add environment")
	}
	
	// Test with nil map
	cfg2 := &Config{}
	cfg2.SetEnvironmentConfig("new", newEnv)
	if cfg2.Environments == nil {
		t.Error("SetEnvironmentConfig() did not initialize map")
	}
}

func TestConfig_GetPasswordPolicy(t *testing.T) {
	cfg := DefaultConfig()
	
	// Test environment-specific policy
	devPolicy := cfg.GetPasswordPolicy("development")
	if devPolicy.MinLength != 8 {
		t.Errorf("Development MinLength = %v, want 8", devPolicy.MinLength)
	}
	
	prodPolicy := cfg.GetPasswordPolicy("production")
	if prodPolicy.MinLength != 16 {
		t.Errorf("Production MinLength = %v, want 16", prodPolicy.MinLength)
	}
	
	// Test fallback to global policy for unknown environment
	unknownPolicy := cfg.GetPasswordPolicy("unknown")
	if unknownPolicy.MinLength != cfg.Security.PasswordPolicy.MinLength {
		t.Error("Unknown environment should use global policy")
	}
	
	// Test environment without policy falls back to global
	cfg.Environments["nopolicy"] = EnvironmentConfig{
		Description: "No policy environment",
	}
	nopolicyPolicy := cfg.GetPasswordPolicy("nopolicy")
	if nopolicyPolicy.MinLength != cfg.Security.PasswordPolicy.MinLength {
		t.Error("Environment without policy should use global policy")
	}
}

func TestConfig_PerEnvironmentPasswords(t *testing.T) {
	cfg := DefaultConfig()
	
	// Default should be false
	if cfg.IsPerEnvironmentPasswordsEnabled() {
		t.Error("IsPerEnvironmentPasswordsEnabled() = true, want false by default")
	}
	
	// Enable and test
	cfg.EnablePerEnvironmentPasswords()
	
	if !cfg.IsPerEnvironmentPasswordsEnabled() {
		t.Error("IsPerEnvironmentPasswordsEnabled() = false after enabling")
	}
	
	if cfg.Version != "2.0.0" {
		t.Errorf("Version = %v, want '2.0.0' after enabling per-env passwords", cfg.Version)
	}
}

func TestConfig_Merge(t *testing.T) {
	base := DefaultConfig()
	base.Project.Name = "base"
	base.Project.Description = "Base description"
	
	override := &Config{
		Version: "2.0.0",
		Project: ProjectConfig{
			Name: "override",
			Team: "new-team",
		},
	}
	
	merged := base.Merge(override)
	
	// Override should take precedence
	if merged.Version != "2.0.0" {
		t.Errorf("Merged Version = %v, want '2.0.0'", merged.Version)
	}
	
	if merged.Project.Name != "override" {
		t.Errorf("Merged Project.Name = %v, want 'override'", merged.Project.Name)
	}
	
	// Non-overridden fields should remain
	if merged.Project.Description != "Base description" {
		t.Errorf("Merged Project.Description = %v, want 'Base description'", merged.Project.Description)
	}
	
	// Test merge with nil
	nilMerged := base.Merge(nil)
	if !reflect.DeepEqual(nilMerged, base) {
		t.Error("Merge(nil) should return original config")
	}
}

func TestPassPolicy(t *testing.T) {
	policy := PassPolicy{
		MinLength:      12,
		RequireUpper:   true,
		RequireLower:   true,
		RequireNumbers: true,
		RequireSpecial: true,
		PreventCommon:  true,
		ExpiryDays:     90,
	}
	
	// Just verify the struct fields work correctly
	if policy.MinLength != 12 {
		t.Errorf("MinLength = %v, want 12", policy.MinLength)
	}
	
	if !policy.RequireUpper {
		t.Error("RequireUpper should be true")
	}
	
	if policy.ExpiryDays != 90 {
		t.Errorf("ExpiryDays = %v, want 90", policy.ExpiryDays)
	}
}

func TestGitConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	// Test defaults
	if !cfg.Git.Enabled {
		t.Error("Git should be enabled by default")
	}
	
	if cfg.Git.AutoCommit {
		t.Error("AutoCommit should be false by default")
	}
	
	if cfg.Git.EncryptionMode != "random" {
		t.Errorf("Git.EncryptionMode = %v, want 'random'", cfg.Git.EncryptionMode)
	}
	
	if cfg.Git.ConflictStrategy != "prompt" {
		t.Errorf("Git.ConflictStrategy = %v, want 'prompt'", cfg.Git.ConflictStrategy)
	}
}

func TestUIConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	// Test defaults
	if cfg.UI.Theme != "dark" {
		t.Errorf("UI.Theme = %v, want 'dark'", cfg.UI.Theme)
	}
	
	if !cfg.UI.Color {
		t.Error("UI.Color should be true by default")
	}
	
	if cfg.UI.DateFormat != "2006-01-02" {
		t.Errorf("UI.DateFormat = %v, want '2006-01-02'", cfg.UI.DateFormat)
	}
}

func BenchmarkConfig_Validate(b *testing.B) {
	cfg := DefaultConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.Validate()
	}
}

func BenchmarkConfig_SaveToWriter(b *testing.B) {
	cfg := DefaultConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = cfg.SaveToWriter(&buf)
	}
}

func BenchmarkLoadFromReader(b *testing.B) {
	cfg := DefaultConfig()
	var buf bytes.Buffer
	cfg.SaveToWriter(&buf)
	content := buf.String()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, _ = LoadFromReader(reader)
	}
}