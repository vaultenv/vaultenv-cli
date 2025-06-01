package ui

import (
    "fmt"
    "os"
    "strings"

    "github.com/fatih/color"
    "github.com/spf13/viper"
)

// Color scheme for consistent output
var (
    // Success is used for positive confirmations
    successColor = color.New(color.FgGreen, color.Bold)

    // Error is used for error messages
    errorColor = color.New(color.FgRed, color.Bold)

    // Warning is used for cautionary messages
    warningColor = color.New(color.FgYellow)

    // Info is used for informational messages
    infoColor = color.New(color.FgCyan)

    // Muted is used for less important information
    mutedColor = color.New(color.FgHiBlack)

    // Header is used for section headers
    headerColor = color.New(color.FgHiWhite, color.Bold)
)

// Success prints a success message with a checkmark
func Success(format string, args ...interface{}) {
    message := fmt.Sprintf(format, args...)
    successColor.Printf("✓ %s\n", message)
}

// Error prints an error message with an X
func Error(format string, args ...interface{}) {
    message := fmt.Sprintf(format, args...)
    errorColor.Fprintf(os.Stderr, "✗ %s\n", message)
}

// Warning prints a warning message with an exclamation
func Warning(format string, args ...interface{}) {
    message := fmt.Sprintf(format, args...)
    warningColor.Printf("! %s\n", message)
}

// Info prints an informational message
func Info(format string, args ...interface{}) {
    message := fmt.Sprintf(format, args...)
    infoColor.Printf("ℹ %s\n", message)
}

// Debug prints debug information (only in verbose mode)
func Debug(format string, args ...interface{}) {
    if !isVerbose() {
        return
    }
    message := fmt.Sprintf(format, args...)
    mutedColor.Printf("› %s\n", message)
}

// Header prints a section header
func Header(text string) {
    fmt.Println()
    headerColor.Println(text)
    headerColor.Println(strings.Repeat("─", len(text)))
}

// HandleError displays an error
func HandleError(err error) {
    if err == nil {
        return
    }
    Error(err.Error())
}

// Helper functions
func isVerbose() bool {
    return viper.GetBool("verbose")
}