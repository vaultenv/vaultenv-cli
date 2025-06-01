package cmd

import (
    "fmt"
    "runtime"

    "github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "version",
        Short: "Display version information",
        Long:  `Display detailed version information about vaultenv-cli`,
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Printf("vaultenv-cli %s\n", buildInfo.Version)
            fmt.Printf("  Commit:     %s\n", buildInfo.Commit)
            fmt.Printf("  Built:      %s\n", buildInfo.BuildTime)
            fmt.Printf("  Built by:   %s\n", buildInfo.BuiltBy)
            fmt.Printf("  Go version: %s\n", runtime.Version())
            fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
        },
    }
    return cmd
}