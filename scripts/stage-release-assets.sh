#!/bin/sh
set -eu

dist_dir=${1:-dist}
artifacts="$dist_dir/artifacts.json"

test -f "$artifacts"
command -v jq >/dev/null 2>&1 || {
	echo "jq is required to stage raw release binaries" >&2
	exit 1
}

jq -r '.[] | select(.type == "Binary" and (.extra.ID // "") == "raw") | [.path, .name] | @tsv' "$artifacts" |
while IFS="$(printf '\t')" read -r source_path asset_name; do
	test -n "$source_path"
	test -n "$asset_name"
	case "$source_path" in
		"$dist_dir"/*) ;;
		*)
			echo "refusing to stage binary outside $dist_dir: $source_path" >&2
			exit 1
			;;
	esac
	case "$asset_name" in
		*/* | *\\*)
			echo "refusing unsafe release asset name: $asset_name" >&2
			exit 1
			;;
	esac
	test -f "$source_path"
	cp "$source_path" "$dist_dir/$asset_name"
done
