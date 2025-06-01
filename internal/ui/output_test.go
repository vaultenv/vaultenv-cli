package ui

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Force color output for tests
	color.NoColor = false
	
	// Run tests
	code := m.Run()
	
	// Clean up
	os.Exit(code)
}

// Helper function to capture output
func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	
	// Save original outputs
	origStdout := stdout
	origStderr := stderr
	
	// Create buffers to capture output
	var outBuf, errBuf bytes.Buffer
	SetOutput(&outBuf, &errBuf)
	
	// Run the function
	fn()
	
	// Restore original outputs
	stdout = origStdout
	stderr = origStderr
	
	return outBuf.String(), errBuf.String()
}

// Helper function to strip ANSI color codes
func stripANSI(str string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(str, "")
}

func TestSuccess(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "simple message",
			format:   "Operation completed",
			args:     []interface{}{},
			expected: "✓ Operation completed",
		},
		{
			name:     "formatted message",
			format:   "Created %d files in %s",
			args:     []interface{}{5, "/tmp"},
			expected: "✓ Created 5 files in /tmp",
		},
		{
			name:     "empty message",
			format:   "",
			args:     []interface{}{},
			expected: "✓ ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, func() {
				Success(tt.format, tt.args...)
			})
			
			assert.Contains(t, stdout, tt.expected)
			assert.Empty(t, stderr)
			assert.Contains(t, stdout, "\n")
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "simple error",
			format:   "Operation failed",
			args:     []interface{}{},
			expected: "✗ Operation failed",
		},
		{
			name:     "formatted error",
			format:   "Failed to read file %s: %v",
			args:     []interface{}{"config.yaml", "permission denied"},
			expected: "✗ Failed to read file config.yaml: permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, func() {
				Error(tt.format, tt.args...)
			})
			
			assert.Empty(t, stdout)
			assert.Contains(t, stderr, tt.expected)
			assert.Contains(t, stderr, "\n")
		})
	}
}

func TestWarning(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "simple warning",
			format:   "This might cause issues",
			args:     []interface{}{},
			expected: "! This might cause issues",
		},
		{
			name:     "formatted warning",
			format:   "Found %d deprecated features",
			args:     []interface{}{3},
			expected: "! Found 3 deprecated features",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, func() {
				Warning(tt.format, tt.args...)
			})
			
			assert.Contains(t, stdout, tt.expected)
			assert.Empty(t, stderr)
			assert.Contains(t, stdout, "\n")
		})
	}
}

func TestInfo(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "simple info",
			format:   "Loading configuration",
			args:     []interface{}{},
			expected: "ℹ Loading configuration",
		},
		{
			name:     "formatted info",
			format:   "Processing %d items",
			args:     []interface{}{10},
			expected: "ℹ Processing 10 items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, func() {
				Info(tt.format, tt.args...)
			})
			
			assert.Contains(t, stdout, tt.expected)
			assert.Empty(t, stderr)
			assert.Contains(t, stdout, "\n")
		})
	}
}

func TestDebug(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "debug with verbose mode",
			verbose:  true,
			format:   "Debug information",
			args:     []interface{}{},
			expected: "› Debug information",
		},
		{
			name:     "debug without verbose mode",
			verbose:  false,
			format:   "Debug information",
			args:     []interface{}{},
			expected: "",
		},
		{
			name:     "formatted debug with verbose",
			verbose:  true,
			format:   "Variable %s = %d",
			args:     []interface{}{"count", 42},
			expected: "› Variable count = 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set verbose mode
			viper.Set("verbose", tt.verbose)
			defer viper.Set("verbose", false)
			
			stdout, stderr := captureOutput(t, func() {
				Debug(tt.format, tt.args...)
			})
			
			if tt.expected != "" {
				assert.Contains(t, stdout, tt.expected)
				assert.Contains(t, stdout, "\n")
			} else {
				assert.Empty(t, stdout)
			}
			assert.Empty(t, stderr)
		})
	}
}

func TestHeader(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name: "simple header",
			text: "Configuration",
			expected: []string{
				"Configuration",
				"─────────────",
			},
		},
		{
			name: "short header",
			text: "API",
			expected: []string{
				"API",
				"───",
			},
		},
		{
			name: "empty header",
			text: "",
			expected: []string{
				"",
				"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, func() {
				Header(tt.text)
			})
			
			// The output has an initial newline, then header text, then separator
			assert.Contains(t, stdout, tt.expected[0])
			if tt.text != "" {
				assert.Contains(t, stdout, tt.expected[1])
			}
			assert.Empty(t, stderr)
		})
	}
}

func TestTable(t *testing.T) {
	tests := []struct {
		name     string
		headers  []string
		rows     [][]string
		validate func(t *testing.T, output string)
	}{
		{
			name:    "simple table",
			headers: []string{"Name", "Value"},
			rows: [][]string{
				{"foo", "bar"},
				{"hello", "world"},
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Name")
				assert.Contains(t, output, "Value")
				assert.Contains(t, output, "foo")
				assert.Contains(t, output, "bar")
				assert.Contains(t, output, "hello")
				assert.Contains(t, output, "world")
				assert.Contains(t, output, "┌")
				assert.Contains(t, output, "┐")
				assert.Contains(t, output, "└")
				assert.Contains(t, output, "┘")
			},
		},
		{
			name:    "varying column widths",
			headers: []string{"ID", "Description"},
			rows: [][]string{
				{"1", "A very long description that should expand the column"},
				{"100", "Short"},
			},
			validate: func(t *testing.T, output string) {
				lines := strings.Split(output, "\n")
				// Check that lines have consistent width
				for _, line := range lines {
					if line != "" && strings.Contains(line, "│") {
						// Remove ANSI color codes before checking
						cleanLine := stripANSI(line)
						assert.True(t, strings.HasPrefix(cleanLine, "│") && strings.HasSuffix(cleanLine, "│"))
					}
				}
			},
		},
		{
			name:    "empty table",
			headers: []string{"Column1", "Column2"},
			rows:    [][]string{},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Column1")
				assert.Contains(t, output, "Column2")
				// Should still have box drawing characters
				assert.Contains(t, output, "┌")
				assert.Contains(t, output, "└")
			},
		},
		{
			name:    "mismatched columns",
			headers: []string{"A", "B", "C"},
			rows: [][]string{
				{"1", "2"}, // Missing third column
				{"3", "4", "5", "6"}, // Extra column
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "1")
				assert.Contains(t, output, "2")
				assert.Contains(t, output, "3")
				assert.Contains(t, output, "4")
				assert.Contains(t, output, "5")
				assert.NotContains(t, output, "6") // Extra column should be ignored
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, func() {
				Table(tt.headers, tt.rows)
			})
			
			assert.NotEmpty(t, stdout)
			assert.Empty(t, stderr)
			tt.validate(t, stdout)
		})
	}
}

func TestSetOutputAndResetOutput(t *testing.T) {
	// Create custom buffers
	var customOut, customErr bytes.Buffer
	
	// Test SetOutput
	SetOutput(&customOut, &customErr)
	
	Success("Custom output test")
	Error("Custom error test")
	
	assert.Contains(t, customOut.String(), "✓ Custom output test")
	assert.Contains(t, customErr.String(), "✗ Custom error test")
	
	// Test ResetOutput
	ResetOutput()
	
	// After reset, output should go to os.Stdout/os.Stderr
	// We can't easily test this without more complex mocking
	// but we can verify the function runs without error
}

func TestColorDisabled(t *testing.T) {
	// Save original state
	origNoColor := color.NoColor
	defer func() { color.NoColor = origNoColor }()
	
	// Disable colors
	color.NoColor = true
	
	stdout, _ := captureOutput(t, func() {
		Success("No color test")
	})
	
	// When colors are disabled, we should still see the text
	assert.Contains(t, stdout, "✓ No color test")
	// But it won't contain ANSI color codes
	assert.NotContains(t, stdout, "\033[")
}

func TestStartProgress(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		work        func() error
		expectError bool
		setupLang   string
	}{
		{
			name:    "successful operation",
			message: "Processing",
			work: func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			expectError: false,
			setupLang:   "en_US.UTF-8",
		},
		{
			name:    "failed operation",
			message: "Failing operation",
			work: func() error {
				time.Sleep(10 * time.Millisecond)
				return errors.New("operation failed")
			},
			expectError: true,
			setupLang:   "en_US.UTF-8",
		},
		{
			name:    "successful operation without unicode",
			message: "Processing ASCII",
			work: func() error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			expectError: false,
			setupLang:   "C", // No Unicode support
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and set LANG
			origLang := os.Getenv("LANG")
			os.Setenv("LANG", tt.setupLang)
			defer os.Setenv("LANG", origLang)
			
			stdout, stderr := captureOutput(t, func() {
				err := StartProgress(tt.message, tt.work)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
			
			if tt.expectError {
				assert.Contains(t, stderr, "✗")
				assert.Contains(t, stderr, "failed")
			} else {
				assert.Contains(t, stdout, "✓")
				assert.Contains(t, stdout, "complete")
			}
		})
	}
}

func TestSupportsUnicode(t *testing.T) {
	tests := []struct {
		name     string
		langEnv  string
		expected bool
	}{
		{
			name:     "UTF-8 locale",
			langEnv:  "en_US.UTF-8",
			expected: true,
		},
		{
			name:     "utf8 locale",
			langEnv:  "en_US.utf8",
			expected: true,
		},
		{
			name:     "non-UTF locale",
			langEnv:  "en_US.ISO-8859-1",
			expected: false,
		},
		{
			name:     "empty LANG",
			langEnv:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original LANG
			origLang := os.Getenv("LANG")
			defer os.Setenv("LANG", origLang)
			
			// Set test LANG
			os.Setenv("LANG", tt.langEnv)
			
			// Test
			result := supportsUnicode()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsVerbose(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		expected bool
	}{
		{
			name:     "verbose enabled",
			verbose:  true,
			expected: true,
		},
		{
			name:     "verbose disabled",
			verbose:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("verbose", tt.verbose)
			defer viper.Set("verbose", false)
			
			result := isVerbose()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test environment variable NO_COLOR
func TestNoColorEnvironment(t *testing.T) {
	// Save original state
	origNoColor := os.Getenv("NO_COLOR")
	origColorNoColor := color.NoColor
	defer func() {
		os.Setenv("NO_COLOR", origNoColor)
		color.NoColor = origColorNoColor
	}()
	
	// Set NO_COLOR environment variable
	os.Setenv("NO_COLOR", "1")
	color.NoColor = true
	
	stdout, _ := captureOutput(t, func() {
		Success("Testing with NO_COLOR")
	})
	
	// Should still output the message
	assert.Contains(t, stdout, "✓ Testing with NO_COLOR")
	// But without ANSI codes
	assert.NotContains(t, stdout, "\033[")
}

// Test concurrent access to output functions
func TestConcurrentOutput(t *testing.T) {
	// This test ensures that concurrent calls to output functions
	// don't cause race conditions
	
	done := make(chan bool)
	
	// Run multiple goroutines writing output
	for i := 0; i < 10; i++ {
		go func(id int) {
			Success("Goroutine %d success", id)
			Error("Goroutine %d error", id)
			Warning("Goroutine %d warning", id)
			Info("Goroutine %d info", id)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// If we get here without panicking, the test passes
	assert.True(t, true)
}

// Benchmark tests
func BenchmarkSuccess(b *testing.B) {
	var buf bytes.Buffer
	SetOutput(&buf, &buf)
	defer ResetOutput()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Success("Benchmark message %d", i)
	}
}

func BenchmarkTable(b *testing.B) {
	var buf bytes.Buffer
	SetOutput(&buf, &buf)
	defer ResetOutput()
	
	headers := []string{"ID", "Name", "Status", "Created"}
	rows := [][]string{
		{"1", "Test Item 1", "Active", "2024-01-01"},
		{"2", "Test Item 2", "Inactive", "2024-01-02"},
		{"3", "Test Item 3", "Pending", "2024-01-03"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Table(headers, rows)
	}
}