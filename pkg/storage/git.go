package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type GitBackend struct {
	basePath    string
	environment string
}

func NewGitBackend(basePath, environment string) (*GitBackend, error) {
	envPath := filepath.Join(basePath, "git", environment)

	if err := os.MkdirAll(envPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &GitBackend{
		basePath:    basePath,
		environment: environment,
	}, nil
}

func (g *GitBackend) Set(key, value string, encrypt bool) error {
	// Validate key name for filesystem
	if err := g.validateKey(key); err != nil {
		return err
	}

	// Create file path
	filePath := g.getFilePath(key)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write value to file
	content := g.formatContent(key, value)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (g *GitBackend) Get(key string) (string, error) {
	filePath := g.getFilePath(key)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return g.parseContent(string(data))
}

func (g *GitBackend) Delete(key string) error {
	filePath := g.getFilePath(key)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			// Follow memory backend behavior - no error for non-existent keys
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Clean up empty directories
	dir := filepath.Dir(filePath)
	envPath := filepath.Join(g.basePath, "git", g.environment)

	for dir != envPath && dir != "/" && dir != "." {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}

	return nil
}

func (g *GitBackend) List() ([]string, error) {
	var keys []string

	envPath := filepath.Join(g.basePath, "git", g.environment)

	err := filepath.Walk(envPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".env") {
			// Convert file path back to key name
			relPath, _ := filepath.Rel(envPath, path)
			key := g.filePathToKey(relPath)
			keys = append(keys, key)
		}

		return nil
	})

	return keys, err
}

func (g *GitBackend) Exists(key string) (bool, error) {
	filePath := g.getFilePath(key)
	_, err := os.Stat(filePath)

	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func (g *GitBackend) Close() error {
	// No resources to cleanup
	return nil
}

// Helper methods

func (g *GitBackend) validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// Check if key starts with dot (hidden file)
	if strings.HasPrefix(key, ".") {
		return fmt.Errorf("key cannot start with dot: %s", key)
	}

	// Check for invalid characters
	for _, char := range key {
		// Only allow alphanumeric characters and underscore
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return fmt.Errorf("key contains invalid character: %c", char)
		}
	}

	return nil
}

func (g *GitBackend) getFilePath(key string) string {
	// Use subdirectories for namespacing (e.g., AWS_ACCESS_KEY -> aws/access_key.env)
	parts := strings.Split(strings.ToLower(key), "_")

	if len(parts) > 1 {
		dir := parts[0]
		name := strings.Join(parts[1:], "_")
		return filepath.Join(g.basePath, "git", g.environment, dir, name+".env")
	}

	return filepath.Join(g.basePath, "git", g.environment, strings.ToLower(key)+".env")
}

func (g *GitBackend) formatContent(key, value string) string {
	// Format for Git readability and diff-ability
	timestamp := time.Now().Format(time.RFC3339)
	return fmt.Sprintf("# Variable: %s\n# Environment: %s\n# Modified: %s\n# Generated: Do not edit directly\n\n%s\n",
		key, g.environment, timestamp, value)
}

func (g *GitBackend) parseContent(content string) (string, error) {
	lines := strings.Split(content, "\n")

	// Find the first non-comment, non-empty line after the header
	inHeader := true
	valueStartIdx := -1

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip comment lines
		if strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Found empty line after comments - this marks end of header
		if trimmedLine == "" && inHeader {
			inHeader = false
			valueStartIdx = i + 1
			break
		}
	}

	// No value section found
	if valueStartIdx == -1 || valueStartIdx >= len(lines) {
		return "", nil
	}

	// Return all lines from value start to end, joined with newlines
	valueLines := lines[valueStartIdx:]
	// Trim trailing empty lines
	for len(valueLines) > 0 && strings.TrimSpace(valueLines[len(valueLines)-1]) == "" {
		valueLines = valueLines[:len(valueLines)-1]
	}

	return strings.Join(valueLines, "\n"), nil
}

func (g *GitBackend) filePathToKey(path string) string {
	// Remove .env extension
	path = strings.TrimSuffix(path, ".env")

	// Convert path separators to underscores
	parts := strings.Split(path, string(os.PathSeparator))

	// Convert to uppercase
	key := strings.ToUpper(strings.Join(parts, "_"))

	return key
}

// Additional Git-specific methods

func (g *GitBackend) GenerateGitIgnore() string {
	return `# VaultEnv encrypted secrets
*.key
*.enc

# Local overrides
*.local
*.override

# Temporary files
*.tmp
*.bak

# Decrypted files (never commit)
*.decrypted
`
}

func (g *GitBackend) GenerateGitAttributes() string {
	return `# Treat encrypted files as binary to avoid merge conflicts
*.enc binary
*.key binary

# Use union merge for environment files
*.env merge=union
`
}
