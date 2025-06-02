package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/viper"
	"golang.org/x/term"
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

	// Output writers (can be overridden for testing)
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

// Success prints a success message with a checkmark
func Success(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	successColor.Fprintf(stdout, "✓ %s\n", message)
}

// Error prints an error message with an X
func Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	errorColor.Fprintf(stderr, "✗ %s\n", message)
}

// Warning prints a warning message with an exclamation
func Warning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	warningColor.Fprintf(stdout, "! %s\n", message)
}

// Info prints an informational message
func Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	infoColor.Fprintf(stdout, "ℹ %s\n", message)
}

// Debug prints debug information (only in verbose mode)
func Debug(format string, args ...interface{}) {
	if !isVerbose() {
		return
	}
	message := fmt.Sprintf(format, args...)
	mutedColor.Fprintf(stdout, "› %s\n", message)
}

// Header prints a section header
func Header(text string) {
	fmt.Fprintln(stdout)
	headerColor.Fprintln(stdout, text)
	headerColor.Fprintln(stdout, strings.Repeat("─", len(text)))
}

// StartProgress begins showing a progress indicator
func StartProgress(message string, work func() error) error {
	// Choose spinner style based on terminal capabilities
	spinnerStyle := spinner.CharSets[14] // Dots style
	if !supportsUnicode() {
		spinnerStyle = spinner.CharSets[9] // ASCII style
	}

	s := spinner.New(spinnerStyle, 100*time.Millisecond)
	s.Suffix = " " + message

	// Use color if available
	if !color.NoColor {
		s.Color("cyan", "bold")
	}

	// Start spinner
	s.Start()

	// Do the work
	err := work()

	// Stop spinner
	s.Stop()

	// Show result
	if err != nil {
		Error("%s failed: %v", message, err)
	} else {
		Success("%s complete", message)
	}

	return err
}

// Table renders data in a nice table format
func Table(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print headers
	headerColor.Fprint(stdout, "┌")
	for i, width := range widths {
		headerColor.Fprint(stdout, strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			headerColor.Fprint(stdout, "┬")
		}
	}
	headerColor.Fprintln(stdout, "┐")

	headerColor.Fprint(stdout, "│")
	for i, header := range headers {
		headerColor.Fprintf(stdout, " %-*s ", widths[i], header)
		headerColor.Fprint(stdout, "│")
	}
	headerColor.Fprintln(stdout)

	// Print separator
	headerColor.Fprint(stdout, "├")
	for i, width := range widths {
		headerColor.Fprint(stdout, strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			headerColor.Fprint(stdout, "┼")
		}
	}
	headerColor.Fprintln(stdout, "┤")

	// Print rows
	for _, row := range rows {
		fmt.Fprint(stdout, "│")
		for i, cell := range row {
			if i < len(widths) {
				fmt.Fprintf(stdout, " %-*s ", widths[i], cell)
				fmt.Fprint(stdout, "│")
			}
		}
		fmt.Fprintln(stdout)
	}

	// Print footer
	fmt.Fprint(stdout, "└")
	for i, width := range widths {
		fmt.Fprint(stdout, strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			fmt.Fprint(stdout, "┴")
		}
	}
	fmt.Fprintln(stdout, "┘")
}

// SetOutput configures the output writers (useful for testing)
func SetOutput(out, err io.Writer) {
	stdout = out
	stderr = err
}

// ResetOutput restores the default output writers
func ResetOutput() {
	stdout = os.Stdout
	stderr = os.Stderr
}

// Helper functions
func isVerbose() bool {
	return viper.GetBool("verbose")
}

func supportsUnicode() bool {
	// Check if terminal supports Unicode
	lang := os.Getenv("LANG")

	// Basic check - can be improved
	return strings.Contains(lang, "UTF-8") || strings.Contains(lang, "utf8")
}

// Select presents a list of options and returns the selected index
func Select(prompt string, options []string) int {
	fmt.Fprintln(stdout, prompt)
	for i, option := range options {
		fmt.Fprintf(stdout, "%d) %s\n", i+1, option)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(stdout, "Enter choice (1-", len(options), "): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err == nil {
			if choice >= 1 && choice <= len(options) {
				return choice - 1
			}
		}
		fmt.Fprintln(stderr, "Invalid choice. Please try again.")
	}
}

// PromptMasked prompts for input with masked characters (for passwords)
func PromptMasked(prompt string) string {
	fmt.Fprint(stdout, prompt)

	// Try to read password without echoing
	if term.IsTerminal(int(os.Stdin.Fd())) {
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(stdout) // New line after password
		if err == nil {
			return string(password)
		}
	}

	// Fallback to regular input if terminal operations fail
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// Progress shows a progress indicator with current/total
func Progress(current, total int, message string) {
	percentage := (current * 100) / total
	fmt.Fprintf(stdout, "\r[%d/%d] %d%% %s", current, total, percentage, message)
	if current == total {
		fmt.Fprintln(stdout) // New line when complete
	}
}

// Confirm prompts for a yes/no confirmation
func Confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprintf(stdout, "%s (y/n): ", prompt)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))

		switch input {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Fprintln(stderr, "Please answer 'y' or 'n'")
		}
	}
}
