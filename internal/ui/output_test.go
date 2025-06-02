package ui

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

func TestOutput_Messages(t *testing.T) {
	// Disable color for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name     string
		fn       func(string, ...interface{})
		format   string
		args     []interface{}
		wantOut  string
		wantErr  string
		isStderr bool
	}{
		{
			name:    "success",
			fn:      Success,
			format:  "Operation %s",
			args:    []interface{}{"completed"},
			wantOut: "✓ Operation completed\n",
		},
		{
			name:     "error",
			fn:       Error,
			format:   "Operation %s",
			args:     []interface{}{"failed"},
			wantErr:  "✗ Operation failed\n",
			isStderr: true,
		},
		{
			name:    "warning",
			fn:      Warning,
			format:  "Be careful about %s",
			args:    []interface{}{"this"},
			wantOut: "! Be careful about this\n",
		},
		{
			name:    "info",
			fn:      Info,
			format:  "FYI: %s",
			args:    []interface{}{"information"},
			wantOut: "ℹ FYI: information\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf, errBuf bytes.Buffer
			SetOutput(&outBuf, &errBuf)
			defer ResetOutput()

			tt.fn(tt.format, tt.args...)

			if tt.isStderr {
				if got := errBuf.String(); got != tt.wantErr {
					t.Errorf("stderr = %q, want %q", got, tt.wantErr)
				}
			} else {
				if got := outBuf.String(); got != tt.wantOut {
					t.Errorf("stdout = %q, want %q", got, tt.wantOut)
				}
			}
		})
	}
}

func TestDebug(t *testing.T) {
	// Disable color for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var outBuf bytes.Buffer
	SetOutput(&outBuf, nil)
	defer ResetOutput()

	// Test debug not shown when verbose is false
	viper.Set("verbose", false)
	Debug("Debug message")

	if outBuf.Len() != 0 {
		t.Error("Debug() should not output when verbose is false")
	}

	// Test debug shown when verbose is true
	viper.Set("verbose", true)
	defer viper.Set("verbose", false)

	Debug("Debug %s", "message")

	want := "› Debug message\n"
	if got := outBuf.String(); got != want {
		t.Errorf("Debug() = %q, want %q", got, want)
	}
}

func TestHeader(t *testing.T) {
	// Disable color for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var outBuf bytes.Buffer
	SetOutput(&outBuf, nil)
	defer ResetOutput()

	Header("Test Section")

	output := outBuf.String()
	lines := strings.Split(output, "\n")

	// Header should produce 4 lines: empty line, header text, separator, and final newline
	if len(lines) != 4 {
		t.Errorf("Header() produced %d lines, want 4", len(lines))
	}

	if lines[0] != "" {
		t.Error("Header() should start with empty line")
	}

	if lines[1] != "Test Section" {
		t.Errorf("Header() text = %q, want 'Test Section'", lines[1])
	}

	if lines[2] != strings.Repeat("─", len("Test Section")) {
		t.Error("Header() separator length doesn't match text")
	}
}

func TestTable(t *testing.T) {
	// Disable color for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var outBuf bytes.Buffer
	SetOutput(&outBuf, nil)
	defer ResetOutput()

	headers := []string{"Name", "Value", "Type"}
	rows := [][]string{
		{"KEY1", "value1", "string"},
		{"LONG_KEY_NAME", "short", "string"},
		{"K", "very long value here", "text"},
	}

	Table(headers, rows)

	output := outBuf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have: top border, header, separator, 3 rows, bottom border = 7 lines
	if len(lines) != 7 {
		t.Errorf("Table() produced %d lines, want 7", len(lines))
	}

	// Check that output contains expected content
	if !strings.Contains(output, "Name") || !strings.Contains(output, "Value") || !strings.Contains(output, "Type") {
		t.Error("Table() missing headers")
	}

	if !strings.Contains(output, "KEY1") || !strings.Contains(output, "LONG_KEY_NAME") {
		t.Error("Table() missing row data")
	}

	// Check borders
	if !strings.HasPrefix(lines[0], "┌") {
		t.Error("Table() missing top border")
	}

	if !strings.HasPrefix(lines[len(lines)-1], "└") {
		t.Error("Table() missing bottom border")
	}
}

func TestStartProgress(t *testing.T) {
	// Disable color for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var outBuf, errBuf bytes.Buffer
	SetOutput(&outBuf, &errBuf)
	defer ResetOutput()

	// Test successful operation
	err := StartProgress("Testing operation", func() error {
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("StartProgress() error = %v", err)
	}

	output := outBuf.String()
	if !strings.Contains(output, "✓ Testing operation complete") {
		t.Error("StartProgress() missing success message")
	}

	// Test failed operation
	outBuf.Reset()
	errBuf.Reset()

	testErr := fmt.Errorf("test error")
	err = StartProgress("Failing operation", func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("StartProgress() error = %v, want %v", err, testErr)
	}

	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "✗ Failing operation failed: test error") {
		t.Error("StartProgress() missing error message")
	}
}

func TestProgress(t *testing.T) {
	var outBuf bytes.Buffer
	SetOutput(&outBuf, nil)
	defer ResetOutput()

	// Test progress updates
	Progress(25, 100, "Processing files")
	output := outBuf.String()

	if !strings.Contains(output, "[25/100]") {
		t.Error("Progress() missing count")
	}

	if !strings.Contains(output, "25%") {
		t.Error("Progress() missing percentage")
	}

	if !strings.Contains(output, "Processing files") {
		t.Error("Progress() missing message")
	}

	// Test completion adds newline
	outBuf.Reset()
	Progress(100, 100, "Complete")
	output = outBuf.String()

	if !strings.HasSuffix(output, "\n") {
		t.Error("Progress() should add newline when complete")
	}
}

func TestSupportsUnicode(t *testing.T) {
	tests := []struct {
		name    string
		langEnv string
		want    bool
	}{
		{"utf8_lowercase", "en_US.utf8", true},
		{"utf8_uppercase", "en_US.UTF-8", true},
		{"no_utf8", "en_US", false},
		{"empty", "", false},
		{"c_locale", "C", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original LANG
			origLang := os.Getenv("LANG")
			defer os.Setenv("LANG", origLang)

			os.Setenv("LANG", tt.langEnv)

			if got := supportsUnicode(); got != tt.want {
				t.Errorf("supportsUnicode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetOutput(t *testing.T) {
	// Create custom buffers
	var customOut, customErr bytes.Buffer

	// Set custom output
	SetOutput(&customOut, &customErr)

	// Test that output goes to custom buffers
	Success("test stdout")
	Error("test stderr")

	if !strings.Contains(customOut.String(), "test stdout") {
		t.Error("SetOutput() stdout not redirected")
	}

	if !strings.Contains(customErr.String(), "test stderr") {
		t.Error("SetOutput() stderr not redirected")
	}

	// Reset and verify
	ResetOutput()

	// After reset, output should go to os.Stdout/os.Stderr
	// (Can't easily test this without capturing os.Stdout/os.Stderr)
}

func TestTableEdgeCases(t *testing.T) {
	// Disable color for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var outBuf bytes.Buffer
	SetOutput(&outBuf, nil)
	defer ResetOutput()

	// Test empty table
	Table([]string{"Col1", "Col2"}, [][]string{})
	output := outBuf.String()

	// Should still have borders and headers
	if !strings.Contains(output, "Col1") || !strings.Contains(output, "Col2") {
		t.Error("Table() with no rows should still show headers")
	}

	// Test mismatched row lengths
	outBuf.Reset()
	headers := []string{"A", "B", "C"}
	rows := [][]string{
		{"1", "2"},           // Too short
		{"1", "2", "3", "4"}, // Too long
		{"1", "2", "3"},      // Just right
	}

	Table(headers, rows)
	output = outBuf.String()

	// Should handle gracefully without panicking
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		t.Error("Table() produced no output with mismatched rows")
	}
}

func TestIsVerbose(t *testing.T) {
	// Test with verbose false
	viper.Set("verbose", false)
	if isVerbose() {
		t.Error("isVerbose() = true, want false")
	}

	// Test with verbose true
	viper.Set("verbose", true)
	if !isVerbose() {
		t.Error("isVerbose() = false, want true")
	}

	// Reset
	viper.Set("verbose", false)
}

func BenchmarkSuccess(b *testing.B) {
	var buf bytes.Buffer
	SetOutput(&buf, nil)
	defer ResetOutput()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Success("Operation completed successfully")
		buf.Reset()
	}
}

func BenchmarkTable(b *testing.B) {
	var buf bytes.Buffer
	SetOutput(&buf, nil)
	defer ResetOutput()

	headers := []string{"Key", "Value", "Type", "Modified"}
	rows := make([][]string, 20)
	for i := range rows {
		rows[i] = []string{
			fmt.Sprintf("KEY_%d", i),
			fmt.Sprintf("value_%d", i),
			"string",
			"2024-01-01",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Table(headers, rows)
		buf.Reset()
	}
}
