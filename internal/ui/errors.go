package ui

import (
	"fmt"
	"strings"
)

// ErrorWithHelp represents an error with helpful context
type ErrorWithHelp struct {
	Message    string
	Suggestion string
	DocsLink   string
	Code       string
}

func (e ErrorWithHelp) Error() string {
	return e.Message
}

// Common errors with helpful suggestions
var commonErrors = map[string]ErrorWithHelp{
	"ENV_NOT_FOUND": {
		Message:    "Environment '%s' not found",
		Suggestion: "Use 'vaultenv env list' to see available environments or 'vaultenv env create %s' to create it",
		DocsLink:   "https://docs.vaultenv.io/environments",
		Code:       "E001",
	},
	"VAR_NOT_FOUND": {
		Message:    "Variable '%s' not found",
		Suggestion: "Use 'vaultenv list' to see available variables or 'vaultenv set %s=value' to create it",
		DocsLink:   "https://docs.vaultenv.io/variables",
		Code:       "E002",
	},
	"AUTH_REQUIRED": {
		Message:    "Authentication required",
		Suggestion: "This environment requires a password. You'll be prompted to enter it.",
		DocsLink:   "https://docs.vaultenv.io/authentication",
		Code:       "E003",
	},
	"PERMISSION_DENIED": {
		Message:    "Permission denied for operation",
		Suggestion: "Check your access level with 'vaultenv env access list %s' or contact your team admin",
		DocsLink:   "https://docs.vaultenv.io/permissions",
		Code:       "E004",
	},
	"INVALID_CONFIG": {
		Message:    "Invalid configuration: %s",
		Suggestion: "Run 'vaultenv init' to create a valid configuration or check your .vaultenv/config.yaml file",
		DocsLink:   "https://docs.vaultenv.io/configuration",
		Code:       "E005",
	},
	"PROJECT_NOT_INITIALIZED": {
		Message:    "No VaultEnv project found in current directory",
		Suggestion: "Run 'vaultenv init' to initialize a new project or navigate to an existing project directory",
		DocsLink:   "https://docs.vaultenv.io/getting-started",
		Code:       "E006",
	},
	"ENCRYPTION_KEY_MISSING": {
		Message:    "Encryption key not found for environment '%s'",
		Suggestion: "The encryption key may have been deleted. Try 'vaultenv security verify' to diagnose the issue",
		DocsLink:   "https://docs.vaultenv.io/security",
		Code:       "E007",
	},
	"STORAGE_BACKEND_ERROR": {
		Message:    "Storage backend error: %s",
		Suggestion: "Check your storage configuration and permissions. Try 'vaultenv migrate --to file' to switch to file storage",
		DocsLink:   "https://docs.vaultenv.io/storage",
		Code:       "E008",
	},
	"INVALID_ENVIRONMENT_NAME": {
		Message:    "Invalid environment name '%s'",
		Suggestion: "Environment names can only contain letters, numbers, hyphens, and underscores",
		DocsLink:   "https://docs.vaultenv.io/environments#naming",
		Code:       "E009",
	},
	"VARIABLE_NAME_INVALID": {
		Message:    "Invalid variable name '%s'",
		Suggestion: "Variable names should follow the format KEY=VALUE. Use uppercase letters, numbers, and underscores",
		DocsLink:   "https://docs.vaultenv.io/variables#naming",
		Code:       "E010",
	},
	"FILE_NOT_FOUND": {
		Message:    "File not found: %s",
		Suggestion: "Check the file path and ensure it exists. Use absolute or relative paths from current directory",
		DocsLink:   "https://docs.vaultenv.io/import-export",
		Code:       "E011",
	},
	"HISTORY_NOT_SUPPORTED": {
		Message:    "History is not supported by the current storage backend",
		Suggestion: "Switch to SQLite storage with 'vaultenv migrate --to sqlite' to enable history tracking",
		DocsLink:   "https://docs.vaultenv.io/history",
		Code:       "E012",
	},
	"GIT_NOT_INITIALIZED": {
		Message:    "Git repository not found",
		Suggestion: "Initialize git with 'git init' and 'vaultenv git init' to enable git integration",
		DocsLink:   "https://docs.vaultenv.io/git-integration",
		Code:       "E013",
	},
	"ENVIRONMENT_LOCKED": {
		Message:    "Environment '%s' is locked",
		Suggestion: "Use 'vaultenv security unlock' to unlock the environment",
		DocsLink:   "https://docs.vaultenv.io/security#locking",
		Code:       "E014",
	},
	"PASSWORD_POLICY_VIOLATION": {
		Message:    "Password does not meet policy requirements",
		Suggestion: "Password must meet the minimum requirements. Check 'vaultenv config get security.password_policy'",
		DocsLink:   "https://docs.vaultenv.io/security#password-policies",
		Code:       "E015",
	},
}

// HandleError displays an error with helpful context
func HandleError(err error) {
	if err == nil {
		return
	}

	// Check if it's one of our errors with help
	if helpErr, ok := err.(ErrorWithHelp); ok {
		displayErrorWithHelp(helpErr)
		return
	}

	// Check for common error patterns
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "connection refused"):
		displayConnectionError()
	case strings.Contains(errStr, "timeout"):
		displayTimeoutError()
	case strings.Contains(errStr, "permission denied"):
		displayPermissionError()
	default:
		// Generic error display
		Error(err.Error())
	}
}

func displayErrorWithHelp(err ErrorWithHelp) {
	// Display the main error
	Error(err.Message)

	// Add spacing
	fmt.Fprintln(stdout)

	// Show suggestion if available
	if err.Suggestion != "" {
		infoColor.Fprintln(stdout, "💡 Suggestion:")
		fmt.Fprintf(stdout, "   %s\n", err.Suggestion)
	}

	// Show docs link if available
	if err.DocsLink != "" {
		fmt.Fprintln(stdout)
		mutedColor.Fprintf(stdout, "📚 Learn more: %s\n", err.DocsLink)
	}

	// Show error code for support
	if err.Code != "" {
		fmt.Fprintln(stdout)
		mutedColor.Fprintf(stdout, "Error code: %s\n", err.Code)
	}
}

func displayConnectionError() {
	Error("Unable to connect to vaultenv-cli service")
	fmt.Fprintln(stdout)
	infoColor.Fprintln(stdout, "💡 Possible solutions:")
	fmt.Fprintln(stdout, "   1. Check your internet connection")
	fmt.Fprintln(stdout, "   2. Verify the service is running: vaultenv-cli status")
	fmt.Fprintln(stdout, "   3. Check if you're behind a proxy")
	fmt.Fprintln(stdout)
	mutedColor.Fprintln(stdout, "📚 Learn more: https://docs.vaultenv-cli.io/troubleshooting")
}

func displayTimeoutError() {
	Error("Operation timed out")
	fmt.Fprintln(stdout)
	infoColor.Fprintln(stdout, "💡 This might be due to:")
	fmt.Fprintln(stdout, "   • Slow network connection")
	fmt.Fprintln(stdout, "   • Large amount of data")
	fmt.Fprintln(stdout, "   • Server under high load")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Try running the command again with --timeout flag")
}

func displayPermissionError() {
	Error("Permission denied")
	fmt.Fprintln(stdout)
	infoColor.Fprintln(stdout, "💡 This might mean:")
	fmt.Fprintln(stdout, "   • You need to authenticate: vaultenv-cli auth login")
	fmt.Fprintln(stdout, "   • You don't have access to this resource")
	fmt.Fprintln(stdout, "   • Your token has expired")
	fmt.Fprintln(stdout)
	mutedColor.Fprintln(stdout, "📚 Learn more: https://docs.vaultenv-cli.io/permissions")
}

// NewError creates an error with helpful context
func NewError(code string, args ...interface{}) error {
	template, exists := commonErrors[code]
	if !exists {
		return fmt.Errorf("unknown error: %s", code)
	}

	// Format message with arguments
	message := fmt.Sprintf(template.Message, args...)

	return ErrorWithHelp{
		Message:    message,
		Suggestion: template.Suggestion,
		DocsLink:   template.DocsLink,
		Code:       code,
	}
}

// PrintDiff prints a colored diff line for environment comparison
func PrintDiff(prefix, key, value string, showValues bool) {
	if showValues {
		if prefix == "+" {
			successColor.Printf("%s %s=%s\n", prefix, key, value)
		} else {
			errorColor.Printf("%s %s=%s\n", prefix, key, value)
		}
	} else {
		// Mask the value for security
		maskedValue := maskValue(value)
		if prefix == "+" {
			successColor.Printf("%s %s=%s\n", prefix, key, maskedValue)
		} else {
			errorColor.Printf("%s %s=%s\n", prefix, key, maskedValue)
		}
	}
}

// maskValue masks sensitive values for display
func maskValue(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}
