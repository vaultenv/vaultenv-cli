package keystore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	keystoreDBName = "keystore.db"
	currentVersion = 1
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrKeyExists   = errors.New("key already exists")
)

// KeyEntry represents a stored encryption key
type KeyEntry struct {
	ProjectID        string    `json:"project_id"`
	Salt             []byte    `json:"salt"`
	VerificationHash string    `json:"verification_hash"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EnvironmentKeyEntry represents a stored encryption key for a specific environment
type EnvironmentKeyEntry struct {
	ProjectID        string    `json:"project_id"`
	Environment      string    `json:"environment"`
	Salt             []byte    `json:"salt"`
	VerificationHash string    `json:"verification_hash"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Algorithm        string    `json:"algorithm"`
	Iterations       uint32    `json:"iterations"`
	Memory           uint32    `json:"memory"`
	Parallelism      uint8     `json:"parallelism"`
}

// Keystore manages encryption keys
type Keystore struct {
	db     *sql.DB
	dbPath string
}

// NewKeystore creates a new keystore instance
func NewKeystore(dataDir string) (*Keystore, error) {
	dbPath := filepath.Join(dataDir, keystoreDBName)
	
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	
	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open keystore database: %w", err)
	}
	
	ks := &Keystore{
		db:     db,
		dbPath: dbPath,
	}
	
	// Initialize schema
	if err := ks.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize keystore schema: %w", err)
	}
	
	return ks, nil
}

// Close closes the keystore database
func (ks *Keystore) Close() error {
	return ks.db.Close()
}

// StoreKey stores an encryption key for a project
func (ks *Keystore) StoreKey(projectID string, entry *KeyEntry) error {
	entry.UpdatedAt = time.Now()
	
	// Serialize entry to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to serialize key entry: %w", err)
	}
	
	// Insert or replace
	query := `
		INSERT OR REPLACE INTO keys (project_id, data, updated_at)
		VALUES (?, ?, ?)
	`
	
	_, err = ks.db.Exec(query, projectID, data, entry.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to store key: %w", err)
	}
	
	return nil
}

// GetKey retrieves an encryption key for a project
func (ks *Keystore) GetKey(projectID string) (*KeyEntry, error) {
	query := `SELECT data FROM keys WHERE project_id = ?`
	
	var data []byte
	err := ks.db.QueryRow(query, projectID).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get key: %w", err)
	}
	
	var entry KeyEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to deserialize key entry: %w", err)
	}
	
	return &entry, nil
}

// DeleteKey removes an encryption key for a project
func (ks *Keystore) DeleteKey(projectID string) error {
	query := `DELETE FROM keys WHERE project_id = ?`
	
	result, err := ks.db.Exec(query, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return ErrKeyNotFound
	}
	
	return nil
}

// ListProjects returns all project IDs that have stored keys
func (ks *Keystore) ListProjects() ([]string, error) {
	query := `SELECT project_id FROM keys ORDER BY updated_at DESC`
	
	rows, err := ks.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()
	
	var projects []string
	for rows.Next() {
		var projectID string
		if err := rows.Scan(&projectID); err != nil {
			return nil, fmt.Errorf("failed to scan project ID: %w", err)
		}
		projects = append(projects, projectID)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate projects: %w", err)
	}
	
	return projects, nil
}

// Backup creates a backup of the keystore
func (ks *Keystore) Backup(backupPath string) error {
	// Ensure backup directory exists
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// Use SQLite backup API
	query := fmt.Sprintf("VACUUM INTO '%s'", backupPath)
	_, err := ks.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to backup keystore: %w", err)
	}
	
	return nil
}

// Restore restores the keystore from a backup
func (ks *Keystore) Restore(backupPath string) error {
	// Verify backup exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}
	
	// Close current database
	if err := ks.db.Close(); err != nil {
		return fmt.Errorf("failed to close current database: %w", err)
	}
	
	// Copy backup over current database
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}
	
	if err := os.WriteFile(ks.dbPath, backupData, 0600); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	
	// Reopen database
	db, err := sql.Open("sqlite3", ks.dbPath)
	if err != nil {
		return fmt.Errorf("failed to reopen database: %w", err)
	}
	
	ks.db = db
	return nil
}

// initSchema initializes the database schema
func (ks *Keystore) initSchema() error {
	// Create version table
	versionTable := `
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	
	if _, err := ks.db.Exec(versionTable); err != nil {
		return fmt.Errorf("failed to create version table: %w", err)
	}
	
	// Check current version
	var version sql.NullInt64
	err := ks.db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get schema version: %w", err)
	}
	
	currentVersion := 0
	if version.Valid {
		currentVersion = int(version.Int64)
	}
	
	// Apply migrations
	if currentVersion < 1 {
		if err := ks.applyMigration1(); err != nil {
			return fmt.Errorf("failed to apply migration 1: %w", err)
		}
	}
	
	if currentVersion < 2 {
		if err := ks.applyMigration2(); err != nil {
			return fmt.Errorf("failed to apply migration 2: %w", err)
		}
	}
	
	return nil
}

// applyMigration1 applies the initial schema
func (ks *Keystore) applyMigration1() error {
	tx, err := ks.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// Create keys table
	keysTable := `
		CREATE TABLE keys (
			project_id TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	
	if _, err := tx.Exec(keysTable); err != nil {
		return err
	}
	
	// Create index
	if _, err := tx.Exec("CREATE INDEX idx_keys_updated_at ON keys(updated_at)"); err != nil {
		return err
	}
	
	// Record migration
	if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (1)"); err != nil {
		return err
	}
	
	return tx.Commit()
}

// applyMigration2 adds support for per-environment keys
func (ks *Keystore) applyMigration2() error {
	tx, err := ks.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// Create environment_keys table
	envKeysTable := `
		CREATE TABLE environment_keys (
			project_id TEXT NOT NULL,
			environment TEXT NOT NULL,
			data BLOB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (project_id, environment)
		)
	`
	
	if _, err := tx.Exec(envKeysTable); err != nil {
		return err
	}
	
	// Create indexes
	if _, err := tx.Exec("CREATE INDEX idx_env_keys_project ON environment_keys(project_id)"); err != nil {
		return err
	}
	
	if _, err := tx.Exec("CREATE INDEX idx_env_keys_updated ON environment_keys(updated_at)"); err != nil {
		return err
	}
	
	// Record migration
	if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (2)"); err != nil {
		return err
	}
	
	return tx.Commit()
}

// StoreEnvironmentKey stores an encryption key for a specific environment
func (ks *Keystore) StoreEnvironmentKey(projectID, environment string, entry *EnvironmentKeyEntry) error {
	entry.UpdatedAt = time.Now()
	
	// Serialize entry to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to serialize key entry: %w", err)
	}
	
	// Insert or replace
	query := `
		INSERT OR REPLACE INTO environment_keys (project_id, environment, data, updated_at)
		VALUES (?, ?, ?, ?)
	`
	
	_, err = ks.db.Exec(query, projectID, environment, data, entry.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to store environment key: %w", err)
	}
	
	return nil
}

// GetEnvironmentKey retrieves an encryption key for a specific environment
func (ks *Keystore) GetEnvironmentKey(projectID, environment string) (*EnvironmentKeyEntry, error) {
	query := `SELECT data FROM environment_keys WHERE project_id = ? AND environment = ?`
	
	var data []byte
	err := ks.db.QueryRow(query, projectID, environment).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get environment key: %w", err)
	}
	
	var entry EnvironmentKeyEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to deserialize key entry: %w", err)
	}
	
	return &entry, nil
}

// DeleteEnvironmentKey removes an encryption key for a specific environment
func (ks *Keystore) DeleteEnvironmentKey(projectID, environment string) error {
	query := `DELETE FROM environment_keys WHERE project_id = ? AND environment = ?`
	
	result, err := ks.db.Exec(query, projectID, environment)
	if err != nil {
		return fmt.Errorf("failed to delete environment key: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return ErrKeyNotFound
	}
	
	return nil
}

// ListEnvironments returns all environments for a project that have stored keys
func (ks *Keystore) ListEnvironments(projectID string) ([]string, error) {
	query := `SELECT environment FROM environment_keys WHERE project_id = ? ORDER BY environment`
	
	rows, err := ks.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}
	defer rows.Close()
	
	var environments []string
	for rows.Next() {
		var environment string
		if err := rows.Scan(&environment); err != nil {
			return nil, fmt.Errorf("failed to scan environment: %w", err)
		}
		environments = append(environments, environment)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate environments: %w", err)
		}
	
	return environments, nil
}