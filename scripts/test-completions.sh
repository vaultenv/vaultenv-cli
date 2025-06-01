#!/bin/bash
# Test script to demonstrate vaultenv-cli completions

echo "=== vaultenv-cli Shell Completion Test ==="
echo

echo "1. Testing command completions:"
echo "   Commands available: init, set, get, list, completion, version"
echo

echo "2. Testing environment completions:"
echo "   Available environments: development, staging, production, testing"
echo "   Try: vaultenv-cli set --env [TAB]"
echo

echo "3. Testing variable name completions for 'set' command:"
echo "   Common variables: DATABASE_URL, API_KEY, AWS_ACCESS_KEY_ID, etc."
echo "   Try: vaultenv-cli set D[TAB] (should suggest DATABASE_URL=)"
echo

echo "4. Testing existing variable completions for 'get' command:"
echo "   Try: vaultenv-cli get [TAB] (shows existing variables)"
echo

echo "5. Testing pattern completions for 'list' command:"
echo "   Common patterns: *, API_*, AWS_*, *_KEY, *_SECRET"
echo "   Try: vaultenv-cli list --pattern [TAB]"
echo

echo "To enable completions in your current shell:"
echo "  source <(./build/vaultenv-cli completion bash)"
echo

echo "To install permanently:"
echo "  ./build/vaultenv-cli completion bash > ~/.local/share/bash-completion/completions/vaultenv-cli"