#!/bin/bash

# Test script to verify encryption support in get and list commands

set -e

echo "=== Testing VaultEnv CLI Encryption Support ==="
echo

# Build the CLI
echo "Building CLI..."
go build -o vaultenv-cli ./cmd/vaultenv-cli
export PATH=$PWD:$PATH

# Create a test directory
TEST_DIR=$(mktemp -d)
cd $TEST_DIR
echo "Working in: $TEST_DIR"
echo

# Initialize vaultenv
echo "Initializing vaultenv..."
vaultenv-cli init --name test-project
echo

# Set some encrypted variables
echo "Setting encrypted variables..."
vaultenv-cli set DB_PASSWORD=supersecret123 API_KEY=sk-test-12345 --encrypt
echo

# Test get command
echo "Testing get command with encrypted storage..."
vaultenv-cli get DB_PASSWORD
vaultenv-cli get API_KEY DB_PASSWORD
echo

# Test list command
echo "Testing list command with encrypted storage..."
vaultenv-cli list
vaultenv-cli list --values
echo

# Test pattern matching with list
echo "Testing list with pattern..."
vaultenv-cli list --pattern "API_*"
echo

# Test export format
echo "Testing export format..."
vaultenv-cli get DB_PASSWORD --export
echo

# Test quiet format
echo "Testing quiet format..."
vaultenv-cli get API_KEY --quiet
echo

# Clean up
cd /
rm -rf $TEST_DIR

echo "=== All tests passed! ==="