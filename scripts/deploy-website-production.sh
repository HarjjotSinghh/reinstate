#!/bin/sh
set -eu

usage() {
  echo "usage: $0 website-vYYYY.MM.DD.N" >&2
  exit 2
}

[ "$#" -eq 1 ] || usage
deployment_tag=$1
printf '%s\n' "$deployment_tag" |
  grep -Eq '^website-v[0-9]{4}\.[0-9]{2}\.[0-9]{2}\.[1-9][0-9]*$' ||
  usage
tag_body=${deployment_tag#website-v}
year=${tag_body%%.*}
month_rest=${tag_body#*.}
month=${month_rest%%.*}
day_rest=${month_rest#*.}
day=${day_rest%%.*}
invalid_website_date() {
  echo "invalid website deployment date: ${year}-${month}-${day}" >&2
  exit 1
}
case $month in
  01|02|03|04|05|06|07|08|09|10|11|12) ;;
  *) invalid_website_date ;;
esac
case $day in
  0[1-9]|1[0-9]|2[0-9]|30|31) ;;
  *) invalid_website_date ;;
esac
case $month in
  04|06|09|11)
    [ "$day" != "31" ] || invalid_website_date
    ;;
  02)
    if [ "$day" = "30" ] || [ "$day" = "31" ]; then
      invalid_website_date
    fi
    if [ "$day" = "29" ]; then
      leap=0
      if [ $((year % 4)) -eq 0 ] && { [ $((year % 100)) -ne 0 ] || [ $((year % 400)) -eq 0 ]; }; then
        leap=1
      fi
      [ "$leap" -eq 1 ] || invalid_website_date
    fi
    ;;
esac

repo_directory=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_directory"

vercel_cli() {
  npm exec --yes --package=vercel@57.0.0 -- vercel "$@"
}

if [ "$(git branch --show-current)" != "main" ]; then
  echo "production deployment requires the main branch" >&2
  exit 1
fi
if [ -n "$(git status --porcelain)" ]; then
  echo "production deployment requires a clean worktree" >&2
  exit 1
fi

posix_version_count=$(
  grep -Ec '^VERSION="v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?"$' \
    website/public/install.sh ||
    true
)
posix_assignment_count=$(
  grep -Ec '^[[:space:]]*VERSION[[:space:]]*=' website/public/install.sh ||
    true
)
powershell_version_count=$(
  grep -Ec '^\$Version = "v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?"$' \
    website/public/install.ps1 ||
    true
)
powershell_assignment_count=$(
  grep -Eic '^[[:space:]]*\$Version[[:space:]]*=' website/public/install.ps1 ||
    true
)
if [ "$posix_assignment_count" -ne 1 ] ||
  [ "$powershell_assignment_count" -ne 1 ] ||
  [ "$posix_version_count" -ne 1 ] ||
  [ "$powershell_version_count" -ne 1 ]; then
  echo "public installers must each declare exactly one pinned CLI release version" >&2
  exit 1
fi
posix_version=$(
  sed -n 's/^VERSION="\([^"]*\)"$/\1/p' website/public/install.sh
)
powershell_version=$(
  sed -n 's/^\$Version = "\([^"]*\)"$/\1/p' website/public/install.ps1
)
if [ "$posix_version" != "$powershell_version" ]; then
  echo "public installer CLI versions do not match" >&2
  exit 1
fi
cli_version=$posix_version
printf '%s\n' "$cli_version" |
  grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$' || {
  echo "public installer CLI version is not valid SemVer: $cli_version" >&2
  exit 1
}

git fetch --quiet --force origin \
  "refs/heads/main:refs/remotes/origin/main" \
  "refs/tags/$deployment_tag:refs/tags/$deployment_tag" \
  "refs/tags/$cli_version:refs/tags/$cli_version"
head_commit=$(git rev-parse HEAD)
origin_commit=$(git rev-parse origin/main)
deployment_commit=$(git rev-parse "$deployment_tag^{}")
if [ "$head_commit" != "$origin_commit" ] || [ "$head_commit" != "$deployment_commit" ]; then
  echo "HEAD, origin/main, and the peeled website deployment tag must be identical" >&2
  exit 1
fi
if [ "$(git cat-file -t "$deployment_tag")" != "tag" ]; then
  echo "$deployment_tag must be an annotated tag" >&2
  exit 1
fi
git -c gpg.format=ssh \
  -c gpg.ssh.allowedSignersFile=.github/allowed_signers \
  verify-tag "$deployment_tag"

if [ "$(git cat-file -t "$cli_version")" != "tag" ]; then
  echo "$cli_version must be an annotated tag" >&2
  exit 1
fi
git -c gpg.format=ssh \
  -c gpg.ssh.allowedSignersFile=.github/allowed_signers \
  verify-tag "$cli_version"
git merge-base --is-ancestor "$cli_version^{}" "$deployment_commit"

git diff --exit-code "$cli_version^{}" -- \
  website/public/install.sh \
  website/public/install.ps1
product_release_count=$(
  grep -Ec '^[[:space:]]*currentRelease[[:space:]]*:' website/src/data/product.ts ||
    true
)
if [ "$product_release_count" -ne 1 ] ||
  [ "$(grep -Fc "currentRelease: '$cli_version'," website/src/data/product.ts)" -ne 1 ]; then
  echo "website product currentRelease must equal public installer version $cli_version" >&2
  exit 1
fi

release_json=$(
  gh release view "$cli_version" \
    --repo HarjjotSinghh/reinstate \
    --json tagName,isDraft,isPrerelease,publishedAt,assets
)
printf '%s\n' "$release_json" |
  node website/scripts/check-cli-release.mjs --tag "$cli_version"

echo "Website deployment tag: $deployment_tag"
echo "CLI installer release: $cli_version"

(
  cd website
  npm ci
  node scripts/check-vercel-project-link.mjs
  npm test
  npm run build
  npm run check:seo
  npm run check:links
  npm run check:agent-surface
  npm run check:performance
  npm run check:freshness
  npm run check:indexnow
  npm run check:lighthouse
  node scripts/indexnow.mjs \
    --current dist/client/sitemap-index.xml \
    --previous https://reinstate.dev/sitemap-index.xml \
    --allow-missing-previous \
    --output "artifacts/indexnow/$deployment_tag-plan.json"
)

deployment_output=$(
  cd website
  vercel_cli deploy --prod --skip-domain --scope harjjot --yes
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

"$repo_directory/scripts/verify-live-installers.sh" "$cli_version" "$deployment_url"
(
  cd website
  npm run check:production-discovery -- \
    --base-url "$deployment_url" \
    --allow-non-production \
    --allow-vercel-preview-noindex \
    --output "artifacts/production-discovery/$deployment_tag-immutable.json"
)
(
  cd website
  vercel_cli promote "$deployment_url" --scope harjjot --yes
)
"$repo_directory/scripts/verify-live-installers.sh" "$cli_version" "https://reinstate.dev"
(
  cd website
  npm run check:production-discovery -- \
    --base-url "https://reinstate.dev" \
    --output "artifacts/production-discovery/$deployment_tag-production.json"
)
echo "IndexNow plan saved at website/artifacts/indexnow/$deployment_tag-plan.json"
echo "Review it, publish the key proof, then submit it explicitly with INDEXNOW_KEY."
