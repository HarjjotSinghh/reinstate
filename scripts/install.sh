#!/usr/bin/env sh
# Reinstate installer — downloads the latest GitHub Release binary.
# Usage: curl -fsSL https://raw.githubusercontent.com/HarjjotSinghh/reinstate/main/scripts/install.sh | sh
set -eu

REPO="HarjjotSinghh/reinstate"
BIN_NAME="reinstate"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

case "$os" in
  linux|darwin) ;;
  msys*|mingw*|cygwin*) os="windows" ;;
  *) echo "unsupported os: $os" >&2; exit 1 ;;
esac

echo "Detecting latest release for ${os}/${arch}..."
if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

API="https://api.github.com/repos/${REPO}/releases/latest"
TAG=$(curl -fsSL "$API" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)
if [ -z "$TAG" ]; then
  echo "No GitHub release found yet. Build from source:" >&2
  echo "  git clone https://github.com/${REPO}.git && cd reinstate && make build" >&2
  exit 1
fi

EXT=""
if [ "$os" = "windows" ]; then EXT=".exe"; fi
ASSET="${BIN_NAME}_${TAG}_${os}_${arch}${EXT}"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${URL}..."
if ! curl -fsSL -o "${TMP}/${BIN_NAME}${EXT}" "$URL"; then
  echo "Download failed. Release asset may not exist for this platform yet." >&2
  exit 1
fi

chmod +x "${TMP}/${BIN_NAME}${EXT}"

if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP}/${BIN_NAME}${EXT}" "${INSTALL_DIR}/${BIN_NAME}${EXT}"
else
  echo "Installing to ${INSTALL_DIR} (may require sudo)..."
  sudo mv "${TMP}/${BIN_NAME}${EXT}" "${INSTALL_DIR}/${BIN_NAME}${EXT}"
fi

echo "Installed ${BIN_NAME} ${TAG} → ${INSTALL_DIR}/${BIN_NAME}${EXT}"
"${INSTALL_DIR}/${BIN_NAME}${EXT}" version || true
