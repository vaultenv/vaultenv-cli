package config

import (
	"fmt"
)

// CurrentVersion is the current version of the configuration format
const CurrentVersion = "3.0.0"

// MigrateConfig applies necessary migrations to bring config to current version
func MigrateConfig(cfg *Config) (*Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	// Apply migrations based on version
	switch cfg.Version {
	case "", "1.0", "1.0.0":
		cfg = migrateV1ToV2(cfg)
		fallthrough
	case "2.0.0":
		cfg = migrateV2ToV3(cfg)
	case CurrentVersion:
		// Already at current version
		return cfg, nil
	default:
		// Unknown version, try to migrate anyway
		cfg = migrateV1ToV2(cfg)
		cfg = migrateV2ToV3(cfg)
	}

	cfg.Version = CurrentVersion
	return cfg, nil
}

// migrateV1ToV2 migrates from version 1.x to 2.0.0
// Main changes: Per-environment passwords support
func migrateV1ToV2(old *Config) *Config {
	// Convert single password to per-environment
	if old.Environments == nil {
		old.Environments = make(map[string]EnvironmentConfig)
	}

	// Ensure all environments have password protection enabled
	for env, config := range old.Environments {
		config.PasswordProtected = true

		// If no password policy is set, use sensible defaults based on environment
		if config.PasswordPolicy.MinLength == 0 {
			switch env {
			case "development":
				config.PasswordPolicy = PassPolicy{
					MinLength:      8,
					RequireUpper:   false,
					RequireLower:   true,
					RequireNumbers: false,
					PreventCommon:  false,
				}
			case "staging":
				config.PasswordPolicy = PassPolicy{
					MinLength:      12,
					RequireUpper:   true,
					RequireLower:   true,
					RequireNumbers: true,
					PreventCommon:  true,
				}
			case "production":
				config.PasswordPolicy = PassPolicy{
					MinLength:      16,
					RequireUpper:   true,
					RequireLower:   true,
					RequireNumbers: true,
					RequireSpecial: true,
					PreventCommon:  true,
					ExpiryDays:     90,
				}
			default:
				// Use global policy as fallback
				config.PasswordPolicy = old.Security.PasswordPolicy
			}
		}

		old.Environments[env] = config
	}

	// If no environments exist, create default ones
	if len(old.Environments) == 0 {
		old.Environments = map[string]EnvironmentConfig{
			"development": {
				Description:       "Development environment",
				PasswordProtected: true,
				PasswordPolicy: PassPolicy{
					MinLength: 8,
				},
			},
			"production": {
				Description:       "Production environment",
				PasswordProtected: true,
				PasswordPolicy: PassPolicy{
					MinLength: 16,
				},
			},
		}
	}

	// Update version
	old.Version = "2.0.0"
	return old
}

// migrateV2ToV3 migrates from version 2.0.0 to 3.0.0
// Main changes: Import/Export config, enhanced Git config
func migrateV2ToV3(old *Config) *Config {
	// Add Import configuration if missing
	// Check if Import config is completely empty (all fields at zero values)
	if old.Import.DefaultParser == (ParserConfig{}) && old.Import.AutoBackup == false && old.Import.ValidateFormat == false {
		old.Import = ImportConfig{
			DefaultParser: ParserConfig{
				TrimSpace:      true,
				ExpandVars:     false,
				IgnoreComments: true,
				IgnoreEmpty:    true,
				IgnoreInvalid:  false,
			},
			AutoBackup:     true,
			ValidateFormat: true,
		}
	}

	// Add Export configuration if missing
	if old.Export.DefaultFormat == "" {
		old.Export = ExportConfig{
			DefaultFormat:   "dotenv",
			IncludeMetadata: false,
			Templates:       make(map[string]string),
		}
	}

	// Update Git configuration
	if old.Git.ConflictStrategy == "" {
		old.Git.ConflictStrategy = "prompt"
	}

	// Migrate DeterministicMode to EncryptionMode
	if old.Git.DeterministicMode && old.Git.EncryptionMode == "" {
		old.Git.EncryptionMode = "deterministic"
	} else if old.Git.EncryptionMode == "" {
		old.Git.EncryptionMode = "random"
	}

	// Ensure AutoPush is set
	// (AutoPush is a new field, so it will be false by default)

	// Update version
	old.Version = "3.0.0"
	return old
}

// IsLegacyConfig checks if the config is from an older version
func IsLegacyConfig(cfg *Config) bool {
	return cfg.Version == "" || cfg.Version < "2.0.0"
}

// NeedsMigration checks if the config needs migration
func NeedsMigration(cfg *Config) bool {
	return cfg.Version != CurrentVersion
}
