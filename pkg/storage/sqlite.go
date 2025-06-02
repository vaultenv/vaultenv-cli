package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteBackend implements the Backend interface using SQLite
type SQLiteBackend struct {
	db          *sql.DB
	environment string
}

// SecretHistory represents a historical version of a secret
type SecretHistory struct {
	Version    int       `json:"version"`
	Value      string    `json:"value"`
	ChangedAt  time.Time `json:"changed_at"`
	ChangedBy  string    `json:"changed_by"`
	ChangeType string    `json:"change_type"`
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	Action       string    `json:"action"`
	Key          string    `json:"key"`
	User         string    `json:"user"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// HistoryBackend extends Backend with history capabilities
type HistoryBackend interface {
	Backend
	GetHistory(key string, limit int) ([]SecretHistory, error)
	GetAuditLog(limit int) ([]AuditEntry, error)
}

// NewSQLiteBackend creates a new SQLite storage backend
func NewSQLiteBackend(basePath, environment string) (*SQLiteBackend, error) {
	// Create database directory
	dbPath := filepath.Join(basePath, "vaultenv.db")
	dbDir := filepath.Dir(dbPath)

	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite3", dbPath+"?mode=rwc&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings for better concurrency
	db.SetMaxOpenConns(1) // SQLite performs best with a single connection
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	backend := &SQLiteBackend{
		db:          db,
		environment: environment,
	}

	// Initialize schema
	if err := backend.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return backend, nil
}

func (s *SQLiteBackend) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS secrets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		environment TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT,
		updated_by TEXT,
		version INTEGER DEFAULT 1,
		UNIQUE(environment, key)
	);
	
	CREATE INDEX IF NOT EXISTS idx_env_key ON secrets(environment, key);
	
	CREATE TABLE IF NOT EXISTS secret_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		secret_id INTEGER NOT NULL,
		environment TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		version INTEGER NOT NULL,
		changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		changed_by TEXT,
		change_type TEXT NOT NULL,
		FOREIGN KEY (secret_id) REFERENCES secrets(id)
	);
	
	CREATE TABLE IF NOT EXISTS audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		environment TEXT NOT NULL,
		action TEXT NOT NULL,
		key TEXT,
		user TEXT,
		ip_address TEXT,
		user_agent TEXT,
		success BOOLEAN DEFAULT TRUE,
		error_message TEXT
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Set stores a variable with optional encryption
func (s *SQLiteBackend) Set(key, value string, encrypt bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if key exists
	var id int64
	var version int
	err = tx.QueryRow(`
		SELECT id, version FROM secrets 
		WHERE environment = ? AND key = ?
	`, s.environment, key).Scan(&id, &version)

	if err == sql.ErrNoRows {
		// Insert new secret
		result, err := tx.Exec(`
			INSERT INTO secrets (environment, key, value, created_by, updated_by)
			VALUES (?, ?, ?, ?, ?)
		`, s.environment, key, value, getCurrentUser(), getCurrentUser())

		if err != nil {
			return fmt.Errorf("failed to insert secret: %w", err)
		}

		id, _ = result.LastInsertId()
		version = 1
	} else if err != nil {
		return fmt.Errorf("failed to query secret: %w", err)
	} else {
		// Update existing secret
		version++
		_, err = tx.Exec(`
			UPDATE secrets 
			SET value = ?, updated_at = CURRENT_TIMESTAMP, 
				updated_by = ?, version = ?
			WHERE id = ?
		`, value, getCurrentUser(), version, id)

		if err != nil {
			return fmt.Errorf("failed to update secret: %w", err)
		}
	}

	// Add to history
	_, err = tx.Exec(`
		INSERT INTO secret_history (secret_id, environment, key, value, version, changed_by, change_type)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, s.environment, key, value, version, getCurrentUser(), "SET")

	if err != nil {
		return fmt.Errorf("failed to add history: %w", err)
	}

	// Add audit log
	_, err = tx.Exec(`
		INSERT INTO audit_log (environment, action, key, user, success)
		VALUES (?, ?, ?, ?, ?)
	`, s.environment, "SET", key, getCurrentUser(), true)

	if err != nil {
		return fmt.Errorf("failed to add audit log: %w", err)
	}

	return tx.Commit()
}

// Get retrieves a variable value
func (s *SQLiteBackend) Get(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`
		SELECT value FROM secrets 
		WHERE environment = ? AND key = ?
	`, s.environment, key).Scan(&value)

	// Add audit log (async to not slow down reads)
	success := err == nil
	go func() {
		s.db.Exec(`
			INSERT INTO audit_log (environment, action, key, user, success)
			VALUES (?, ?, ?, ?, ?)
		`, s.environment, "GET", key, getCurrentUser(), success)
	}()

	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}

	if err != nil {
		return "", fmt.Errorf("failed to get secret: %w", err)
	}

	return value, nil
}

// Exists checks if a variable exists
func (s *SQLiteBackend) Exists(key string) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM secrets 
		WHERE environment = ? AND key = ?
	`, s.environment, key).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return count > 0, nil
}

// Delete removes a variable
func (s *SQLiteBackend) Delete(key string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get secret ID for history
	var id int64
	var version int
	err = tx.QueryRow(`
		SELECT id, version FROM secrets 
		WHERE environment = ? AND key = ?
	`, s.environment, key).Scan(&id, &version)

	if err == sql.ErrNoRows {
		// Key doesn't exist, nothing to delete
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to query secret: %w", err)
	}

	// Add to history before deletion
	_, err = tx.Exec(`
		INSERT INTO secret_history (secret_id, environment, key, value, version, changed_by, change_type)
		SELECT id, environment, key, value, version+1, ?, 'DELETE'
		FROM secrets WHERE id = ?
	`, getCurrentUser(), id)

	if err != nil {
		return fmt.Errorf("failed to add history: %w", err)
	}

	// Delete the secret
	_, err = tx.Exec(`
		DELETE FROM secrets WHERE id = ?
	`, id)

	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	// Add audit log
	_, err = tx.Exec(`
		INSERT INTO audit_log (environment, action, key, user, success)
		VALUES (?, ?, ?, ?, ?)
	`, s.environment, "DELETE", key, getCurrentUser(), true)

	if err != nil {
		return fmt.Errorf("failed to add audit log: %w", err)
	}

	return tx.Commit()
}

// List returns all variable names
func (s *SQLiteBackend) List() ([]string, error) {
	rows, err := s.db.Query(`
		SELECT key FROM secrets 
		WHERE environment = ?
		ORDER BY key
	`, s.environment)

	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	// Add audit log for LIST operation
	go func() {
		s.db.Exec(`
			INSERT INTO audit_log (environment, action, key, user, success)
			VALUES (?, ?, ?, ?, ?)
		`, s.environment, "LIST", "", getCurrentUser(), true)
	}()

	return keys, rows.Err()
}

// Close closes the storage backend
func (s *SQLiteBackend) Close() error {
	return s.db.Close()
}

// GetHistory returns the change history for a specific key
func (s *SQLiteBackend) GetHistory(key string, limit int) ([]SecretHistory, error) {
	rows, err := s.db.Query(`
		SELECT version, value, changed_at, changed_by, change_type
		FROM secret_history
		WHERE environment = ? AND key = ?
		ORDER BY version DESC
		LIMIT ?
	`, s.environment, key, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []SecretHistory
	for rows.Next() {
		var h SecretHistory
		err := rows.Scan(&h.Version, &h.Value, &h.ChangedAt, &h.ChangedBy, &h.ChangeType)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, rows.Err()
}

// GetAuditLog returns recent audit log entries
func (s *SQLiteBackend) GetAuditLog(limit int) ([]AuditEntry, error) {
	rows, err := s.db.Query(`
		SELECT timestamp, action, key, user, success, error_message
		FROM audit_log
		WHERE environment = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, s.environment, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var errorMsg sql.NullString
		err := rows.Scan(&e.Timestamp, &e.Action, &e.Key, &e.User, &e.Success, &errorMsg)
		if err != nil {
			return nil, err
		}
		if errorMsg.Valid {
			e.ErrorMessage = errorMsg.String
		}
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// getCurrentUser returns the current OS user
func getCurrentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}
