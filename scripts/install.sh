#!/bin/sh
set -e

REPO="dotdevlabs/clusterctl"
BINARY="clusterctl"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)          ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Resolve version (default: latest)
VERSION="${VERSION:-}"
if [ -z "$VERSION" ]; then
  VERSION=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
fi
if [ -z "$VERSION" ]; then
  echo "Could not determine latest version" >&2
  exit 1
fi

FILENAME="${BINARY}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${BINARY} ${VERSION} for ${OS}/${ARCH}..."
curl -sSfL "${BASE_URL}/${FILENAME}"   -o "${TMP_DIR}/${FILENAME}"
curl -sSfL "${BASE_URL}/checksums.txt" -o "${TMP_DIR}/checksums.txt"

# Verify checksum
cd "$TMP_DIR"
if command -v sha256sum >/dev/null 2>&1; then
  grep "${FILENAME}" checksums.txt | sha256sum -c
elif command -v shasum >/dev/null 2>&1; then
  grep "${FILENAME}" checksums.txt | shasum -a 256 -c
else
  echo "Warning: sha256sum/shasum not found, skipping checksum verification" >&2
fi

# Extract and install
tar -xzf "${FILENAME}"
chmod +x "${BINARY}"

echo "Installing ${BINARY} to ${INSTALL_DIR}/${BINARY}..."
if [ -w "$INSTALL_DIR" ]; then
  mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo ""
"${INSTALL_DIR}/${BINARY}" version
