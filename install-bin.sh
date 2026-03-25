#!/usr/bin/env bash
set -euo pipefail

# gwx binary installer — downloads pre-built binary to /usr/local/bin
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/redredchen01/gwx/main/install-bin.sh | sudo bash
#   curl -fsSL ... | bash -s -- --dir ~/.local/bin   (no sudo, custom dir)

VERSION="${GWX_VERSION:-latest}"
INSTALL_DIR="${1:-/usr/local/bin}"
REPO="redredchen01/gwx"

# Parse --dir flag
for arg in "$@"; do
    case "$arg" in
        --dir=*) INSTALL_DIR="${arg#*=}" ;;
        --dir)   shift; INSTALL_DIR="${1:-/usr/local/bin}" ;;
    esac
done

# Detect OS and arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    darwin|linux) ;;
    *)            echo "Unsupported OS: $OS (use npm or go install instead)"; exit 1 ;;
esac

# Resolve version
if [ "$VERSION" = "latest" ]; then
    VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')"
    if [ -z "$VERSION" ]; then
        echo "Failed to fetch latest version"
        exit 1
    fi
fi

BINARY="gwx_${VERSION#v}_${OS}_${ARCH}"
URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY"

echo ""
echo "  gwx installer"
echo "  Version:  $VERSION"
echo "  OS/Arch:  ${OS}/${ARCH}"
echo "  Target:   ${INSTALL_DIR}/gwx"
echo ""

# Download
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "→ Downloading $URL ..."
if ! curl -fsSL -o "$TMP/gwx" "$URL"; then
    echo "✗ Download failed. Check version and platform."
    echo "  Available at: https://github.com/$REPO/releases"
    exit 1
fi

chmod +x "$TMP/gwx"

# Verify binary runs
if ! "$TMP/gwx" version >/dev/null 2>&1; then
    echo "✗ Downloaded binary failed to execute"
    exit 1
fi

# Install
mkdir -p "$INSTALL_DIR"
mv "$TMP/gwx" "$INSTALL_DIR/gwx"

INSTALLED_VERSION="$("$INSTALL_DIR/gwx" version --format plain 2>/dev/null || echo "$VERSION")"
echo "✓ Installed gwx $INSTALLED_VERSION to $INSTALL_DIR/gwx"

# Check PATH
if ! command -v gwx &>/dev/null; then
    echo ""
    echo "⚠ gwx is not in your PATH. Add this to your shell profile:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo ""
echo "Next: gwx onboard"
echo ""
