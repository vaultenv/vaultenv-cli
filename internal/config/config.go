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
	Version  string          `yaml:"version"`
	Project  ProjectConfig   `yaml:"project"`
	Vault    VaultConfig     `yaml:"vault"`
	Security SecurityConfig  `yaml:"security"`
	Sync     SyncConfig      `yaml:"sync"`
	Git      GitConfig       `yaml:"git"`
	UI       UIConfig        `yaml:"ui"`
	Plugins  []PluginConfig  `yaml:"plugins,omitempty"`
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
	RequireMFA          bool     `yaml:"require_mfa"`
	AllowedIPs          []string `yaml:"allowed_ips,omitempty"`
	SessionTimeout      string   `yaml:"session_timeout"`
	MaxLoginAttempts    int      `yaml:"max_login_attempts"`
	PasswordPolicy      PassPolicy `yaml:"password_policy"`
	AuditLog            bool     `yaml:"audit_log"`
	SecureDelete        bool     `yaml:"secure_delete"`
	MemoryProtection    bool     `yaml:"memory_protection"`
}

// PassPolicy defines password requirements
type PassPolicy struct {
	MinLength      int  `yaml:"min_length"`
	RequireUpper   bool `yaml:"require_upper"`
	RequireLower   bool `yaml:"require_lower"`
	RequireNumbers bool `yaml:"require_numbers"`
	RequireSpecial bool `yaml:"require_special"`
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
	Enabled        bool     `yaml:"enabled"`
	AutoCommit     bool     `yaml:"auto_commit"`
	CommitMessage  string   `yaml:"commit_message,omitempty"`
	IgnorePatterns []string `yaml:"ignore_patterns,omitempty"`
	Hooks          GitHooks `yaml:"hooks,omitempty"`
}

// GitHooks defines Git hook configurations
type GitHooks struct {
	PreCommit  []string `yaml:"pre_commit,omitempty"`
	PostCommit []string `yaml:"post_commit,omitempty"`
	PrePush    []string `yaml:"pre_push,omitempty"`
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
		Version: "1.0",
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
			},
			AuditLog:         true,
			SecureDelete:     true,
			MemoryProtection: true,
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
			CommitMessage: "Update vault configuration",
			IgnorePatterns: []string{
				"*.key",
				"*.pem",
				"*.p12",
				".env.local",
			},
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
	validTypes := map[string]bool{"file": true, "sqlite": true, "cloud": true}
	if !validTypes[c.Vault.Type] {
		return fmt.Errorf("vault type must be 'file', 'sqlite', or 'cloud'")
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