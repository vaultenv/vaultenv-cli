package dotenv

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Parser handles various .env file formats with tolerance for common variations
type Parser struct {
	// Configuration for parser behavior
	TrimSpace      bool // Remove leading/trailing whitespace
	ExpandVars     bool // Expand ${VAR} references
	IgnoreComments bool // Skip lines starting with #
	IgnoreEmpty    bool // Skip empty lines
	IgnoreInvalid  bool // Skip malformed lines vs error
}

// NewParser creates a new parser with sensible defaults
func NewParser() *Parser {
	return &Parser{
		TrimSpace:      true,
		ExpandVars:     false, // Disabled by default for security
		IgnoreComments: true,
		IgnoreEmpty:    true,
		IgnoreInvalid:  false, // Fail fast by default
	}
}

// ParseFile reads and parses a .env file
func (p *Parser) ParseFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	return p.Parse(file)
}

// Parse reads a .env file and returns parsed variables
// This parser is intentionally forgiving because .env files in the wild
// often have inconsistent formatting
func (p *Parser) Parse(reader io.Reader) (map[string]string, error) {
	vars := make(map[string]string)
	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Handle common edge cases in .env files
		if p.shouldSkipLine(line) {
			continue
		}

		key, value, err := p.parseLine(line)
		if err != nil {
			if p.IgnoreInvalid {
				// In production, we might want to log this
				continue
			}
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		// Handle variable expansion if enabled
		if p.ExpandVars {
			value = p.expandVariables(value, vars)
		}

		vars[key] = value
	}

	return vars, scanner.Err()
}

// shouldSkipLine determines if a line should be skipped during parsing
func (p *Parser) shouldSkipLine(line string) bool {
	if p.TrimSpace {
		line = strings.TrimSpace(line)
	}

	// Skip empty lines
	if p.IgnoreEmpty && len(line) == 0 {
		return true
	}

	// Skip comments
	if p.IgnoreComments && strings.HasPrefix(line, "#") {
		return true
	}

	return false
}

// parseLine handles various .env line formats:
// KEY=value
// KEY="quoted value"
// KEY='single quoted'
// export KEY=value (shell format)
// KEY=multi\nline\nvalue
func (p *Parser) parseLine(line string) (key, value string, err error) {
	if p.TrimSpace {
		line = strings.TrimSpace(line)
	}

	// Remove 'export ' prefix if present (common in shell scripts)
	line = strings.TrimPrefix(line, "export ")

	// Find the first equals sign
	equalIndex := strings.Index(line, "=")
	if equalIndex == -1 {
		return "", "", fmt.Errorf("no '=' found in line: %s", line)
	}

	key = line[:equalIndex]
	value = line[equalIndex+1:]

	if p.TrimSpace {
		key = strings.TrimSpace(key)
	}

	// Validate key format
	if !isValidEnvVarName(key) {
		return "", "", fmt.Errorf("invalid variable name: %s", key)
	}

	// Handle quoted values
	value, err = p.unquoteValue(value)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse value for %s: %w", key, err)
	}

	return key, value, nil
}

// isValidEnvVarName checks if a string is a valid environment variable name
// Environment variable names must start with a letter or underscore
// and contain only letters, digits, and underscores
func isValidEnvVarName(name string) bool {
	if len(name) == 0 {
		return false
	}

	// Must start with letter or underscore
	if !((name[0] >= 'A' && name[0] <= 'Z') ||
		(name[0] >= 'a' && name[0] <= 'z') ||
		name[0] == '_') {
		return false
	}

	// Rest must be alphanumeric or underscore
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !((c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') ||
			c == '_') {
			return false
		}
	}

	return true
}

// unquoteValue handles quoted values in .env files
func (p *Parser) unquoteValue(value string) (string, error) {
	if p.TrimSpace {
		value = strings.TrimSpace(value)
	}

	// Handle empty value
	if len(value) == 0 {
		return "", nil
	}

	// Handle double quoted values
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		// Remove surrounding quotes
		value = value[1 : len(value)-1]
		// Process escape sequences
		return p.processEscapeSequences(value)
	}

	// Handle single quoted values (no escape sequences)
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1], nil
	}

	// Unquoted value - trim trailing whitespace but preserve leading
	return strings.TrimRightFunc(value, func(r rune) bool {
		return r == ' ' || r == '\t'
	}), nil
}

// processEscapeSequences handles escape sequences in double-quoted strings
func (p *Parser) processEscapeSequences(value string) (string, error) {
	var result strings.Builder
	i := 0

	for i < len(value) {
		if value[i] == '\\' && i+1 < len(value) {
			switch value[i+1] {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			case 'x':
				// Hex escape sequence \xHH
				if i+3 < len(value) {
					hex := value[i+2 : i+4]
					if val, err := strconv.ParseUint(hex, 16, 8); err == nil {
						result.WriteByte(byte(val))
						i += 4
						continue
					}
				}
				// Invalid hex, treat as literal
				result.WriteByte(value[i])
				result.WriteByte(value[i+1])
			default:
				// Unknown escape, treat as literal
				result.WriteByte(value[i])
				result.WriteByte(value[i+1])
			}
			i += 2
		} else {
			result.WriteByte(value[i])
			i++
		}
	}

	return result.String(), nil
}

// expandVariables expands ${VAR} and $VAR references in values
func (p *Parser) expandVariables(value string, vars map[string]string) string {
	// Pattern for ${VAR} and $VAR
	re := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

	return re.ReplaceAllStringFunc(value, func(match string) string {
		// Extract variable name
		var varName string
		if strings.HasPrefix(match, "${") {
			varName = match[2 : len(match)-1]
		} else {
			varName = match[1:]
		}

		// Look up in parsed variables first
		if val, ok := vars[varName]; ok {
			return val
		}

		// Fall back to environment variables
		if val := os.Getenv(varName); val != "" {
			return val
		}

		// Return original if not found
		return match
	})
}

// Variable represents a parsed environment variable
type Variable struct {
	Key      string
	Value    string
	LineNum  int
	Comment  string
	Original string // Original line for debugging
}

// ParseWithMetadata returns variables with additional metadata
func (p *Parser) ParseWithMetadata(reader io.Reader) ([]Variable, error) {
	var variables []Variable
	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		original := line

		if p.shouldSkipLine(line) {
			continue
		}

		key, value, err := p.parseLine(line)
		if err != nil {
			if p.IgnoreInvalid {
				continue
			}
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		// Handle variable expansion if enabled
		if p.ExpandVars {
			// For metadata parsing, we need to track which variables exist
			existingVars := make(map[string]string)
			for _, v := range variables {
				existingVars[v.Key] = v.Value
			}
			value = p.expandVariables(value, existingVars)
		}

		variables = append(variables, Variable{
			Key:      key,
			Value:    value,
			LineNum:  lineNum,
			Original: original,
		})
	}

	return variables, scanner.Err()
}

// Stats provides statistics about the parsed file
type Stats struct {
	TotalLines    int
	Variables     int
	Comments      int
	EmptyLines    int
	InvalidLines  int
	DuplicateKeys []string
}

// ParseWithStats returns variables and parsing statistics
func (p *Parser) ParseWithStats(reader io.Reader) (map[string]string, Stats, error) {
	vars := make(map[string]string)
	stats := Stats{}
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		stats.TotalLines++
		line := scanner.Text()

		// Check for empty lines
		if strings.TrimSpace(line) == "" {
			stats.EmptyLines++
			continue
		}

		// Check for comments
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			stats.Comments++
			if p.IgnoreComments {
				continue
			}
		}

		if p.shouldSkipLine(line) {
			continue
		}

		key, value, err := p.parseLine(line)
		if err != nil {
			stats.InvalidLines++
			if p.IgnoreInvalid {
				continue
			}
			return nil, stats, fmt.Errorf("line %d: %w", lineNum, err)
		}

		// Check for duplicates
		if seen[key] {
			stats.DuplicateKeys = append(stats.DuplicateKeys, key)
		}
		seen[key] = true

		// Handle variable expansion if enabled
		if p.ExpandVars {
			value = p.expandVariables(value, vars)
		}

		vars[key] = value
		stats.Variables++
	}

	return vars, stats, scanner.Err()
}
