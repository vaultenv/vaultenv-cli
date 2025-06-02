package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func newGitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git",
		Short: "Git integration commands",
		Long:  `Commands for integrating VaultEnv with Git version control.`,
	}

	cmd.AddCommand(
		newGitInitCommand(),
		newGitStatusCommand(),
		newGitDiffCommand(),
		newGitPushCommand(),
		newGitPullCommand(),
	)

	return cmd
}

func newGitInitCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Git integration",
		Long:  `Set up Git integration by creating appropriate .gitignore and .gitattributes files.`,

		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if files already exist
			gitignorePath := ".gitignore"
			gitattributesPath := ".gitattributes"

			// Check .gitignore
			gitignoreExists := false
			if _, err := os.Stat(gitignorePath); err == nil {
				gitignoreExists = true
				if !force {
					ui.Warning(".gitignore already exists. Use --force to overwrite.")
				}
			}

			// Check .gitattributes
			gitattributesExists := false
			if _, err := os.Stat(gitattributesPath); err == nil {
				gitattributesExists = true
				if !force {
					ui.Warning(".gitattributes already exists. Use --force to overwrite.")
				}
			}

			// If both exist and no force flag, exit
			if gitignoreExists && gitattributesExists && !force {
				return fmt.Errorf("git integration files already exist. Use --force to overwrite")
			}

			gitBackend := &storage.GitBackend{}

			// Create or update .gitignore
			if !gitignoreExists || force {
				gitignore := gitBackend.GenerateGitIgnore()
				if err := os.WriteFile(gitignorePath, []byte(gitignore), 0644); err != nil {
					return fmt.Errorf("failed to create .gitignore: %w", err)
				}
				ui.Success("Created .gitignore")
			} else {
				// Append to existing .gitignore
				existingContent, err := os.ReadFile(gitignorePath)
				if err != nil {
					return fmt.Errorf("failed to read .gitignore: %w", err)
				}

				// Check if VaultEnv section already exists
				if !strings.Contains(string(existingContent), "VaultEnv encrypted secrets") {
					gitignore := "\n" + gitBackend.GenerateGitIgnore()
					file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
					if err != nil {
						return fmt.Errorf("failed to open .gitignore: %w", err)
					}
					defer file.Close()

					if _, err := file.WriteString(gitignore); err != nil {
						return fmt.Errorf("failed to append to .gitignore: %w", err)
					}
					ui.Success("Updated .gitignore")
				} else {
					ui.Info(".gitignore already contains VaultEnv rules")
				}
			}

			// Create or update .gitattributes
			if !gitattributesExists || force {
				gitattributes := gitBackend.GenerateGitAttributes()
				if err := os.WriteFile(gitattributesPath, []byte(gitattributes), 0644); err != nil {
					return fmt.Errorf("failed to create .gitattributes: %w", err)
				}
				ui.Success("Created .gitattributes")
			} else {
				// Append to existing .gitattributes
				existingContent, err := os.ReadFile(gitattributesPath)
				if err != nil {
					return fmt.Errorf("failed to read .gitattributes: %w", err)
				}

				// Check if VaultEnv section already exists
				if !strings.Contains(string(existingContent), "Treat encrypted files as binary") {
					gitattributes := "\n" + gitBackend.GenerateGitAttributes()
					file, err := os.OpenFile(gitattributesPath, os.O_APPEND|os.O_WRONLY, 0644)
					if err != nil {
						return fmt.Errorf("failed to open .gitattributes: %w", err)
					}
					defer file.Close()

					if _, err := file.WriteString(gitattributes); err != nil {
						return fmt.Errorf("failed to append to .gitattributes: %w", err)
					}
					ui.Success("Updated .gitattributes")
				} else {
					ui.Info(".gitattributes already contains VaultEnv rules")
				}
			}

			ui.Success("Git integration initialized")
			ui.Info("Add .vaultenv/git/ to your repository to track encrypted secrets")

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing files")

	return cmd
}

func newGitStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Git status of VaultEnv files",
		Long:  `Display the Git status of VaultEnv configuration and secret files.`,

		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if we're in a Git repository
			if err := checkGitRepo(); err != nil {
				return err
			}

			// Run git status on .vaultenv directory
			gitCmd := exec.Command("git", "status", "--short", ".vaultenv/")
			output, err := gitCmd.CombinedOutput()
			if err != nil {
				// If .vaultenv doesn't exist in git, show general status
				gitCmd = exec.Command("git", "status", "--short")
				output, err = gitCmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to run git status: %w", err)
				}
			}

			if len(output) == 0 {
				ui.Success("No VaultEnv changes detected")
				return nil
			}

			ui.Info("VaultEnv file changes:")
			fmt.Print(string(output))

			// Count changes
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			added := 0
			modified := 0
			deleted := 0

			for _, line := range lines {
				if len(line) < 2 {
					continue
				}
				status := line[:2]
				if strings.Contains(status, "A") || strings.Contains(status, "?") {
					added++
				} else if strings.Contains(status, "M") {
					modified++
				} else if strings.Contains(status, "D") {
					deleted++
				}
			}

			// Show summary
			fmt.Println()
			ui.Info(fmt.Sprintf("Summary: %d added, %d modified, %d deleted", added, modified, deleted))

			return nil
		},
	}
}

func newGitDiffCommand() *cobra.Command {
	var staged bool

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show diffs of VaultEnv changes",
		Long:  `Display differences in VaultEnv configuration and secret files.`,

		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if we're in a Git repository
			if err := checkGitRepo(); err != nil {
				return err
			}

			// Build git diff command
			cmdArgs := []string{"diff"}
			if staged {
				cmdArgs = append(cmdArgs, "--cached")
			}
			cmdArgs = append(cmdArgs, ".vaultenv/")

			// Run git diff
			gitCmd := exec.Command("git", cmdArgs...)
			gitCmd.Stdout = os.Stdout
			gitCmd.Stderr = os.Stderr

			if err := gitCmd.Run(); err != nil {
				// If .vaultenv doesn't exist, try without the path
				cmdArgs = cmdArgs[:len(cmdArgs)-1]
				gitCmd = exec.Command("git", cmdArgs...)
				gitCmd.Stdout = os.Stdout
				gitCmd.Stderr = os.Stderr

				if err := gitCmd.Run(); err != nil {
					return fmt.Errorf("failed to run git diff: %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&staged, "staged", false, "Show staged changes")

	return cmd
}

// checkGitRepo verifies that we're in a Git repository
func checkGitRepo() error {
	gitDir := filepath.Join(".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Try running git rev-parse to check if we're in a Git repo
		cmd := exec.Command("git", "rev-parse", "--git-dir")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("not a Git repository (or any of the parent directories)")
		}
	}
	return nil
}
