#!/usr/bin/env sh
# Reinstate installer — downloads a GitHub Release binary and verifies SHA-256.
# Usage: curl -fsSL https://raw.githubusercontent.com/HarjjotSinghh/reinstate/main/scripts/install.sh | sh
set -eu

REPO="HarjjotSinghh/reinstate"
BIN_NAME="reinstate"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
VERSION="${REINSTATE_VERSION:-}"

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

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

if [ -z "$VERSION" ]; then
  API="https://api.github.com/repos/${REPO}/releases/latest"
  VERSION=$(curl -fsSL "$API" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)
fi
if [ -z "$VERSION" ]; then
  echo "No GitHub release found yet. Build from source:" >&2
  echo "  git clone https://github.com/${REPO}.git && cd reinstate && make build" >&2
  exit 1
fi

EXT=""
ARCHIVE_EXT="tar.gz"
if [ "$os" = "windows" ]; then
  EXT=".exe"
  ARCHIVE_EXT="zip"
fi

ASSET="${BIN_NAME}_${VERSION}_${os}_${arch}.${ARCHIVE_EXT}"
BASE="https://github.com/${REPO}/releases/download/${VERSION}"
URL="${BASE}/${ASSET}"
SUM_URL="${BASE}/checksums.txt"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${URL}..."
curl -fsSL -o "${TMP}/${ASSET}" "$URL"
echo "Downloading checksums..."
curl -fsSL -o "${TMP}/checksums.txt" "$SUM_URL"

# Verify checksum
expected=$(grep -E "[[:space:]]${ASSET}$" "${TMP}/checksums.txt" | awk '{print $1}' | head -n1)
if [ -z "$expected" ]; then
  echo "checksum entry missing for ${ASSET}" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "${TMP}/${ASSET}" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "${TMP}/${ASSET}" | awk '{print $1}')
else
  echo "sha256sum or shasum required" >&2
  exit 1
fi
if [ "$expected" != "$actual" ]; then
  echo "checksum mismatch for ${ASSET}" >&2
  echo "expected ${expected}" >&2
  echo "actual   ${actual}" >&2
  exit 1
fi
echo "checksum ok"

# Extract binary
mkdir -p "${TMP}/extract"
case "$ARCHIVE_EXT" in
  tar.gz) tar -xzf "${TMP}/${ASSET}" -C "${TMP}/extract" ;;
  zip)
    if command -v unzip >/dev/null 2>&1; then
      unzip -q "${TMP}/${ASSET}" -d "${TMP}/extract"
    else
      echo "unzip required for windows archives" >&2
      exit 1
    fi
    ;;
esac

BIN_PATH=$(find "${TMP}/extract" -type f -name "${BIN_NAME}${EXT}" | head -n1)
if [ -z "$BIN_PATH" ]; then
  echo "binary not found in archive" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
install -m 755 "$BIN_PATH" "${INSTALL_DIR}/${BIN_NAME}${EXT}"
# optional short alias
ln -sfn "${BIN_NAME}${EXT}" "${INSTALL_DIR}/rein${EXT}" 2>/dev/null || true

echo "Installed ${BIN_NAME} ${VERSION} → ${INSTALL_DIR}/${BIN_NAME}${EXT}"
"${INSTALL_DIR}/${BIN_NAME}${EXT}" version || true
