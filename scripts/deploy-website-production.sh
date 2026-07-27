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
  npm run check:seo
  npm run check:links
  npm run check:performance
  npm run check:freshness
  npm run check:indexnow
  npm run check:lighthouse
  node scripts/indexnow.mjs \
    --current dist/client/sitemap-index.xml \
    --previous https://reinstate.dev/sitemap-index.xml \
    --allow-missing-previous \
    --output "artifacts/indexnow/$version-plan.json"
)

deployment_output=$(
  cd website
  npx --yes vercel deploy --prod --skip-domain --scope harjjot --yes
)
echo "$deployment_output"
deployment_url=$(
  printf '%s\n' "$deployment_output" |
    node website/scripts/parse-vercel-deployment-url.mjs
)
if [ -z "$deployment_url" ]; then
  echo "Vercel did not return an immutable deployment URL" >&2
  exit 1
fi

"$repo_directory/scripts/verify-live-installers.sh" "$version" "$deployment_url"
(
  cd website
  npm run check:production-discovery -- \
    --base-url "$deployment_url" \
    --allow-non-production \
    --output "artifacts/production-discovery/$version-immutable.json"
)
(
  cd website
  npx --yes vercel promote "$deployment_url" --scope harjjot --yes
)
"$repo_directory/scripts/verify-live-installers.sh" "$version" "https://reinstate.dev"
(
  cd website
  npm run check:production-discovery -- \
    --base-url "https://reinstate.dev" \
    --output "artifacts/production-discovery/$version-production.json"
)
echo "IndexNow plan saved at website/artifacts/indexnow/$version-plan.json"
echo "Review it, publish the key proof, then submit it explicitly with INDEXNOW_KEY."
