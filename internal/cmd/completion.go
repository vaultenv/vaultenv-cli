package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func newCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for vaultenv-cli.

To load completions in your current shell session:

  Bash:
    $ source <(vaultenv-cli completion bash)

  Zsh:
    $ source <(vaultenv-cli completion zsh)

  Fish:
    $ vaultenv-cli completion fish | source

  PowerShell:
    PS> vaultenv-cli completion powershell | Out-String | Invoke-Expression

To load completions for every new session, execute once:

  Bash:
    $ vaultenv-cli completion bash > /etc/bash_completion.d/vaultenv-cli

  Zsh:
    $ vaultenv-cli completion zsh > "${fpath[1]}/_vaultenv-cli"

  Fish:
    $ vaultenv-cli completion fish > ~/.config/fish/completions/vaultenv-cli.fish

  PowerShell:
    PS> vaultenv-cli completion powershell > $PROFILE`,

		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		RunE:                  runCompletion,
	}

	return cmd
}

func runCompletion(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return fmt.Errorf("unsupported shell: %s", args[0])
	}
}

// environmentCompletion provides shell completion for environment names
func environmentCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// In a real implementation, this would fetch from config
	environments := []string{"development", "staging", "production", "testing"}

	var matches []string
	for _, env := range environments {
		if strings.HasPrefix(env, toComplete) {
			matches = append(matches, env)
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}

// variableNameCompletion provides shell completion for variable names
func variableNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Common environment variable patterns
	suggestions := []string{
		"DATABASE_URL",
		"API_KEY",
		"API_SECRET",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_REGION",
		"REDIS_URL",
		"REDIS_PASSWORD",
		"JWT_SECRET",
		"JWT_EXPIRY",
		"SMTP_HOST",
		"SMTP_PORT",
		"SMTP_USER",
		"SMTP_PASSWORD",
		"PORT",
		"HOST",
		"NODE_ENV",
		"LOG_LEVEL",
		"DEBUG",
		"SENTRY_DSN",
		"STRIPE_API_KEY",
		"STRIPE_SECRET_KEY",
		"GITHUB_TOKEN",
		"GITLAB_TOKEN",
	}

	var matches []string
	for _, suggestion := range suggestions {
		if strings.HasPrefix(strings.ToUpper(suggestion), strings.ToUpper(toComplete)) {
			matches = append(matches, suggestion)
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}

// existingVariableCompletion provides completion for existing variable names
func existingVariableCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Get environment from flag
	env, _ := cmd.Flags().GetString("env")
	if env == "" {
		env = "development"
	}

	// Get storage backend
	store, err := storage.GetBackend(env)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer store.Close()

	// Get all variable names
	keys, err := store.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var matches []string
	for _, key := range keys {
		if strings.HasPrefix(strings.ToUpper(key), strings.ToUpper(toComplete)) {
			matches = append(matches, key)
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}

// patternCompletion provides completion for pattern flags
func patternCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	patterns := []string{
		"*",
		"API_*",
		"AWS_*",
		"DATABASE_*",
		"REDIS_*",
		"SMTP_*",
		"*_KEY",
		"*_SECRET",
		"*_URL",
		"*_TOKEN",
	}

	var matches []string
	for _, pattern := range patterns {
		if strings.HasPrefix(pattern, toComplete) {
			matches = append(matches, pattern)
		}
	}

	return matches, cobra.ShellCompDirectiveNoFileComp
}
