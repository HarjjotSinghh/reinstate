#!/bin/sh
set -eu

usage() {
  echo "usage: $0 vX.Y.Z[-prerelease] [base-url]" >&2
  exit 2
}

[ "$#" -ge 1 ] && [ "$#" -le 2 ] || usage
version=$1
base_url=${2:-https://reinstate.dev}

case "$version" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) usage ;;
esac

base_url=${base_url%/}
temporary_directory=$(mktemp -d "${TMPDIR:-/tmp}/reinstate-live-installers.XXXXXX")
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM

for installer in install.sh install.ps1; do
  live_file="$temporary_directory/live-$installer"
  tagged_file="$temporary_directory/tagged-$installer"
  headers_file="$temporary_directory/headers-$installer"

  curl --proto '=https' --tlsv1.2 --fail --silent --show-error \
    --dump-header "$headers_file" \
    --output "$live_file" \
    "$base_url/$installer"
  curl --proto '=https' --tlsv1.2 --fail --silent --show-error \
    --output "$tagged_file" \
    "https://raw.githubusercontent.com/HarjjotSinghh/reinstate/$version/website/public/$installer"

  cmp "$tagged_file" "$live_file"
  grep -F "$version" "$live_file" >/dev/null

  normalized_headers="$temporary_directory/normalized-$installer"
  tr -d '\r' <"$headers_file" | tr '[:upper:]' '[:lower:]' >"$normalized_headers"
  grep -F 'content-type: text/plain; charset=utf-8' "$normalized_headers" >/dev/null
  grep -F 'x-content-type-options: nosniff' "$normalized_headers" >/dev/null
  grep -E '^cache-control: .*must-revalidate' "$normalized_headers" >/dev/null

  live_sha256=$(shasum -a 256 "$live_file" | awk '{print $1}')
  echo "$base_url/$installer matches $version ($live_sha256)"
done
