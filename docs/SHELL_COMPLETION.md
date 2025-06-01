# Shell Completion Guide for vaultenv-cli

vaultenv-cli provides intelligent shell completions for all major shells (bash, zsh, fish, powershell).

## Features

### 1. Command Completion
All commands and subcommands are automatically completed:
- `init` - Initialize a new project
- `set` - Set environment variables
- `get` - Get environment variables
- `list` - List environment variables
- `completion` - Generate completion scripts
- `version` - Show version information

### 2. Flag Completion
All flags are automatically completed with descriptions:
- `--env` / `-e` - Environment name
- `--force` / `-f` - Force operations
- `--quiet` / `-q` - Quiet output
- `--values` - Show values in list
- `--pattern` / `-p` - Filter pattern

### 3. Dynamic Completions

#### Environment Names
The `--env` flag provides completion for environment names:
- development
- staging
- production
- testing

#### Variable Names
Different commands provide context-aware variable name completions:

**For `set` command:**
- Suggests common variable names with `=` appended
- Examples: `DATABASE_URL=`, `API_KEY=`, `AWS_ACCESS_KEY_ID=`

**For `get` command:**
- Shows existing variables from the current environment
- Dynamically queries the storage backend

#### Pattern Completions
The `list --pattern` flag suggests common patterns:
- `*` - All variables
- `API_*` - All API-related variables
- `AWS_*` - All AWS-related variables
- `*_KEY` - All variables ending with KEY
- `*_SECRET` - All variables ending with SECRET

## Installation

### Bash
```bash
# Load in current session
source <(vaultenv-cli completion bash)

# Install permanently (Linux)
vaultenv-cli completion bash > /etc/bash_completion.d/vaultenv-cli

# Install permanently (macOS with Homebrew)
vaultenv-cli completion bash > $(brew --prefix)/etc/bash_completion.d/vaultenv-cli

# Install permanently (user-specific)
vaultenv-cli completion bash > ~/.local/share/bash-completion/completions/vaultenv-cli
```

### Zsh
```zsh
# Load in current session
source <(vaultenv-cli completion zsh)

# Install permanently
vaultenv-cli completion zsh > "${fpath[1]}/_vaultenv-cli"

# You may need to rebuild zcompdump
rm -f ~/.zcompdump && compinit
```

### Fish
```fish
# Load in current session
vaultenv-cli completion fish | source

# Install permanently
vaultenv-cli completion fish > ~/.config/fish/completions/vaultenv-cli.fish
```

### PowerShell
```powershell
# Load in current session
vaultenv-cli completion powershell | Out-String | Invoke-Expression

# Install permanently
vaultenv-cli completion powershell >> $PROFILE
```

## Usage Examples

### Example 1: Setting Variables
```bash
$ vaultenv-cli set D<TAB>
DATABASE_URL=  DEBUG=

$ vaultenv-cli set DATABASE_URL=postgres://localhost --env <TAB>
development  production  staging  testing
```

### Example 2: Getting Variables
```bash
$ vaultenv-cli get <TAB>
# Shows all existing variables in current environment

$ vaultenv-cli get API_<TAB>
API_KEY  API_SECRET  API_URL
```

### Example 3: Listing with Patterns
```bash
$ vaultenv-cli list --pattern <TAB>
*  API_*  AWS_*  DATABASE_*  REDIS_*  SMTP_*  *_KEY  *_SECRET  *_URL  *_TOKEN
```

## Advanced Features

### Case-Insensitive Matching
Variable name completions are case-insensitive for convenience:
```bash
$ vaultenv-cli get api<TAB>
API_KEY  API_SECRET
```

### Context-Aware Completions
The `get` command provides completions based on existing variables in the selected environment:
```bash
$ vaultenv-cli get --env production <TAB>
# Shows only variables that exist in production environment
```

## Troubleshooting

### Completions Not Working

1. **Check shell compatibility:**
   ```bash
   echo $SHELL
   ```

2. **Verify completion is loaded:**
   ```bash
   # For bash
   complete -p vaultenv-cli
   ```

3. **Reload your shell:**
   ```bash
   exec $SHELL
   ```

### Slow Completions
If completions for existing variables are slow, it may be due to storage backend latency. The completion system includes timeouts to prevent hanging.

## Development

To add new completion functions, edit `internal/cmd/completion.go` and implement a function with this signature:

```go
func myCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    // Return matching strings and directive
    return matches, cobra.ShellCompDirectiveNoFileComp
}
```

Then register it in the appropriate command:
```go
cmd.RegisterFlagCompletionFunc("flag-name", myCompletion)
// or for positional args:
cmd.ValidArgsFunction = myCompletion
```