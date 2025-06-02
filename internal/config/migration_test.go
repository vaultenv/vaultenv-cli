package config

import (
	"fmt"
	"reflect"
	"testing"
)

func TestMigrateConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   *Config
		wantErr bool
	}{
		{
			name:    "nil_config",
			input:   nil,
			wantErr: true,
		},
		{
			name: "v1_config",
			input: &Config{
				Version: "1.0.0",
				Project: ProjectConfig{
					Name: "test",
				},
				Security: SecurityConfig{
					PasswordPolicy: PassPolicy{
						MinLength: 10,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "v2_config",
			input: &Config{
				Version: "2.0.0",
				Project: ProjectConfig{
					Name: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "current_version",
			input: &Config{
				Version: CurrentVersion,
				Project: ProjectConfig{
					Name: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "unknown_version",
			input: &Config{
				Version: "99.0.0",
				Project: ProjectConfig{
					Name: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "empty_version",
			input: &Config{
				Version: "",
				Project: ProjectConfig{
					Name: "test",
				},
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MigrateConfig(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("MigrateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err == nil && got != nil {
				// Verify migration sets current version
				if got.Version != CurrentVersion {
					t.Errorf("MigrateConfig() version = %v, want %v", got.Version, CurrentVersion)
				}
			}
		})
	}
}

func TestMigrateV1ToV2(t *testing.T) {
	tests := []struct {
		name  string
		input *Config
		check func(*Config) error
	}{
		{
			name: "empty_environments",
			input: &Config{
				Version: "1.0.0",
			},
			check: func(cfg *Config) error {
				if len(cfg.Environments) < 2 {
					return fmt.Errorf("expected at least 2 default environments, got %d", len(cfg.Environments))
				}
				
				if _, exists := cfg.Environments["development"]; !exists {
					return fmt.Errorf("missing development environment")
				}
				
				if _, exists := cfg.Environments["production"]; !exists {
					return fmt.Errorf("missing production environment")
				}
				
				return nil
			},
		},
		{
			name: "existing_environments",
			input: &Config{
				Version: "1.0.0",
				Environments: map[string]EnvironmentConfig{
					"test": {
						Description: "Test env",
					},
				},
				Security: SecurityConfig{
					PasswordPolicy: PassPolicy{
						MinLength: 15,
					},
				},
			},
			check: func(cfg *Config) error {
				testEnv := cfg.Environments["test"]
				
				if !testEnv.PasswordProtected {
					return fmt.Errorf("test environment should have password protection enabled")
				}
				
				if testEnv.PasswordPolicy.MinLength != 15 {
					return fmt.Errorf("test environment should use global policy, got MinLength=%d", testEnv.PasswordPolicy.MinLength)
				}
				
				return nil
			},
		},
		{
			name: "standard_environments",
			input: &Config{
				Version: "1.0.0",
				Environments: map[string]EnvironmentConfig{
					"development": {},
					"staging":     {},
					"production":  {},
				},
			},
			check: func(cfg *Config) error {
				// Check development policy
				devPolicy := cfg.Environments["development"].PasswordPolicy
				if devPolicy.MinLength != 8 {
					return fmt.Errorf("development MinLength = %d, want 8", devPolicy.MinLength)
				}
				if devPolicy.RequireUpper {
					return fmt.Errorf("development should not require uppercase")
				}
				
				// Check staging policy
				stagingPolicy := cfg.Environments["staging"].PasswordPolicy
				if stagingPolicy.MinLength != 12 {
					return fmt.Errorf("staging MinLength = %d, want 12", stagingPolicy.MinLength)
				}
				if !stagingPolicy.RequireNumbers {
					return fmt.Errorf("staging should require numbers")
				}
				
				// Check production policy
				prodPolicy := cfg.Environments["production"].PasswordPolicy
				if prodPolicy.MinLength != 16 {
					return fmt.Errorf("production MinLength = %d, want 16", prodPolicy.MinLength)
				}
				if !prodPolicy.RequireSpecial {
					return fmt.Errorf("production should require special characters")
				}
				if prodPolicy.ExpiryDays != 90 {
					return fmt.Errorf("production ExpiryDays = %d, want 90", prodPolicy.ExpiryDays)
				}
				
				return nil
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrateV1ToV2(tt.input)
			
			if result.Version != "2.0.0" {
				t.Errorf("Version = %v, want 2.0.0", result.Version)
			}
			
			if err := tt.check(result); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestMigrateV2ToV3(t *testing.T) {
	tests := []struct {
		name  string
		input *Config
		check func(*Config) error
	}{
		{
			name: "empty_import_export",
			input: &Config{
				Version: "2.0.0",
			},
			check: func(cfg *Config) error {
				// Check Import defaults
				if !cfg.Import.DefaultParser.TrimSpace {
					return fmt.Errorf("Import.DefaultParser.TrimSpace should be true")
				}
				if cfg.Import.DefaultParser.ExpandVars {
					return fmt.Errorf("Import.DefaultParser.ExpandVars should be false")
				}
				if !cfg.Import.AutoBackup {
					return fmt.Errorf("Import.AutoBackup should be true")
				}
				
				// Check Export defaults
				if cfg.Export.DefaultFormat != "dotenv" {
					return fmt.Errorf("Export.DefaultFormat = %v, want 'dotenv'", cfg.Export.DefaultFormat)
				}
				if cfg.Export.Templates == nil {
					return fmt.Errorf("Export.Templates should be initialized")
				}
				
				return nil
			},
		},
		{
			name: "migrate_deterministic_mode",
			input: &Config{
				Version: "2.0.0",
				Git: GitConfig{
					DeterministicMode: true,
				},
			},
			check: func(cfg *Config) error {
				if cfg.Git.EncryptionMode != "deterministic" {
					return fmt.Errorf("Git.EncryptionMode = %v, want 'deterministic'", cfg.Git.EncryptionMode)
				}
				return nil
			},
		},
		{
			name: "default_encryption_mode",
			input: &Config{
				Version: "2.0.0",
				Git: GitConfig{
					DeterministicMode: false,
				},
			},
			check: func(cfg *Config) error {
				if cfg.Git.EncryptionMode != "random" {
					return fmt.Errorf("Git.EncryptionMode = %v, want 'random'", cfg.Git.EncryptionMode)
				}
				return nil
			},
		},
		{
			name: "git_conflict_strategy",
			input: &Config{
				Version: "2.0.0",
				Git:     GitConfig{},
			},
			check: func(cfg *Config) error {
				if cfg.Git.ConflictStrategy != "prompt" {
					return fmt.Errorf("Git.ConflictStrategy = %v, want 'prompt'", cfg.Git.ConflictStrategy)
				}
				return nil
			},
		},
		{
			name: "preserve_existing_values",
			input: &Config{
				Version: "2.0.0",
				Import: ImportConfig{
					DefaultParser: ParserConfig{
						TrimSpace: false,
						ExpandVars: true,
					},
					AutoBackup: false,
				},
				Export: ExportConfig{
					DefaultFormat: "json",
				},
				Git: GitConfig{
					ConflictStrategy: "ours",
					EncryptionMode: "custom",
				},
			},
			check: func(cfg *Config) error {
				// Should preserve existing values
				if cfg.Import.DefaultParser.ExpandVars != true {
					return fmt.Errorf("Import.DefaultParser.ExpandVars should be preserved")
				}
				if cfg.Export.DefaultFormat != "json" {
					return fmt.Errorf("Export.DefaultFormat should be preserved")
				}
				if cfg.Git.ConflictStrategy != "ours" {
					return fmt.Errorf("Git.ConflictStrategy should be preserved")
				}
				if cfg.Git.EncryptionMode != "custom" {
					return fmt.Errorf("Git.EncryptionMode should be preserved")
				}
				return nil
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := migrateV2ToV3(tt.input)
			
			if result.Version != "3.0.0" {
				t.Errorf("Version = %v, want 3.0.0", result.Version)
			}
			
			if err := tt.check(result); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestIsLegacyConfig(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"empty_version", "", true},
		{"v1.0", "1.0", true},
		{"v1.0.0", "1.0.0", true},
		{"v1.5.0", "1.5.0", true},
		{"v2.0.0", "2.0.0", false},
		{"v3.0.0", "3.0.0", false},
		{"future", "4.0.0", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Version: tt.version}
			if got := IsLegacyConfig(cfg); got != tt.want {
				t.Errorf("IsLegacyConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedsMigration(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"empty_version", "", true},
		{"v1.0.0", "1.0.0", true},
		{"v2.0.0", "2.0.0", true},
		{"current", CurrentVersion, false},
		{"future", "99.0.0", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Version: tt.version}
			if got := NeedsMigration(cfg); got != tt.want {
				t.Errorf("NeedsMigration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMigrationChain(t *testing.T) {
	// Test full migration chain from v1 to current
	v1Config := &Config{
		Version: "1.0.0",
		Project: ProjectConfig{
			Name: "migration-test",
			ID:   "test-123",
		},
		Vault: VaultConfig{
			Path: ".vault",
			Type: "file",
		},
		Security: SecurityConfig{
			PasswordPolicy: PassPolicy{
				MinLength: 8,
			},
		},
		Git: GitConfig{
			DeterministicMode: true,
		},
	}
	
	migrated, err := MigrateConfig(v1Config)
	if err != nil {
		t.Fatalf("MigrateConfig() error = %v", err)
	}
	
	// Verify final state
	if migrated.Version != CurrentVersion {
		t.Errorf("Final version = %v, want %v", migrated.Version, CurrentVersion)
	}
	
	// Verify v1->v2 migration happened
	if len(migrated.Environments) == 0 {
		t.Error("Environments should be created during v1->v2 migration")
	}
	
	// Verify v2->v3 migration happened
	if migrated.Import.DefaultParser.TrimSpace != true {
		t.Error("Import config should be set during v2->v3 migration")
	}
	
	if migrated.Export.DefaultFormat != "dotenv" {
		t.Error("Export config should be set during v2->v3 migration")
	}
	
	if migrated.Git.EncryptionMode != "deterministic" {
		t.Error("Git.EncryptionMode should be migrated from DeterministicMode")
	}
	
	// Verify original fields preserved
	if migrated.Project.Name != "migration-test" {
		t.Errorf("Project.Name = %v, want 'migration-test'", migrated.Project.Name)
	}
	
	if migrated.Vault.Path != ".vault" {
		t.Errorf("Vault.Path = %v, want '.vault'", migrated.Vault.Path)
	}
}

func TestMigrationIdempotency(t *testing.T) {
	// Test that running migration multiple times produces same result
	original := &Config{
		Version: "1.0.0",
		Project: ProjectConfig{
			Name: "test",
		},
		Environments: map[string]EnvironmentConfig{
			"custom": {
				Description: "Custom env",
			},
		},
	}
	
	// First migration
	first, err := MigrateConfig(original)
	if err != nil {
		t.Fatalf("First migration error = %v", err)
	}
	
	// Second migration (should be no-op)
	second, err := MigrateConfig(first)
	if err != nil {
		t.Fatalf("Second migration error = %v", err)
	}
	
	// Compare results (excluding version which gets updated)
	first.Version = ""
	second.Version = ""
	
	if !reflect.DeepEqual(first, second) {
		t.Error("Multiple migrations produced different results")
	}
}

func BenchmarkMigrateConfig(b *testing.B) {
	cfg := &Config{
		Version: "1.0.0",
		Project: ProjectConfig{
			Name: "bench-test",
		},
		Environments: map[string]EnvironmentConfig{
			"dev":  {},
			"prod": {},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy to avoid modifying the original
		cfgCopy := *cfg
		_, _ = MigrateConfig(&cfgCopy)
	}
}