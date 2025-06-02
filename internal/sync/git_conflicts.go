package sync

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

// GitConflictDetector detects and helps resolve git merge conflicts
type GitConflictDetector struct {
	storage storage.Backend
	config  *config.Config
}

// NewGitConflictDetector creates a new git conflict detector
func NewGitConflictDetector(storage storage.Backend, config *config.Config) *GitConflictDetector {
	return &GitConflictDetector{
		storage: storage,
		config:  config,
	}
}

// DetectConflicts scans for git merge conflicts in vault files
func (gcd *GitConflictDetector) DetectConflicts() ([]Conflict, error) {
	var conflicts []Conflict

	// Scan .vaultenv/git directory for conflict markers
	gitDir := filepath.Join(".vaultenv", "git")
	err := filepath.Walk(gitDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-.env files
		if info.IsDir() || !strings.HasSuffix(path, ".env") {
			return nil
		}

		// Check file for conflict markers
		if hasConflictMarkers(path) {
			conflict, err := gcd.parseConflictFile(path)
			if err != nil {
				return fmt.Errorf("failed to parse conflict in %s: %w", path, err)
			}
			if conflict != nil {
				conflicts = append(conflicts, *conflict)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan for conflicts: %w", err)
	}

	return conflicts, nil
}

// hasConflictMarkers checks if a file contains git conflict markers
func hasConflictMarkers(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "<<<<<<<") ||
			strings.HasPrefix(line, "=======") ||
			strings.HasPrefix(line, ">>>>>>>") {
			return true
		}
	}

	return false
}

// parseConflictFile parses a file with git conflict markers
func (gcd *GitConflictDetector) parseConflictFile(path string) (*Conflict, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Extract environment and variable from path
	// Expected: .vaultenv/git/<environment>/<namespace>/<variable>.env
	parts := strings.Split(path, string(os.PathSeparator))
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid path structure: %s", path)
	}

	environment := parts[2]
	variable := strings.TrimSuffix(filepath.Base(path), ".env")

	// Parse conflict sections
	var (
		inOurs     bool
		inTheirs   bool
		ourLines   []string
		theirLines []string
		baseLines  []string
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "<<<<<<<") {
			inOurs = true
			continue
		} else if strings.HasPrefix(line, "=======") {
			inOurs = false
			inTheirs = true
			continue
		} else if strings.HasPrefix(line, ">>>>>>>") {
			inTheirs = false
			continue
		}

		if inOurs {
			ourLines = append(ourLines, line)
		} else if inTheirs {
			theirLines = append(theirLines, line)
		} else {
			baseLines = append(baseLines, line)
		}
	}

	// Extract values from the sections
	ourValue := extractValue(ourLines)
	theirValue := extractValue(theirLines)
	baseValue := extractValue(baseLines)

	// Create conflict object
	conflict := &Conflict{
		Environment: environment,
		Variable:    variable,
		LocalValue:  ourValue,
		RemoteValue: theirValue,
		BaseValue:   baseValue,
		LocalChange: Change{
			Author:    getCurrentUser(),
			Timestamp: time.Now(), // TODO: Get actual timestamp from git
			Action:    "set",
		},
		RemoteChange: Change{
			Author:    "remote", // TODO: Get actual author from git
			Timestamp: time.Now(),
			Action:    "set",
		},
	}

	return conflict, nil
}

// extractValue extracts the value from vault file lines
func extractValue(lines []string) string {
	for _, line := range lines {
		// Skip comments and empty lines
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Look for key=value pattern
		if idx := strings.Index(line, "="); idx > 0 {
			return strings.TrimSpace(line[idx+1:])
		}
	}
	return ""
}

// getCurrentUser gets the current git user
func getCurrentUser() string {
	// Try to get git user.name
	if name := os.Getenv("GIT_AUTHOR_NAME"); name != "" {
		return name
	}

	// Fallback to system user
	if user := os.Getenv("USER"); user != "" {
		return user
	}

	return "unknown"
}

// ResolveConflictFile writes the resolved value back to the file
func (gcd *GitConflictDetector) ResolveConflictFile(path string, value string) error {
	// Read the original file to get metadata
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Extract any metadata comments
	var metadata []string
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") && !strings.Contains(line, "<<<<") {
			metadata = append(metadata, line)
		}
	}

	// Build new content
	var newContent strings.Builder
	for _, meta := range metadata {
		newContent.WriteString(meta + "\n")
	}
	newContent.WriteString(fmt.Sprintf("value=%s\n", value))

	// Write resolved content
	if err := os.WriteFile(path, []byte(newContent.String()), 0644); err != nil {
		return fmt.Errorf("failed to write resolved file: %w", err)
	}

	return nil
}

// MarkResolved stages the resolved file in git
func (gcd *GitConflictDetector) MarkResolved(path string) error {
	// TODO: Run git add <path>
	return nil
}
