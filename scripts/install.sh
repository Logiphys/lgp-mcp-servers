#!/usr/bin/env bash
set -euo pipefail

# LGP MCP Servers — Install Script
# Downloads and installs pre-built binaries from GitHub Releases.
# Usage: curl -sSL https://raw.githubusercontent.com/Logiphys/lgp-mcp/main/scripts/install.sh | bash

REPO="Logiphys/lgp-mcp"
SERVERS="autotask-mcp itglue-mcp datto-rmm-mcp rocketcyber-mcp"
VERSION="${LGP_VERSION:-latest}"
INSTALL_DIR="${LGP_INSTALL_DIR:-/usr/local/bin}"

# Detect OS
case "$(uname -s)" in
    Darwin) OS="darwin" ;;
    Linux)  OS="linux" ;;
    *)      echo "Unsupported OS: $(uname -s)"; exit 1 ;;
esac

# Detect architecture
case "$(uname -m)" in
    x86_64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)       echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
esac

echo "Platform: ${OS}/${ARCH}"
echo "Install directory: ${INSTALL_DIR}"

# Resolve latest version
if [ "$VERSION" = "latest" ]; then
    VERSION=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
fi
echo "Version: ${VERSION}"

# Ensure install directory exists
mkdir -p "$INSTALL_DIR"

# Download and install each server
for server in $SERVERS; do
    binary="${server}-${OS}-${ARCH}"
    url="https://github.com/${REPO}/releases/download/${VERSION}/${binary}"
    dest="${INSTALL_DIR}/${server}"

    echo "Downloading ${binary}..."
    if curl -sSL -f -o "$dest" "$url"; then
        chmod +x "$dest"
        echo "  Installed: ${dest}"
    else
        echo "  Warning: Failed to download ${binary}" >&2
    fi
done

echo ""
echo "Installation complete."
echo "Configure your MCP client with the binary paths above."
echo "See: https://github.com/${REPO}/tree/main/config for example configurations."
