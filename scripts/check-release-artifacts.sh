#!/bin/sh
set -eu

dist_dir=${1:-dist}
checksums="$dist_dir/checksums.txt"

test -f "$checksums"
(cd "$dist_dir" && sha256sum -c checksums.txt)

archives=$(find "$dist_dir" -maxdepth 1 -type f \( -name 'reinstate_*_darwin_*.tar.gz' -o -name 'reinstate_*_linux_*.tar.gz' -o -name 'reinstate_*_windows_*.zip' \) | sort)
archive_count=$(printf '%s\n' "$archives" | sed '/^$/d' | wc -l | tr -d ' ')
test "$archive_count" -eq 5

raw_binaries=$(awk -v prefix="$dist_dir/" '
	$2 ~ /^reinstate_[^ ]+_(darwin|linux)_(amd64|arm64)$/ ||
	$2 ~ /^reinstate_[^ ]+_windows_amd64\.exe$/ {
		print prefix $2
	}
' "$checksums" | sort)
raw_binary_count=$(printf '%s\n' "$raw_binaries" | sed '/^$/d' | wc -l | tr -d ' ')
test "$raw_binary_count" -eq 5

linux_packages=$(find "$dist_dir" -maxdepth 1 -type f \( -name 'reinstate_*_linux_*.apk' -o -name 'reinstate_*_linux_*.deb' -o -name 'reinstate_*_linux_*.pkg.tar.zst' -o -name 'reinstate_*_linux_*.rpm' \) | sort)
linux_package_count=$(printf '%s\n' "$linux_packages" | sed '/^$/d' | wc -l | tr -d ' ')
test "$linux_package_count" -eq 8

for archive in $archives; do
	test -f "$archive.sbom.json"
	case "$archive" in
		*.tar.gz)
			contents=$(tar -tzf "$archive")
			;;
		*.zip)
			contents=$(unzip -Z1 "$archive")
			;;
	esac
	printf '%s\n' "$contents" | grep -Eq '(^|/)reinstate(\.exe)?$'
	case "$archive" in
		*_windows_*.zip)
			printf '%s\n' "$contents" | grep -Eq '(^|/)rein\.exe$'
			;;
	esac
	for required in LICENSE NOTICE README.md CHANGELOG.md; do
		printf '%s\n' "$contents" | grep -Eq "(^|/)$required$"
	done
done

for binary in $raw_binaries; do
	test -s "$binary"
done

source_archive=$(find "$dist_dir" -maxdepth 1 -type f -name 'reinstate_*_source.tar.gz' -print)
test -n "$source_archive"
test "$(printf '%s\n' "$source_archive" | wc -l | tr -d ' ')" -eq 1
source_contents=$(tar -tzf "$source_archive")
printf '%s\n' "$source_contents" | grep -Eq '^reinstate-[^/]+/go\.mod$'
if printf '%s\n' "$source_contents" | grep -Eq '(^|/)(\.git|bin|dist|\.env)(/|$)'; then
	echo "source archive contains a forbidden generated or secret path" >&2
	exit 1
fi
