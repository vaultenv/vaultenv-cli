package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete vaultenv configuration
type Config struct {
	Version      string                        `yaml:"version"`
	Project      ProjectConfig                 `yaml:"project"`
	Environments map[string]EnvironmentConfig `yaml:"environments,omitempty"`
	Vault        VaultConfig                   `yaml:"vault"`
	Security     SecurityConfig                `yaml:"security"`
	Sync         SyncConfig                    `yaml:"sync"`
	Git          GitConfig                     `yaml:"git"`
	Import       ImportConfig                  `yaml:"import"`
	Export       ExportConfig                  `yaml:"export"`
	UI           UIConfig                      `yaml:"ui"`
	Plugins      []PluginConfig                `yaml:"plugins,omitempty"`
}

// ProjectConfig holds project-specific settings
type ProjectConfig struct {
	Name        string            `yaml:"name"`
	ID          string            `yaml:"id"`
	Description string            `yaml:"description,omitempty"`
	Team        string            `yaml:"team,omitempty"`
	Tags        []string          `yaml:"tags,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
}

// VaultConfig defines vault storage settings
type VaultConfig struct {
	Path            string        `yaml:"path"`
	Type            string        `yaml:"type"` // "file", "sqlite", or "cloud"
	EncryptionAlgo  string        `yaml:"encryption_algo"`
	KeyDerivation   KDFConfig     `yaml:"key_derivation"`
	AutoLock        bool          `yaml:"auto_lock"`
	LockTimeout     time.Duration `yaml:"lock_timeout"`
	BackupEnabled   bool          `yaml:"backup_enabled"`
	BackupPath      string        `yaml:"backup_path,omitempty"`
	BackupRetention int           `yaml:"backup_retention,omitempty"`
}

// KDFConfig holds key derivation function settings
type KDFConfig struct {
	Algorithm   string `yaml:"algorithm"` // "argon2id", "scrypt", "pbkdf2"
	Iterations  int    `yaml:"iterations,omitempty"`
	Memory      int    `yaml:"memory,omitempty"`
	Parallelism int    `yaml:"parallelism,omitempty"`
	SaltLength  int    `yaml:"salt_length,omitempty"`
}

// SecurityConfig contains security-related settings
type SecurityConfig struct {
	RequireMFA               bool       `yaml:"require_mfa"`
	AllowedIPs               []string   `yaml:"allowed_ips,omitempty"`
	SessionTimeout           string     `yaml:"session_timeout"`
	MaxLoginAttempts         int        `yaml:"max_login_attempts"`
	PasswordPolicy           PassPolicy `yaml:"password_policy"`
	AuditLog                 bool       `yaml:"audit_log"`
	SecureDelete             bool       `yaml:"secure_delete"`
	MemoryProtection         bool       `yaml:"memory_protection"`
	PerEnvironmentPasswords  bool       `yaml:"per_environment_passwords"`
}

// PassPolicy defines password requirements
type PassPolicy struct {
	MinLength      int  `yaml:"min_length"`
	RequireUpper   bool `yaml:"require_upper"`
	RequireLower   bool `yaml:"require_lower"`
	RequireNumbers bool `yaml:"require_numbers"`
	RequireSpecial bool `yaml:"require_special"`
	PreventCommon  bool `yaml:"prevent_common"`
	ExpiryDays     int  `yaml:"expiry_days"`
}

// EnvironmentConfig holds environment-specific settings
type EnvironmentConfig struct {
	Description       string     `yaml:"description,omitempty"`
	PasswordProtected bool       `yaml:"password_protected"`
	PasswordPolicy    PassPolicy `yaml:"password_policy,omitempty"`
	AutoLoad          string     `yaml:"auto_load,omitempty"`
	Restrictions      []string   `yaml:"restrictions,omitempty"`
	RequireApproval   bool       `yaml:"require_approval,omitempty"`
	Notifications     bool       `yaml:"notifications,omitempty"`
}

// SyncConfig handles synchronization settings
type SyncConfig struct {
	Enabled       bool          `yaml:"enabled"`
	URL           string        `yaml:"url,omitempty"`
	Interval      time.Duration `yaml:"interval"`
	AutoSync      bool          `yaml:"auto_sync"`
	ConflictMode  string        `yaml:"conflict_mode"` // "manual", "local", "remote"
	Compression   bool          `yaml:"compression"`
	BatchSize     int           `yaml:"batch_size"`
	RetryAttempts int           `yaml:"retry_attempts"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
}

// GitConfig defines Git integration settings
type GitConfig struct {
	Enabled            bool     `yaml:"enabled"`
	AutoCommit         bool     `yaml:"auto_commit"`
	AutoPush           bool     `yaml:"auto_push"`
	CommitMessage      string   `yaml:"commit_message,omitempty"`
	CommitTemplate     string   `yaml:"commit_template,omitempty"`
	ConflictStrategy   string   `yaml:"conflict_strategy,omitempty"` // "prompt", "ours", "theirs", "newest"
	EncryptionMode     string   `yaml:"encryption_mode"`             // "deterministic" or "random"
	DeterministicMode  bool     `yaml:"deterministic_mode"`          // Use deterministic encryption for git (deprecated, use EncryptionMode)
	IgnorePatterns     []string `yaml:"ignore_patterns,omitempty"`
	Hooks              GitHooks `yaml:"hooks,omitempty"`
}

// GitHooks defines Git hook configurations
type GitHooks struct {
	PreCommit  []string `yaml:"pre_commit,omitempty"`
	PostCommit []string `yaml:"post_commit,omitempty"`
	PrePush    []string `yaml:"pre_push,omitempty"`
}

// ImportConfig defines import behavior settings
type ImportConfig struct {
	DefaultParser  ParserConfig `yaml:"default_parser"`
	AutoBackup     bool         `yaml:"auto_backup"`
	ValidateFormat bool         `yaml:"validate_format"`
}

// ParserConfig defines parser settings for import
type ParserConfig struct {
	TrimSpace      bool `yaml:"trim_space"`
	ExpandVars     bool `yaml:"expand_vars"`
	IgnoreComments bool `yaml:"ignore_comments"`
	IgnoreEmpty    bool `yaml:"ignore_empty"`
	IgnoreInvalid  bool `yaml:"ignore_invalid"`
}

// ExportConfig defines export behavior settings
type ExportConfig struct {
	DefaultFormat   string            `yaml:"default_format"`
	IncludeMetadata bool              `yaml:"include_metadata"`
	Templates       map[string]string `yaml:"templates"`
}

// UIConfig contains UI/UX preferences
type UIConfig struct {
	Theme           string `yaml:"theme"` // "dark", "light", "auto"
	Color           bool   `yaml:"color"`
	Emoji           bool   `yaml:"emoji"`
	ProgressBar     bool   `yaml:"progress_bar"`
	Notifications   bool   `yaml:"notifications"`
	DateFormat      string `yaml:"date_format,omitempty"`
	TimeFormat      string `yaml:"time_format,omitempty"`
	Language        string `yaml:"language,omitempty"`
}

// PluginConfig represents a plugin configuration
type PluginConfig struct {
	Name    string                 `yaml:"name"`
	Enabled bool                   `yaml:"enabled"`
	Path    string                 `yaml:"path,omitempty"`
	Config  map[string]interface{} `yaml:"config,omitempty"`
}

// DefaultConfig returns a new Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Version: CurrentVersion,
		Project: ProjectConfig{
			Name: "default",
			ID:   "",
		},
		Vault: VaultConfig{
			Path:           ".vaultenv",
			Type:           "file",
			EncryptionAlgo: "aes-256-gcm",
			KeyDerivation: KDFConfig{
				Algorithm:   "argon2id",
				Iterations:  3,
				Memory:      64 * 1024, // 64MB
				Parallelism: 4,
				SaltLength:  32,
			},
			AutoLock:        true,
			LockTimeout:     15 * time.Minute,
			BackupEnabled:   true,
			BackupPath:      ".vaultenv/backups",
			BackupRetention: 7,
		},
		Security: SecurityConfig{
			RequireMFA:       false,
			SessionTimeout:   "24h",
			MaxLoginAttempts: 5,
			PasswordPolicy: PassPolicy{
				MinLength:      12,
				RequireUpper:   true,
				RequireLower:   true,
				RequireNumbers: true,
				RequireSpecial: true,
				PreventCommon:  false,
				ExpiryDays:     0,
			},
			AuditLog:                true,
			SecureDelete:            true,
			MemoryProtection:        true,
			PerEnvironmentPasswords: false, // Default to legacy mode
		},
		Environments: map[string]EnvironmentConfig{
			"development": {
				Description: "Development environment",
				PasswordProtected: true,
				PasswordPolicy: PassPolicy{
					MinLength:      8,
					RequireUpper:   false,
					RequireNumbers: false,
					PreventCommon:  false,
				},
				Notifications: true,
			},
			"staging": {
				Description: "Staging environment",
				PasswordProtected: true,
				PasswordPolicy: PassPolicy{
					MinLength:      12,
					RequireUpper:   true,
					RequireLower:   true,
					RequireNumbers: true,
					PreventCommon:  true,
				},
				RequireApproval: true,
				Notifications:   true,
			},
			"production": {
				Description: "Production environment",
				PasswordProtected: true,
				PasswordPolicy: PassPolicy{
					MinLength:      16,
					RequireUpper:   true,
					RequireLower:   true,
					RequireNumbers: true,
					RequireSpecial: true,
					PreventCommon:  true,
					ExpiryDays:     90,
				},
				RequireApproval: true,
				Notifications:   true,
			},
		},
		Sync: SyncConfig{
			Enabled:       false,
			Interval:      5 * time.Minute,
			AutoSync:      false,
			ConflictMode:  "manual",
			Compression:   true,
			BatchSize:     100,
			RetryAttempts: 3,
			RetryDelay:    5 * time.Second,
		},
		Git: GitConfig{
			Enabled:       true,
			AutoCommit:    false,
			AutoPush:      false,
			CommitMessage: "Update vault configuration",
			ConflictStrategy: "prompt",
			EncryptionMode: "random",
			IgnorePatterns: []string{
				"*.key",
				"*.pem",
				"*.p12",
				".env.local",
			},
		},
		Import: ImportConfig{
			DefaultParser: ParserConfig{
				TrimSpace:      true,
				ExpandVars:     false,
				IgnoreComments: true,
				IgnoreEmpty:    true,
				IgnoreInvalid:  false,
			},
			AutoBackup:     true,
			ValidateFormat: true,
		},
		Export: ExportConfig{
			DefaultFormat:   "dotenv",
			IncludeMetadata: false,
			Templates:       make(map[string]string),
		},
		UI: UIConfig{
			Theme:         "dark",
			Color:         true,
			Emoji:         true,
			ProgressBar:   true,
			Notifications: true,
			DateFormat:    "2006-01-02",
			TimeFormat:    "15:04:05",
			Language:      "en",
		},
		Plugins: []PluginConfig{},
	}
}

// Load reads configuration from file system, walking up directory tree if needed
func Load() (*Config, error) {
	configPath, err := findConfigFile()
	if err != nil {
		// If no config file exists, return default config
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("error finding config file: %w", err)
	}

	return LoadFromFile(configPath)
}

// LoadFromFile reads configuration from a specific file
func LoadFromFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	return LoadFromReader(file)
}

// LoadFromReader reads configuration from an io.Reader
func LoadFromReader(r io.Reader) (*Config, error) {
	config := DefaultConfig()
	
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Apply migrations if needed
	if NeedsMigration(config) {
		migratedConfig, err := MigrateConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to migrate config: %w", err)
		}
		config = migratedConfig
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Save writes the configuration to the default location
func (c *Config) Save() error {
	configPath := filepath.Join(".vaultenv", "config.yaml")
	return c.SaveToFile(configPath)
}

// SaveToFile writes the configuration to a specific file
func (c *Config) SaveToFile(path string) error {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	return c.SaveToWriter(file)
}

// SaveToWriter writes the configuration to an io.Writer
func (c *Config) SaveToWriter(w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	defer encoder.Close()

	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate version
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}

	// Validate project
	if c.Project.Name == "" {
		return fmt.Errorf("project name is required")
	}

	// Validate vault settings
	validTypes := map[string]bool{"file": true, "sqlite": true, "git": true, "cloud": true}
	if !validTypes[c.Vault.Type] {
		return fmt.Errorf("vault type must be 'file', 'sqlite', 'git', or 'cloud'")
	}

	// Validate encryption algorithm
	validAlgos := map[string]bool{
		"aes-256-gcm": true,
		"chacha20-poly1305": true,
	}
	if !validAlgos[c.Vault.EncryptionAlgo] {
		return fmt.Errorf("unsupported encryption algorithm: %s", c.Vault.EncryptionAlgo)
	}

	// Validate KDF settings
	validKDFs := map[string]bool{
		"argon2id": true,
		"scrypt":   true,
		"pbkdf2":   true,
	}
	if !validKDFs[c.Vault.KeyDerivation.Algorithm] {
		return fmt.Errorf("unsupported KDF algorithm: %s", c.Vault.KeyDerivation.Algorithm)
	}

	// Validate sync conflict mode
	validConflictModes := map[string]bool{
		"manual": true,
		"local":  true,
		"remote": true,
	}
	if c.Sync.Enabled && !validConflictModes[c.Sync.ConflictMode] {
		return fmt.Errorf("invalid sync conflict mode: %s", c.Sync.ConflictMode)
	}

	// Validate UI theme
	validThemes := map[string]bool{
		"dark":  true,
		"light": true,
		"auto":  true,
	}
	if !validThemes[c.UI.Theme] {
		return fmt.Errorf("invalid UI theme: %s", c.UI.Theme)
	}

	return nil
}

// Merge combines this config with another, with the other config taking precedence
func (c *Config) Merge(other *Config) *Config {
	if other == nil {
		return c
	}

	// Deep copy the current config
	merged := *c

	// Merge fields where other has non-zero values
	if other.Version != "" {
		merged.Version = other.Version
	}

	// Merge project config
	if other.Project.Name != "" {
		merged.Project.Name = other.Project.Name
	}
	if other.Project.ID != "" {
		merged.Project.ID = other.Project.ID
	}
	if other.Project.Description != "" {
		merged.Project.Description = other.Project.Description
	}
	if other.Project.Team != "" {
		merged.Project.Team = other.Project.Team
	}
	if len(other.Project.Tags) > 0 {
		merged.Project.Tags = other.Project.Tags
	}
	if len(other.Project.Metadata) > 0 {
		merged.Project.Metadata = other.Project.Metadata
	}

	// Continue with other sections...
	// This is a simplified merge; in production, you'd want deep merging

	return &merged
}

// findConfigFile walks up the directory tree looking for .vaultenv/config.yaml
func findConfigFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		configPath := filepath.Join(dir, ".vaultenv", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Check if we've reached the root
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", os.ErrNotExist
}

// GetVaultPath returns the absolute path to the vault file
func (c *Config) GetVaultPath() string {
	if filepath.IsAbs(c.Vault.Path) {
		return c.Vault.Path
	}

	// If relative, make it relative to the config directory
	configDir := filepath.Dir(c.Vault.Path)
	if configDir == "" || configDir == "." {
		configDir = ".vaultenv"
	}
	
	return filepath.Join(configDir, filepath.Base(c.Vault.Path))
}

// IsLocked checks if the vault should be locked based on timeout
func (c *Config) IsLocked(lastActivity time.Time) bool {
	if !c.Vault.AutoLock {
		return false
	}

	return time.Since(lastActivity) > c.Vault.LockTimeout
}

// IsEncrypted returns true if the vault is configured to use encryption
func (v *VaultConfig) IsEncrypted() bool {
	// A vault is encrypted if it has an encryption algorithm set
	return v.EncryptionAlgo != "" && v.EncryptionAlgo != "none"
}

// GetEnvironmentNames returns a list of all configured environment names
func (c *Config) GetEnvironmentNames() []string {
	names := make([]string, 0, len(c.Environments))
	for name := range c.Environments {
		names = append(names, name)
	}
	return names
}

// GetEnvironmentConfig returns the configuration for a specific environment
func (c *Config) GetEnvironmentConfig(name string) (EnvironmentConfig, bool) {
	config, exists := c.Environments[name]
	return config, exists
}

// SetEnvironmentConfig sets the configuration for a specific environment
func (c *Config) SetEnvironmentConfig(name string, config EnvironmentConfig) {
	if c.Environments == nil {
		c.Environments = make(map[string]EnvironmentConfig)
	}
	c.Environments[name] = config
}

// HasEnvironment checks if an environment is configured
func (c *Config) HasEnvironment(name string) bool {
	_, exists := c.Environments[name]
	return exists
}

// GetPasswordPolicy returns the password policy for an environment, falling back to global policy
func (c *Config) GetPasswordPolicy(environment string) PassPolicy {
	if envConfig, exists := c.Environments[environment]; exists {
		// If environment has its own policy, use it
		if envConfig.PasswordPolicy.MinLength > 0 {
			return envConfig.PasswordPolicy
		}
	}
	
	// Fall back to global policy
	return c.Security.PasswordPolicy
}

// IsPerEnvironmentPasswordsEnabled returns true if per-environment passwords are enabled
func (c *Config) IsPerEnvironmentPasswordsEnabled() bool {
	return c.Security.PerEnvironmentPasswords
}

// EnablePerEnvironmentPasswords enables per-environment password mode
func (c *Config) EnablePerEnvironmentPasswords() {
	c.Security.PerEnvironmentPasswords = true
	c.Version = "2.0.0" // Bump version to indicate new features
}