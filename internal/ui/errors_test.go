package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorWithHelp_Error(t *testing.T) {
	err := ErrorWithHelp{
		Message:    "Test error message",
		Suggestion: "Try this",
		DocsLink:   "https://example.com",
		Code:       "TEST_ERROR",
	}
	
	assert.Equal(t, "Test error message", err.Error())
}

func TestNewError(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		args         []interface{}
		expectedMsg  string
		expectError  bool
	}{
		{
			name:        "ENV_NOT_FOUND error",
			code:        "ENV_NOT_FOUND",
			args:        []interface{}{"production"},
			expectedMsg: "Environment 'production' not found",
			expectError: false,
		},
		{
			name:        "VAR_NOT_FOUND error",
			code:        "VAR_NOT_FOUND",
			args:        []interface{}{"API_KEY"},
			expectedMsg: "Variable 'API_KEY' not found",
			expectError: false,
		},
		{
			name:        "AUTH_REQUIRED error",
			code:        "AUTH_REQUIRED",
			args:        []interface{}{},
			expectedMsg: "Authentication required",
			expectError: false,
		},
		{
			name:        "PERMISSION_DENIED error",
			code:        "PERMISSION_DENIED",
			args:        []interface{}{},
			expectedMsg: "Permission denied for operation",
			expectError: false,
		},
		{
			name:        "INVALID_CONFIG error",
			code:        "INVALID_CONFIG",
			args:        []interface{}{"missing field 'name'"},
			expectedMsg: "Invalid configuration: missing field 'name'",
			expectError: false,
		},
		{
			name:        "unknown error code",
			code:        "UNKNOWN_ERROR",
			args:        []interface{}{},
			expectedMsg: "unknown error: UNKNOWN_ERROR",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.code, tt.args...)
			assert.NotNil(t, err)
			assert.Equal(t, tt.expectedMsg, err.Error())
			
			if !tt.expectError {
				helpErr, ok := err.(ErrorWithHelp)
				assert.True(t, ok)
				assert.NotEmpty(t, helpErr.Suggestion)
				assert.NotEmpty(t, helpErr.DocsLink)
				assert.Equal(t, tt.code, helpErr.Code)
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedOutput []string
	}{
		{
			name: "nil error",
			err:  nil,
			expectedOutput: []string{},
		},
		{
			name: "ErrorWithHelp",
			err: ErrorWithHelp{
				Message:    "Test error",
				Suggestion: "Try this solution",
				DocsLink:   "https://docs.example.com",
				Code:       "TEST_001",
			},
			expectedOutput: []string{
				"✗ Test error",
				"💡 Suggestion:",
				"Try this solution",
				"📚 Learn more: https://docs.example.com",
				"Error code: TEST_001",
			},
		},
		{
			name: "connection refused error",
			err:  errors.New("dial tcp 127.0.0.1:8080: connection refused"),
			expectedOutput: []string{
				"✗ Unable to connect to vaultenv-cli service",
				"💡 Possible solutions:",
				"Check your internet connection",
				"Verify the service is running",
				"Check if you're behind a proxy",
				"📚 Learn more: https://docs.vaultenv-cli.io/troubleshooting",
			},
		},
		{
			name: "timeout error",
			err:  errors.New("operation timeout exceeded"),
			expectedOutput: []string{
				"✗ Operation timed out",
				"💡 This might be due to:",
				"Slow network connection",
				"Large amount of data",
				"Server under high load",
				"Try running the command again with --timeout flag",
			},
		},
		{
			name: "permission denied error",
			err:  errors.New("permission denied: cannot access resource"),
			expectedOutput: []string{
				"✗ Permission denied",
				"💡 This might mean:",
				"You need to authenticate",
				"You don't have access to this resource",
				"Your token has expired",
				"📚 Learn more: https://docs.vaultenv-cli.io/permissions",
			},
		},
		{
			name: "generic error",
			err:  errors.New("something went wrong"),
			expectedOutput: []string{
				"✗ something went wrong",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, func() {
				HandleError(tt.err)
			})
			
			output := stdout + stderr
			
			if tt.err == nil {
				assert.Empty(t, output)
			} else {
				for _, expected := range tt.expectedOutput {
					assert.Contains(t, output, expected)
				}
			}
		})
	}
}

func TestDisplayErrorWithHelp(t *testing.T) {
	tests := []struct {
		name  string
		err   ErrorWithHelp
		check func(t *testing.T, stdout, stderr string)
	}{
		{
			name: "full error with all fields",
			err: ErrorWithHelp{
				Message:    "Full error message",
				Suggestion: "Here's what to do",
				DocsLink:   "https://example.com/docs",
				Code:       "ERR_001",
			},
			check: func(t *testing.T, stdout, stderr string) {
				assert.Contains(t, stderr, "✗ Full error message")
				assert.Contains(t, stdout, "💡 Suggestion:")
				assert.Contains(t, stdout, "Here's what to do")
				assert.Contains(t, stdout, "📚 Learn more: https://example.com/docs")
				assert.Contains(t, stdout, "Error code: ERR_001")
			},
		},
		{
			name: "error without suggestion",
			err: ErrorWithHelp{
				Message:  "Error without suggestion",
				DocsLink: "https://example.com/docs",
				Code:     "ERR_002",
			},
			check: func(t *testing.T, stdout, stderr string) {
				assert.Contains(t, stderr, "✗ Error without suggestion")
				assert.NotContains(t, stdout, "💡 Suggestion:")
				assert.Contains(t, stdout, "📚 Learn more:")
				assert.Contains(t, stdout, "Error code: ERR_002")
			},
		},
		{
			name: "error without docs link",
			err: ErrorWithHelp{
				Message:    "Error without docs",
				Suggestion: "Try this",
				Code:       "ERR_003",
			},
			check: func(t *testing.T, stdout, stderr string) {
				assert.Contains(t, stderr, "✗ Error without docs")
				assert.Contains(t, stdout, "💡 Suggestion:")
				assert.NotContains(t, stdout, "📚 Learn more:")
				assert.Contains(t, stdout, "Error code: ERR_003")
			},
		},
		{
			name: "error without code",
			err: ErrorWithHelp{
				Message:    "Error without code",
				Suggestion: "Try this",
				DocsLink:   "https://example.com/docs",
			},
			check: func(t *testing.T, stdout, stderr string) {
				assert.Contains(t, stderr, "✗ Error without code")
				assert.Contains(t, stdout, "💡 Suggestion:")
				assert.Contains(t, stdout, "📚 Learn more:")
				assert.NotContains(t, stdout, "Error code:")
			},
		},
		{
			name: "minimal error",
			err: ErrorWithHelp{
				Message: "Just an error message",
			},
			check: func(t *testing.T, stdout, stderr string) {
				assert.Contains(t, stderr, "✗ Just an error message")
				assert.NotContains(t, stdout, "💡 Suggestion:")
				assert.NotContains(t, stdout, "📚 Learn more:")
				assert.NotContains(t, stdout, "Error code:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(t, func() {
				displayErrorWithHelp(tt.err)
			})
			
			tt.check(t, stdout, stderr)
		})
	}
}

func TestDisplayConnectionError(t *testing.T) {
	stdout, stderr := captureOutput(t, func() {
		displayConnectionError()
	})
	
	assert.Contains(t, stderr, "✗ Unable to connect to vaultenv-cli service")
	assert.Contains(t, stdout, "💡 Possible solutions:")
	assert.Contains(t, stdout, "Check your internet connection")
	assert.Contains(t, stdout, "Verify the service is running")
	assert.Contains(t, stdout, "Check if you're behind a proxy")
	assert.Contains(t, stdout, "📚 Learn more: https://docs.vaultenv-cli.io/troubleshooting")
}

func TestDisplayTimeoutError(t *testing.T) {
	stdout, stderr := captureOutput(t, func() {
		displayTimeoutError()
	})
	
	assert.Contains(t, stderr, "✗ Operation timed out")
	assert.Contains(t, stdout, "💡 This might be due to:")
	assert.Contains(t, stdout, "Slow network connection")
	assert.Contains(t, stdout, "Large amount of data")
	assert.Contains(t, stdout, "Server under high load")
	assert.Contains(t, stdout, "Try running the command again with --timeout flag")
}

func TestDisplayPermissionError(t *testing.T) {
	stdout, stderr := captureOutput(t, func() {
		displayPermissionError()
	})
	
	assert.Contains(t, stderr, "✗ Permission denied")
	assert.Contains(t, stdout, "💡 This might mean:")
	assert.Contains(t, stdout, "You need to authenticate")
	assert.Contains(t, stdout, "You don't have access to this resource")
	assert.Contains(t, stdout, "Your token has expired")
	assert.Contains(t, stdout, "📚 Learn more: https://docs.vaultenv-cli.io/permissions")
}

// Test error formatting edge cases
func TestErrorFormatting(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		args     []interface{}
		expected string
	}{
		{
			name:     "format with multiple placeholders",
			code:     "INVALID_CONFIG",
			args:     []interface{}{"field 'name' is missing"},
			expected: "Invalid configuration: field 'name' is missing",
		},
		{
			name:     "format with special characters",
			code:     "ENV_NOT_FOUND",
			args:     []interface{}{"test-env-123"},
			expected: "Environment 'test-env-123' not found",
		},
		{
			name:     "format with empty string",
			code:     "VAR_NOT_FOUND",
			args:     []interface{}{""},
			expected: "Variable '' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.code, tt.args...)
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

// Test that common errors are properly defined
func TestCommonErrorsCompleteness(t *testing.T) {
	expectedCodes := []string{
		"ENV_NOT_FOUND",
		"VAR_NOT_FOUND",
		"AUTH_REQUIRED",
		"PERMISSION_DENIED",
		"INVALID_CONFIG",
	}
	
	for _, code := range expectedCodes {
		t.Run(fmt.Sprintf("error_%s_exists", code), func(t *testing.T) {
			template, exists := commonErrors[code]
			assert.True(t, exists, "Error code %s should exist", code)
			assert.NotEmpty(t, template.Message)
			assert.NotEmpty(t, template.Suggestion)
			assert.NotEmpty(t, template.DocsLink)
			// Verify placeholders if any
			if strings.Contains(template.Message, "%s") || strings.Contains(template.Message, "%d") || strings.Contains(template.Message, "%v") {
				// Make sure format string is valid
				_ = fmt.Sprintf(template.Message, "test")
			}
		})
	}
}

// Benchmark error creation
func BenchmarkNewError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewError("ENV_NOT_FOUND", "production")
	}
}

func BenchmarkHandleError(b *testing.B) {
	err := errors.New("connection refused")
	var buf strings.Builder
	SetOutput(&buf, &buf)
	defer ResetOutput()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HandleError(err)
		buf.Reset()
	}
}