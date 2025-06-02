#!/bin/bash
#
# VaultEnv CLI Installation Script
# This script downloads and installs the latest version of vaultenv-cli
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default installation directory
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${1:-latest}"

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case $OS in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) echo -e "${RED}Unsupported operating system: $OS${NC}"; exit 1 ;;
    esac

    case $ARCH in
        x86_64|amd64) ARCH="x86_64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        i386|i686) ARCH="i386" ;;
        *) echo -e "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
    esac

    # Windows doesn't support arm64 in our builds
    if [ "$OS" = "windows" ] && [ "$ARCH" = "arm64" ]; then
        echo -e "${RED}Windows ARM64 is not supported${NC}"
        exit 1
    fi

    # macOS doesn't support 386
    if [ "$OS" = "darwin" ] && [ "$ARCH" = "i386" ]; then
        echo -e "${RED}macOS 32-bit is not supported${NC}"
        exit 1
    fi
}

# Get the download URL
get_download_url() {
    local BASE_URL="https://github.com/vaultenv/vaultenv-cli/releases"
    
    if [ "$VERSION" = "latest" ]; then
        # Get the latest release tag
        VERSION=$(curl -s "https://api.github.com/repos/vaultenv/vaultenv-cli/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -z "$VERSION" ]; then
            echo -e "${RED}Failed to get latest version${NC}"
            exit 1
        fi
    fi

    # Construct filename based on OS
    if [ "$OS" = "windows" ]; then
        FILENAME="vaultenv-cli_${VERSION#v}_${OS^}_${ARCH}.zip"
    else
        FILENAME="vaultenv-cli_${VERSION#v}_${OS^}_${ARCH}.tar.gz"
    fi

    DOWNLOAD_URL="${BASE_URL}/download/${VERSION}/${FILENAME}"
    echo "$DOWNLOAD_URL"
}

# Download and extract the binary
download_and_install() {
    local URL=$1
    local TEMP_DIR=$(mktemp -d)
    
    echo -e "${YELLOW}Downloading vaultenv-cli ${VERSION}...${NC}"
    
    cd "$TEMP_DIR"
    
    if ! curl -LO "$URL"; then
        echo -e "${RED}Failed to download from $URL${NC}"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    echo -e "${YELLOW}Extracting...${NC}"
    
    if [ "$OS" = "windows" ]; then
        unzip -q *.zip
    else
        tar -xzf *.tar.gz
    fi

    # Check if we need sudo for installation
    if [ -w "$INSTALL_DIR" ]; then
        SUDO=""
    else
        SUDO="sudo"
        echo -e "${YELLOW}Installation requires sudo access${NC}"
    fi

    echo -e "${YELLOW}Installing to $INSTALL_DIR...${NC}"
    
    if [ "$OS" = "windows" ]; then
        $SUDO mv vaultenv-cli.exe "$INSTALL_DIR/"
    else
        $SUDO mv vaultenv-cli "$INSTALL_DIR/"
        $SUDO chmod +x "$INSTALL_DIR/vaultenv-cli"
    fi

    # Clean up
    cd - > /dev/null
    rm -rf "$TEMP_DIR"
}

# Verify installation
verify_installation() {
    if command -v vaultenv-cli &> /dev/null; then
        echo -e "${GREEN}vaultenv-cli installed successfully!${NC}"
        echo ""
        vaultenv-cli version
    else
        echo -e "${RED}Installation failed. Please check the error messages above.${NC}"
        exit 1
    fi
}

# Main installation flow
main() {
    echo "VaultEnv CLI Installer"
    echo "===================="
    echo ""

    # Check for curl
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}Error: curl is required but not installed.${NC}"
        echo "Please install curl and try again."
        exit 1
    fi

    # Detect platform
    detect_platform
    echo -e "Detected platform: ${GREEN}${OS}/${ARCH}${NC}"

    # Get download URL
    DOWNLOAD_URL=$(get_download_url)
    echo -e "Download URL: ${GREEN}${DOWNLOAD_URL}${NC}"

    # Download and install
    download_and_install "$DOWNLOAD_URL"

    # Verify
    verify_installation

    echo ""
    echo -e "${GREEN}Installation complete!${NC}"
    echo ""
    echo "Get started with:"
    echo "  vaultenv-cli init"
    echo ""
    echo "For more information, visit:"
    echo "  https://github.com/vaultenv/vaultenv-cli"
}

# Run main function
main