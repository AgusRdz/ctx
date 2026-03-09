#!/bin/sh
set -e

REPO="AgusRdz/ctx"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)  OS="linux" ;;
  Darwin*) OS="darwin" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
  *) echo "unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Set install directory (Windows: AppData/Local/Programs/ctx, others: ~/bin)
if [ "$OS" = "windows" ]; then
  DEFAULT_DIR="$LOCALAPPDATA/Programs/ctx"
  EXT=".exe"
else
  DEFAULT_DIR="$HOME/bin"
  EXT=""
fi
INSTALL_DIR="${CTX_INSTALL_DIR:-$DEFAULT_DIR}"

BINARY="ctx-${OS}-${ARCH}${EXT}"

# Get latest version
if [ -z "$CTX_VERSION" ]; then
  CTX_VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')
fi

if [ -z "$CTX_VERSION" ]; then
  echo "failed to determine latest version" >&2
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${CTX_VERSION}/${BINARY}"

echo "installing ctx ${CTX_VERSION} (${OS}/${ARCH})..."

mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" -o "${INSTALL_DIR}/ctx${EXT}"
chmod +x "${INSTALL_DIR}/ctx${EXT}"

echo "installed ctx to ${INSTALL_DIR}/ctx${EXT}"
echo ""

# Check if install dir is in PATH
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    if [ "$OS" = "windows" ]; then
      # Convert to Windows path for setx
      WIN_DIR=$(cygpath -w "$INSTALL_DIR" 2>/dev/null || echo "$INSTALL_DIR")
      echo "Adding ${WIN_DIR} to your PATH..."
      powershell.exe -Command "[Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ';${WIN_DIR}', 'User')"
      echo "PATH updated. Restart your terminal for changes to take effect."
    else
      echo "NOTE: ${INSTALL_DIR} is not in your PATH."
      echo "Add it with:"
      echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
    echo ""
    ;;
esac

echo "Next steps:"
echo ""
echo "  # Register hooks in Claude Code:"
echo "  ctx init"
echo "  ctx init --status    # check if installed"
