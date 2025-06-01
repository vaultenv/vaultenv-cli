package cmd

import "os"

// isTestEnvironment checks if we're running in a test environment
func isTestEnvironment() bool {
    // Check if we have a test backend set
    return os.Getenv("VAULTENV_TEST") == "1"
}