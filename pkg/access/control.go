package access

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AccessControl defines the interface for environment access control
type AccessControl interface {
	// Check if user has access to environment
	HasAccess(user, environment string) (bool, error)

	// Grant access to user for environment
	GrantAccess(user, environment string, level AccessLevel) error

	// Revoke access from user for environment
	RevokeAccess(user, environment string) error

	// List users with access to environment
	ListAccess(environment string) ([]AccessEntry, error)
}

// AccessLevel defines the level of access a user has
type AccessLevel string

const (
	AccessLevelRead  AccessLevel = "read"
	AccessLevelWrite AccessLevel = "write"
	AccessLevelAdmin AccessLevel = "admin"
)

// AccessEntry represents a user's access to an environment
type AccessEntry struct {
	User        string      `json:"user"`
	Environment string      `json:"environment"`
	Level       AccessLevel `json:"level"`
	GrantedAt   time.Time   `json:"granted_at"`
	GrantedBy   string      `json:"granted_by"`
	ExpiresAt   *time.Time  `json:"expires_at,omitempty"`
}

// LocalAccessControl implements file-based access control for the open source version
type LocalAccessControl struct {
	configPath string
}

// NewLocalAccessControl creates a new local access control instance
func NewLocalAccessControl(configPath string) *LocalAccessControl {
	return &LocalAccessControl{
		configPath: configPath,
	}
}

// AccessConfig represents the access control configuration
type AccessConfig struct {
	Environments map[string]*EnvironmentAccess `json:"environments"`
	UpdatedAt    time.Time                     `json:"updated_at"`
}

// EnvironmentAccess defines access rules for an environment
type EnvironmentAccess struct {
	AllowedUsers []string `json:"allowed_users"`
	AllowedRoles []string `json:"allowed_roles"`
	Entries      []AccessEntry `json:"entries"`
}

// HasAccess checks if a user has access to an environment
func (l *LocalAccessControl) HasAccess(user, environment string) (bool, error) {
	// In local mode, check against config file
	config, err := l.loadAccessConfig()
	if err != nil {
		return false, err
	}

	// Check environment access rules
	envConfig, exists := config.Environments[environment]
	if !exists {
		// No specific rules, deny by default
		return false, nil
	}

	// Check if user is in allowed users list
	for _, allowedUser := range envConfig.AllowedUsers {
		if allowedUser == user || allowedUser == "*" {
			return true, nil
		}
	}

	// Check specific access entries
	for _, entry := range envConfig.Entries {
		if entry.User == user && entry.Environment == environment {
			// Check if access has expired
			if entry.ExpiresAt != nil && entry.ExpiresAt.Before(time.Now()) {
				continue
			}
			return true, nil
		}
	}

	// Check if user matches any role
	userRoles := l.getUserRoles(user)
	for _, role := range envConfig.AllowedRoles {
		if contains(userRoles, role) {
			return true, nil
		}
	}

	return false, nil
}

// GrantAccess grants access to a user for an environment
func (l *LocalAccessControl) GrantAccess(user, environment string, level AccessLevel) error {
	config, err := l.loadAccessConfig()
	if err != nil {
		return err
	}

	// Ensure environment exists in config
	if config.Environments == nil {
		config.Environments = make(map[string]*EnvironmentAccess)
	}
	if config.Environments[environment] == nil {
		config.Environments[environment] = &EnvironmentAccess{
			AllowedUsers: []string{},
			AllowedRoles: []string{},
			Entries:      []AccessEntry{},
		}
	}

	envConfig := config.Environments[environment]

	// Check if user already has access
	for i, entry := range envConfig.Entries {
		if entry.User == user && entry.Environment == environment {
			// Update existing entry
			envConfig.Entries[i].Level = level
			envConfig.Entries[i].GrantedAt = time.Now()
			envConfig.Entries[i].GrantedBy = getCurrentUser()
			return l.saveAccessConfig(config)
		}
	}

	// Add new access entry
	entry := AccessEntry{
		User:        user,
		Environment: environment,
		Level:       level,
		GrantedAt:   time.Now(),
		GrantedBy:   getCurrentUser(),
	}

	envConfig.Entries = append(envConfig.Entries, entry)
	
	// Also add to allowed users if not already present
	if !contains(envConfig.AllowedUsers, user) {
		envConfig.AllowedUsers = append(envConfig.AllowedUsers, user)
	}

	return l.saveAccessConfig(config)
}

// RevokeAccess revokes access from a user for an environment
func (l *LocalAccessControl) RevokeAccess(user, environment string) error {
	config, err := l.loadAccessConfig()
	if err != nil {
		return err
	}

	envConfig, exists := config.Environments[environment]
	if !exists {
		return nil // Nothing to revoke
	}

	// Remove from allowed users
	envConfig.AllowedUsers = removeString(envConfig.AllowedUsers, user)

	// Remove from entries
	var newEntries []AccessEntry
	for _, entry := range envConfig.Entries {
		if !(entry.User == user && entry.Environment == environment) {
			newEntries = append(newEntries, entry)
		}
	}
	envConfig.Entries = newEntries

	return l.saveAccessConfig(config)
}

// ListAccess lists users with access to an environment
func (l *LocalAccessControl) ListAccess(environment string) ([]AccessEntry, error) {
	config, err := l.loadAccessConfig()
	if err != nil {
		return nil, err
	}

	envConfig, exists := config.Environments[environment]
	if !exists {
		return []AccessEntry{}, nil
	}

	return envConfig.Entries, nil
}

// loadAccessConfig loads the access configuration from file
func (l *LocalAccessControl) loadAccessConfig() (*AccessConfig, error) {
	accessPath := filepath.Join(filepath.Dir(l.configPath), "access.json")

	data, err := os.ReadFile(accessPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config
			return &AccessConfig{
				Environments: make(map[string]*EnvironmentAccess),
				UpdatedAt:    time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to read access config: %w", err)
	}

	var config AccessConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse access config: %w", err)
	}

	return &config, nil
}

// saveAccessConfig saves the access configuration to file
func (l *LocalAccessControl) saveAccessConfig(config *AccessConfig) error {
	config.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal access config: %w", err)
	}

	accessPath := filepath.Join(filepath.Dir(l.configPath), "access.json")
	if err := os.WriteFile(accessPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write access config: %w", err)
	}

	return nil
}

// getUserRoles returns the roles for a user (placeholder for local implementation)
func (l *LocalAccessControl) getUserRoles(user string) []string {
	// In a real implementation, this would look up user roles
	// For local version, we can use OS groups or a simple mapping
	return []string{}
}

// Helper functions

func getCurrentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeString(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}