#!/bin/sh
set -eu

usage() {
  echo "usage: $0 vX.Y.Z[-prerelease]" >&2
  exit 2
}

[ "$#" -eq 1 ] || usage
version=$1
case "$version" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) usage ;;
esac

repo_directory=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_directory"

if [ "$(git branch --show-current)" != "main" ]; then
  echo "production deployment requires the main branch" >&2
  exit 1
fi
if [ -n "$(git status --porcelain)" ]; then
  echo "production deployment requires a clean worktree" >&2
  exit 1
fi

git fetch --quiet origin main "$version"
head_commit=$(git rev-parse HEAD)
origin_commit=$(git rev-parse origin/main)
tag_commit=$(git rev-parse "$version^{}")
if [ "$head_commit" != "$origin_commit" ] || [ "$head_commit" != "$tag_commit" ]; then
  echo "HEAD, origin/main, and the peeled release tag must be identical" >&2
  exit 1
fi
if [ "$(git cat-file -t "$version")" != "tag" ]; then
  echo "$version must be an annotated tag" >&2
  exit 1
fi
git -c gpg.format=ssh \
  -c gpg.ssh.allowedSignersFile=.github/allowed_signers \
  verify-tag "$version"

grep -F "$version" website/public/install.sh >/dev/null
grep -F "$version" website/public/install.ps1 >/dev/null

(
  cd website
  npm ci
  npm test
  npm run build
)

deployment_output=$(
  cd website
  npx --yes vercel deploy --prod --skip-domain --scope harjjot --yes
)
echo "$deployment_output"
deployment_url=$(printf '%s\n' "$deployment_output" | awk '/^https:\/\// { url=$0 } END { print url }')
if [ -z "$deployment_url" ]; then
  echo "Vercel did not return an immutable deployment URL" >&2
  exit 1
fi

"$repo_directory/scripts/verify-live-installers.sh" "$version" "$deployment_url"
(
  cd website
  npx --yes vercel promote "$deployment_url" --scope harjjot --yes
)
"$repo_directory/scripts/verify-live-installers.sh" "$version" "https://reinstate.dev"
