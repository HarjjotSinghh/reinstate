#!/bin/sh
set -eu

dist_dir=${1:-dist}
expected_commit=${2:-}
expected_version=${3:-}

case "$expected_commit" in
	????????????????????????????????????????) ;;
	*)
		echo "expected commit must be a full 40-character git SHA" >&2
		exit 1
		;;
esac
case "$expected_commit" in
	*[!0-9a-fA-F]*)
		echo "expected commit must contain only hexadecimal characters" >&2
		exit 1
		;;
esac
test -n "$expected_version"

case "$(uname -s)" in
	Darwin) host_os=darwin ;;
	Linux) host_os=linux ;;
	*)
		echo "unsupported release-validation host: $(uname -s)" >&2
		exit 1
		;;
esac
case "$(uname -m)" in
	x86_64|amd64) host_arch=amd64 ;;
	arm64|aarch64) host_arch=arm64 ;;
	*)
		echo "unsupported release-validation architecture: $(uname -m)" >&2
		exit 1
		;;
esac

archive="$dist_dir/reinstate_${expected_version}_${host_os}_${host_arch}.tar.gz"
test -f "$archive"

identity_dir=$(mktemp -d)
trap 'rm -rf "$identity_dir"' EXIT HUP INT TERM
tar -xzf "$archive" -C "$identity_dir"

binary="$identity_dir/reinstate"
test -x "$binary"
identity_json=$($binary version --json)

actual_commit=$(printf '%s\n' "$identity_json" | sed -n 's/.*"commit"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
actual_version=$(printf '%s\n' "$identity_json" | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')

if test "$actual_commit" != "$expected_commit"; then
	echo "release binary commit mismatch: expected $expected_commit, got $actual_commit" >&2
	exit 1
fi
if test "$actual_version" != "$expected_version"; then
	echo "release binary version mismatch: expected $expected_version, got $actual_version" >&2
	exit 1
fi

printf 'release binary identity ok: version=%s commit=%s\n' "$actual_version" "$actual_commit"
