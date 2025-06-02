package main

import (
	"fmt"
	"os"

	"github.com/vaultenv/vaultenv-cli/internal/cmd"
)

// These variables are populated by the build process
// This allows us to display accurate version information
var (
	version   = "dev"     // Will be set by goreleaser
	commit    = "unknown" // Git commit hash for debugging
	buildTime = "unknown" // Build timestamp
	builtBy   = "unknown" // Build system identifier
)

func main() {
	// Pass version information to our command handler
	// This separation makes testing easier
	buildInfo := cmd.BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildTime: buildTime,
		BuiltBy:   builtBy,
	}

	// Execute our root command and handle any errors
	if err := cmd.Execute(buildInfo); err != nil {
		// Exit with non-zero code to indicate failure
		// This is important for CI/CD systems
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
