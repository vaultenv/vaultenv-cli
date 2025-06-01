package cmd

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/AlecAivazis/survey/v2"

    "github.com/vaultenv/vaultenv-cli/internal/ui"
)

func newInitCommand() *cobra.Command {
    var (
        force bool
        name  string
    )

    cmd := &cobra.Command{
        Use:   "init",
        Short: "Initialize a new vaultenv project",
        Long: `Initialize a new vaultenv project in the current directory.

This command creates:
  • .vaultenv/config.yaml - Project configuration
  • .vaultenv/.gitignore - Git ignore rules
  • .env.example - Example environment file`,

        Example: `  # Initialize in current directory
  vaultenv-cli init

  # Initialize with project name
  vaultenv-cli init --name myproject

  # Force overwrite existing configuration
  vaultenv-cli init --force`,

        RunE: func(cmd *cobra.Command, args []string) error {
            return runInit(name, force)
        },
    }

    // Add command-specific flags
    cmd.Flags().BoolVarP(&force, "force", "f", false,
        "overwrite existing configuration")
    cmd.Flags().StringVarP(&name, "name", "n", "",
        "project name (defaults to directory name)")

    return cmd
}

func runInit(projectName string, force bool) error {
    // Get current directory
    currentDir, err := os.Getwd()
    if err != nil {
        return fmt.Errorf("failed to get current directory: %w", err)
    }

    // Default project name to directory name
    if projectName == "" {
        projectName = filepath.Base(currentDir)
    }

    // Check if already initialized
    configDir := filepath.Join(currentDir, ".vaultenv")
    configFile := filepath.Join(configDir, "config.yaml")

    if _, err := os.Stat(configFile); err == nil && !force {
        return fmt.Errorf("project already initialized. Use --force to overwrite")
    }

    // Collect project information
    var answers struct {
        ProjectName  string
        Description  string
        Environments []string
    }

    questions := []*survey.Question{
        {
            Name: "ProjectName",
            Prompt: &survey.Input{
                Message: "Project name:",
                Default: projectName,
            },
            Validate: survey.Required,
        },
        {
            Name: "Description",
            Prompt: &survey.Input{
                Message: "Project description:",
                Default: "Environment configuration for " + projectName,
            },
        },
        {
            Name: "Environments",
            Prompt: &survey.MultiSelect{
                Message: "Select environments to create:",
                Options: []string{"development", "staging", "production", "testing"},
                Default: []string{"development", "staging", "production"},
            },
        },
    }

    if err := survey.Ask(questions, &answers); err != nil {
        return err
    }

    // Create .vaultenv directory
    ui.StartProgress("Creating project structure", func() error {
        return os.MkdirAll(configDir, 0755)
    })

    // Create config file
    configContent := fmt.Sprintf(`# vaultenv Configuration
# Project: %s

project:
  name: %s
  description: %s

# Environments
environments:
`, answers.ProjectName, answers.ProjectName, answers.Description)

    for _, env := range answers.Environments {
        configContent += fmt.Sprintf("  - %s\n", env)
    }

    configContent += `
# Security settings
security:
  # Encryption algorithm (aes-gcm-256 or chacha20-poly1305)
  algorithm: aes-gcm-256
  
  # Key derivation settings
  key_derivation:
    iterations: 3
    memory: 65536  # 64MB
    threads: 4

# Storage settings
storage:
  # Backend type (file, git, or cloud)
  type: file
  
  # File storage settings
  file:
    path: .vaultenv/data

# Sync settings
sync:
  # Enable automatic sync
  enabled: false
  
  # Sync interval (in seconds)
  interval: 300
`

    err = ui.StartProgress("Creating configuration", func() error {
        return os.WriteFile(configFile, []byte(configContent), 0644)
    })
    if err != nil {
        return err
    }

    // Create .gitignore
    gitignoreContent := `# vaultenv files
*.enc
*.key
.vaultenv/data/
.vaultenv/keys/
.vaultenv/tmp/

# Local environment files
.env
.env.local
.env.*.local

# But keep examples
!.env.example
!.env.*.example
`

    err = ui.StartProgress("Creating .gitignore", func() error {
        return os.WriteFile(filepath.Join(configDir, ".gitignore"), []byte(gitignoreContent), 0644)
    })
    if err != nil {
        return err
    }

    // Create .env.example
    envExampleContent := `# Example environment configuration
# Copy this file to .env and update with your values

# Database
DATABASE_URL=postgres://user:password@localhost:5432/dbname

# API Keys
API_KEY=your-api-key-here
API_SECRET=your-api-secret-here

# Application
APP_ENV=development
APP_DEBUG=true
APP_PORT=3000

# External Services
REDIS_URL=redis://localhost:6379
SMTP_HOST=smtp.example.com
SMTP_PORT=587
`

    err = ui.StartProgress("Creating .env.example", func() error {
        return os.WriteFile(filepath.Join(currentDir, ".env.example"), []byte(envExampleContent), 0644)
    })
    if err != nil {
        return err
    }

    // Success message
    ui.Success("Project initialized successfully!")
    fmt.Println()
    ui.Info("Next steps:")
    fmt.Println("  1. Review .vaultenv/config.yaml")
    fmt.Println("  2. Copy .env.example to .env and add your variables")
    fmt.Println("  3. Run 'vaultenv-cli set KEY=VALUE' to store variables")
    fmt.Println("  4. Commit .vaultenv/config.yaml and .env.example to git")

    return nil
}