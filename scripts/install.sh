#!/usr/bin/env sh
# Reinstate installer — downloads one exact GitHub Release and verifies SHA-256.
# Public: curl -fsSL https://reinstate.dev/install.sh | sh
# Exact-tag audit: curl -fsSL https://raw.githubusercontent.com/HarjjotSinghh/reinstate/vX.Y.Z/scripts/install.sh | REINSTATE_VERSION=vX.Y.Z sh
set -eu

REPO="HarjjotSinghh/reinstate"
BIN_NAME="reinstate"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
VERSION="${REINSTATE_VERSION:-}"
CANDIDATE=""

cleanup() {
  if [ -n "${CANDIDATE}" ]; then
    rm -f "${CANDIDATE}"
  fi
  rm -rf "${TMP}"
}

confirm_replace() {
  existing_version=$1
  if [ "${REINSTATE_CONFIRM_REPLACE:-0}" = "1" ]; then
    return 0
  fi

  confirm_timeout=${REINSTATE_CONFIRM_TIMEOUT_SECONDS:-30}
  case "$confirm_timeout" in
    ''|*[!0-9]*|????*)
      echo "REINSTATE_CONFIRM_TIMEOUT_SECONDS must be an integer from 1 to 300" >&2
      return 1
      ;;
  esac
  if [ "$confirm_timeout" -lt 1 ] || [ "$confirm_timeout" -gt 300 ]; then
    echo "REINSTATE_CONFIRM_TIMEOUT_SECONDS must be an integer from 1 to 300" >&2
    return 1
  fi

  if [ -r /dev/tty ]; then
    read_timeout_status=0
    if (IFS= read -r -t 0 _reinstate_timeout_probe </dev/null) 2>/dev/null; then
      read_timeout_status=0
    else
      read_timeout_status=$?
    fi
    if [ "$read_timeout_status" -ne 1 ]; then
      echo "interactive replacement confirmation is unavailable because this shell lacks bounded reads; refusing to replace existing Reinstate ${existing_version}; set REINSTATE_CONFIRM_REPLACE=1 after reviewing the version change" >&2
      return 1
    fi

    if ! { printf 'Replace Reinstate %s with %s? [y/N] ' "${existing_version}" "${ASSET_VERSION}" >/dev/tty; } 2>/dev/null; then
      echo "refusing to replace existing Reinstate ${existing_version}; set REINSTATE_CONFIRM_REPLACE=1 after reviewing the version change" >&2
      return 1
    fi
    answer=""
    if ! IFS= read -r -t "$confirm_timeout" answer </dev/tty 2>/dev/null; then
      { printf '\n' >/dev/tty; } 2>/dev/null || true
      echo "replacement confirmation timed out after ${confirm_timeout}s; refusing to replace existing Reinstate ${existing_version}; set REINSTATE_CONFIRM_REPLACE=1 after reviewing the version change" >&2
      return 1
    fi
    case "$answer" in
      y|Y|yes|YES) return 0 ;;
    esac
  fi
  echo "refusing to replace existing Reinstate ${existing_version}; set REINSTATE_CONFIRM_REPLACE=1 after reviewing the version change" >&2
  return 1
}

json_version() {
  "$1" version --json 2>/dev/null |
    sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
    head -n 1
}

ensure_alias() {
  destination=$1
  alias_path=$2
  if [ -L "$alias_path" ]; then
    link_target=$(readlink "$alias_path" 2>/dev/null || true)
    if [ "$link_target" = "$(basename "$destination")" ]; then
      return 0
    fi
  elif [ -e "$alias_path" ]; then
    if cmp -s "$destination" "$alias_path"; then
      return 0
    fi
    echo "refusing to overwrite unrelated alias path: ${alias_path}" >&2
    return 1
  fi
  rm -f "$alias_path"
  ln -s "$(basename "$destination")" "$alias_path"
}

if [ -z "$VERSION" ]; then
  echo "REINSTATE_VERSION is required (example format: vX.Y.Z)." >&2
  echo "Refusing an unpinned latest release." >&2
  exit 1
fi
if ! printf '%s\n' "$VERSION" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'; then
  echo "REINSTATE_VERSION must be an exact v-prefixed SemVer" >&2
  exit 1
fi
ASSET_VERSION=${VERSION#v}

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac
case "$os" in
  linux|darwin) ;;
  *) echo "unsupported os: $os (use install.ps1 on native Windows)" >&2; exit 1 ;;
esac

for required in curl tar; do
  if ! command -v "$required" >/dev/null 2>&1; then
    echo "$required is required" >&2
    exit 1
  fi
done

ASSET="${BIN_NAME}_${ASSET_VERSION}_${os}_${arch}.tar.gz"
BASE="${REINSTATE_RELEASE_BASE_URL:-https://github.com/${REPO}/releases/download/${VERSION}}"
URL="${BASE}/${ASSET}"
SUM_URL="${BASE}/checksums.txt"

TMP=$(mktemp -d)
trap cleanup EXIT HUP INT TERM

echo "Downloading ${URL}..."
curl -fsSL -o "${TMP}/${ASSET}" "$URL"
echo "Downloading checksums..."
curl -fsSL -o "${TMP}/checksums.txt" "$SUM_URL"

expected=$(awk -v asset="$ASSET" '$2 == asset {print $1; exit}' "${TMP}/checksums.txt")
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
  exit 1
fi
echo "checksum ok"

mkdir -p "${TMP}/extract"
tar -xzf "${TMP}/${ASSET}" -C "${TMP}/extract"
BIN_PATH=$(find "${TMP}/extract" -type f -name "${BIN_NAME}" | head -n 1)
if [ -z "$BIN_PATH" ]; then
  echo "binary not found in archive" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
DEST="${INSTALL_DIR}/${BIN_NAME}"
ALIAS="${INSTALL_DIR}/rein"
CANDIDATE=$(mktemp "${INSTALL_DIR}/.reinstate.XXXXXX")
install -m 755 "$BIN_PATH" "$CANDIDATE"

if [ "${REINSTATE_SKIP_VERSION_CHECK:-0}" != "1" ]; then
  candidate_version=$(json_version "$CANDIDATE")
  if [ "$candidate_version" != "$ASSET_VERSION" ]; then
    echo "downloaded binary reported version ${candidate_version:-unknown}; expected ${ASSET_VERSION}" >&2
    exit 1
  fi
fi

if [ -x "$DEST" ]; then
  existing_version=$(json_version "$DEST")
  if [ "$existing_version" = "$ASSET_VERSION" ]; then
    ensure_alias "$DEST" "$ALIAS"
    echo "Reinstate ${VERSION} is already installed at ${DEST}"
    exit 0
  fi
  confirm_replace "${existing_version:-unknown}"
elif [ -e "$DEST" ]; then
  confirm_replace "unknown"
fi

mv -f "$CANDIDATE" "$DEST"
CANDIDATE=""
ensure_alias "$DEST" "$ALIAS"

echo "Installed ${BIN_NAME} ${VERSION} → ${DEST}"
echo "Installed rein alias → ${ALIAS}"
case ":${PATH:-}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "Add ${INSTALL_DIR} to PATH, then open a new terminal." ;;
esac
