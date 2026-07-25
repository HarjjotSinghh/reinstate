#!/bin/sh
set -eu

dist_dir=${1:-dist}
checksums="$dist_dir/checksums.txt"

test -f "$checksums"
(cd "$dist_dir" && sha256sum -c checksums.txt)

archives=$(find "$dist_dir" -maxdepth 1 -type f \( -name 'reinstate_*_darwin_*.tar.gz' -o -name 'reinstate_*_linux_*.tar.gz' -o -name 'reinstate_*_windows_*.zip' \) | sort)
archive_count=$(printf '%s\n' "$archives" | sed '/^$/d' | wc -l | tr -d ' ')
test "$archive_count" -eq 5

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
	for required in LICENSE NOTICE README.md CHANGELOG.md; do
		printf '%s\n' "$contents" | grep -Eq "(^|/)$required$"
	done
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
