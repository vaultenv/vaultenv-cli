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
        Suggestion: "Use 'vaultenv-cli env list' to see available environments",
        DocsLink:   "https://docs.vaultenv-cli.io/environments",
    },
    "VAR_NOT_FOUND": {
        Message:    "Variable '%s' not found",
        Suggestion: "Use 'vaultenv-cli list' to see available variables",
        DocsLink:   "https://docs.vaultenv-cli.io/variables",
    },
    "AUTH_REQUIRED": {
        Message:    "Authentication required",
        Suggestion: "Run 'vaultenv-cli auth login' to authenticate",
        DocsLink:   "https://docs.vaultenv-cli.io/authentication",
    },
    "PERMISSION_DENIED": {
        Message:    "Permission denied for operation",
        Suggestion: "Check your access level or contact your team admin",
        DocsLink:   "https://docs.vaultenv-cli.io/permissions",
    },
    "INVALID_CONFIG": {
        Message:    "Invalid configuration: %s",
        Suggestion: "Run 'vaultenv-cli init' to create a valid configuration",
        DocsLink:   "https://docs.vaultenv-cli.io/configuration",
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