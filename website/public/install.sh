#!/usr/bin/env sh
# Reinstate public bootstrap — pins and verifies one canonical release installer.
# Usage: curl -fsSL https://reinstate.dev/install.sh | sh
set -eu

VERSION="v0.4.0-rc.10"
PINNED_INSTALLER_SHA256="7776adb4ace8aa333745cd3f3e42b3a10d1400b9394c612d065c20a739db2e66"
INSTALLER_URL="https://raw.githubusercontent.com/HarjjotSinghh/reinstate/${VERSION}/scripts/install.sh"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"

cleanup() {
  rm -rf "$TMP"
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    echo "sha256sum or shasum is required" >&2
    return 1
  fi
}

shell_quote() {
  escaped=$(printf '%s' "$1" | sed "s/'/'\\\\''/g")
  printf "'%s'" "$escaped"
}

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

TMP=$(mktemp -d)
trap cleanup EXIT HUP INT TERM
installer_path="${TMP}/install.sh"

echo "Downloading verified Reinstate installer ${VERSION}..."
curl -fsSL -o "$installer_path" "$INSTALLER_URL"
actual_installer_sha256=$(sha256_file "$installer_path")
if [ "$actual_installer_sha256" != "$PINNED_INSTALLER_SHA256" ]; then
  echo "installer checksum mismatch" >&2
  echo "expected: ${PINNED_INSTALLER_SHA256}" >&2
  echo "actual:   ${actual_installer_sha256}" >&2
  exit 1
fi
echo "installer checksum ok"

REINSTATE_VERSION="$VERSION" \
  REINSTATE_RELEASE_BASE_URL= \
  REINSTATE_SKIP_VERSION_CHECK=0 \
  sh "$installer_path"

case ":${PATH:-}:" in
  *":${INSTALL_DIR}:"*) path_is_current=1 ;;
  *) path_is_current=0 ;;
esac

if [ "$path_is_current" = "0" ] && [ "${REINSTATE_SKIP_PATH_UPDATE:-0}" != "1" ]; then
  shell_name=${SHELL##*/}
  case "$shell_name" in
    zsh) profile="${HOME}/.zshrc" ;;
    bash) profile="${HOME}/.bashrc" ;;
    *) profile="${HOME}/.profile" ;;
  esac

  quoted_install_dir=$(shell_quote "$INSTALL_DIR")
  path_line="export PATH=${quoted_install_dir}:\$PATH"
  if [ ! -f "$profile" ] || ! grep -Fqx "$path_line" "$profile"; then
    {
      printf '\n# Reinstate CLI\n'
      printf '%s\n' "$path_line"
    } >>"$profile"
    echo "Added ${INSTALL_DIR} to PATH in ${profile}."
  fi
fi

quoted_rein=$(shell_quote "${INSTALL_DIR}/rein")
printf 'Next: %s init\n' "$quoted_rein"
if [ "$path_is_current" = "0" ]; then
  echo "New terminals: rein init"
fi
