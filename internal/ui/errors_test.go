package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestErrorWithHelp(t *testing.T) {
	err := ErrorWithHelp{
		Message:    "Test error message",
		Suggestion: "Try this solution",
		DocsLink:   "https://example.com/docs",
		Code:       "TEST_ERROR",
	}
	
	if err.Error() != "Test error message" {
		t.Errorf("Error() = %v, want 'Test error message'", err.Error())
	}
}

func TestNewError(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		args        []interface{}
		wantMessage string
		wantCode    string
		wantErr     bool
	}{
		{
			name:        "env_not_found",
			code:        "ENV_NOT_FOUND",
			args:        []interface{}{"production"},
			wantMessage: "Environment 'production' not found",
			wantCode:    "ENV_NOT_FOUND",
		},
		{
			name:        "var_not_found",
			code:        "VAR_NOT_FOUND",
			args:        []interface{}{"API_KEY"},
			wantMessage: "Variable 'API_KEY' not found",
			wantCode:    "VAR_NOT_FOUND",
		},
		{
			name:        "auth_required",
			code:        "AUTH_REQUIRED",
			args:        []interface{}{},
			wantMessage: "Authentication required",
			wantCode:    "AUTH_REQUIRED",
		},
		{
			name:        "invalid_config",
			code:        "INVALID_CONFIG",
			args:        []interface{}{"missing project ID"},
			wantMessage: "Invalid configuration: missing project ID",
			wantCode:    "INVALID_CONFIG",
		},
		{
			name:        "unknown_code",
			code:        "UNKNOWN_ERROR",
			args:        []interface{}{},
			wantMessage: "unknown error: UNKNOWN_ERROR",
			wantErr:     true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.code, tt.args...)
			
			if err.Error() != tt.wantMessage {
				t.Errorf("NewError() message = %v, want %v", err.Error(), tt.wantMessage)
			}
			
			if !tt.wantErr {
				if helpErr, ok := err.(ErrorWithHelp); ok {
					if helpErr.Code != tt.wantCode {
						t.Errorf("NewError() code = %v, want %v", helpErr.Code, tt.wantCode)
					}
					
					// Check that common errors have suggestions and docs
					template := commonErrors[tt.code]
					if helpErr.Suggestion != template.Suggestion {
						t.Error("NewError() missing suggestion")
					}
					if helpErr.DocsLink != template.DocsLink {
						t.Error("NewError() missing docs link")
					}
				} else {
					t.Error("NewError() did not return ErrorWithHelp")
				}
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	// Disable color for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()
	
	tests := []struct {
		name       string
		err        error
		wantOutput []string
	}{
		{
			name: "nil_error",
			err:  nil,
			wantOutput: []string{},
		},
		{
			name: "error_with_help",
			err: ErrorWithHelp{
				Message:    "Test error",
				Suggestion: "Try this",
				DocsLink:   "https://docs.example.com",
				Code:       "TEST_001",
			},
			wantOutput: []string{
				"✗ Test error",
				"💡 Suggestion:",
				"Try this",
				"📚 Learn more: https://docs.example.com",
				"Error code: TEST_001",
			},
		},
		{
			name: "connection_error",
			err:  errors.New("dial tcp: connection refused"),
			wantOutput: []string{
				"✗ Unable to connect to vaultenv-cli service",
				"💡 Possible solutions:",
				"Check your internet connection",
				"📚 Learn more: https://docs.vaultenv-cli.io/troubleshooting",
			},
		},
		{
			name: "timeout_error",
			err:  errors.New("operation timeout exceeded"),
			wantOutput: []string{
				"✗ Operation timed out",
				"💡 This might be due to:",
				"Slow network connection",
				"Try running the command again with --timeout flag",
			},
		},
		{
			name: "permission_error",
			err:  errors.New("permission denied: access forbidden"),
			wantOutput: []string{
				"✗ Permission denied",
				"💡 This might mean:",
				"You need to authenticate: vaultenv-cli auth login",
				"📚 Learn more: https://docs.vaultenv-cli.io/permissions",
			},
		},
		{
			name: "generic_error",
			err:  errors.New("something went wrong"),
			wantOutput: []string{
				"✗ something went wrong",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf, errBuf bytes.Buffer
			SetOutput(&outBuf, &errBuf)
			defer ResetOutput()
			
			HandleError(tt.err)
			
			output := outBuf.String() + errBuf.String()
			
			for _, expected := range tt.wantOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("HandleError() output missing %q\nGot: %s", expected, output)
				}
			}
		})
	}
}

func TestDisplayErrorWithHelp(t *testing.T) {
	// Disable color for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()
	
	var outBuf, errBuf bytes.Buffer
	SetOutput(&outBuf, &errBuf)
	defer ResetOutput()
	
	// Test with all fields
	err := ErrorWithHelp{
		Message:    "Full error",
		Suggestion: "Do this instead",
		DocsLink:   "https://example.com",
		Code:       "ERR_123",
	}
	
	displayErrorWithHelp(err)
	
	output := outBuf.String() + errBuf.String()
	
	// Check all components are present
	expectedParts := []string{
		"✗ Full error",
		"💡 Suggestion:",
		"Do this instead",
		"📚 Learn more: https://example.com",
		"Error code: ERR_123",
	}
	
	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("displayErrorWithHelp() missing %q", part)
		}
	}
	
	// Test with minimal fields
	outBuf.Reset()
	errBuf.Reset()
	
	minErr := ErrorWithHelp{
		Message: "Minimal error",
	}
	
	displayErrorWithHelp(minErr)
	
	output = outBuf.String() + errBuf.String()
	
	if !strings.Contains(output, "✗ Minimal error") {
		t.Error("displayErrorWithHelp() missing error message")
	}
	
	// Should not show optional fields
	if strings.Contains(output, "💡 Suggestion:") {
		t.Error("displayErrorWithHelp() showed empty suggestion")
	}
	if strings.Contains(output, "📚 Learn more:") {
		t.Error("displayErrorWithHelp() showed empty docs link")
	}
	if strings.Contains(output, "Error code:") {
		t.Error("displayErrorWithHelp() showed empty error code")
	}
}

func TestCommonErrors(t *testing.T) {
	// Verify all common errors have required fields
	requiredCodes := []string{
		"ENV_NOT_FOUND",
		"VAR_NOT_FOUND",
		"AUTH_REQUIRED",
		"PERMISSION_DENIED",
		"INVALID_CONFIG",
	}
	
	for _, code := range requiredCodes {
		err, exists := commonErrors[code]
		if !exists {
			t.Errorf("Missing common error: %s", code)
			continue
		}
		
		if err.Message == "" {
			t.Errorf("Common error %s has empty message", code)
		}
		if err.Suggestion == "" {
			t.Errorf("Common error %s has empty suggestion", code)
		}
		if err.DocsLink == "" {
			t.Errorf("Common error %s has empty docs link", code)
		}
	}
}

func TestDisplaySpecializedErrors(t *testing.T) {
	// Disable color for consistent testing
	color.NoColor = true
	defer func() { color.NoColor = false }()
	
	tests := []struct {
		name     string
		fn       func()
		expected []string
	}{
		{
			name: "connection_error",
			fn:   displayConnectionError,
			expected: []string{
				"Unable to connect to vaultenv-cli service",
				"Check your internet connection",
				"Verify the service is running",
				"Check if you're behind a proxy",
			},
		},
		{
			name: "timeout_error",
			fn:   displayTimeoutError,
			expected: []string{
				"Operation timed out",
				"Slow network connection",
				"Large amount of data",
				"Server under high load",
				"--timeout flag",
			},
		},
		{
			name: "permission_error",
			fn:   displayPermissionError,
			expected: []string{
				"Permission denied",
				"vaultenv-cli auth login",
				"don't have access",
				"token has expired",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf, errBuf bytes.Buffer
			SetOutput(&outBuf, &errBuf)
			defer ResetOutput()
			
			tt.fn()
			
			output := outBuf.String() + errBuf.String()
			
			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("%s() missing %q\nGot: %s", tt.name, expected, output)
				}
			}
		})
	}
}

func BenchmarkHandleError(b *testing.B) {
	var buf bytes.Buffer
	SetOutput(&buf, &buf)
	defer ResetOutput()
	
	err := NewError("ENV_NOT_FOUND", "production")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HandleError(err)
		buf.Reset()
	}
}

func BenchmarkNewError(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewError("VAR_NOT_FOUND", "TEST_VAR")
	}
}