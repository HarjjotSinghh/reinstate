import { pathToFileURL } from 'node:url';

const CLI_TAG_PATTERN =
  /^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$/;

export function expectedCliReleaseAssets(tag) {
  if (!CLI_TAG_PATTERN.test(tag)) {
    throw new Error(`invalid CLI release tag: ${JSON.stringify(tag)}`);
  }

  const version = tag.slice(1);
  const archives = [
    `reinstate_${version}_darwin_amd64.tar.gz`,
    `reinstate_${version}_darwin_arm64.tar.gz`,
    `reinstate_${version}_linux_amd64.tar.gz`,
    `reinstate_${version}_linux_arm64.tar.gz`,
    `reinstate_${version}_windows_amd64.zip`,
  ];

  return [
    'checksums.txt',
    ...archives,
    ...archives.map((archive) => `${archive}.sbom.json`),
    `reinstate_${version}_source.tar.gz`,
  ];
}

export function validateCliRelease(candidate, tag) {
  const requiredAssets = expectedCliReleaseAssets(tag);
  if (
    candidate === null ||
    typeof candidate !== 'object' ||
    Array.isArray(candidate)
  ) {
    throw new Error('GitHub CLI release response must be a JSON object');
  }
  if (candidate.tagName !== tag) {
    throw new Error(
      `GitHub CLI release tagName must be ${JSON.stringify(tag)}`,
    );
  }
  if (candidate.isDraft !== false) {
    throw new Error(`GitHub CLI release ${tag} must not be a draft`);
  }
  if (
    typeof candidate.publishedAt !== 'string' ||
    candidate.publishedAt.length === 0 ||
    !Number.isFinite(Date.parse(candidate.publishedAt))
  ) {
    throw new Error(`GitHub CLI release ${tag} must be published`);
  }
  if (!Array.isArray(candidate.assets)) {
    throw new Error(`GitHub CLI release ${tag} assets must be an array`);
  }

  const assetNames = candidate.assets.map((asset, index) => {
    if (
      asset === null ||
      typeof asset !== 'object' ||
      Array.isArray(asset) ||
      typeof asset.name !== 'string' ||
      asset.name.length === 0
    ) {
      throw new Error(
        `GitHub CLI release ${tag} asset ${index + 1} must have a name`,
      );
    }
    return asset.name;
  });
  const uniqueAssetNames = new Set(assetNames);
  if (uniqueAssetNames.size !== assetNames.length) {
    throw new Error(`GitHub CLI release ${tag} contains duplicate asset names`);
  }

  const missing = requiredAssets.filter(
    (asset) => !uniqueAssetNames.has(asset),
  );
  if (missing.length > 0) {
    throw new Error(
      `GitHub CLI release ${tag} is missing required assets: ${missing.join(', ')}`,
    );
  }
  const assetsByName = new Map(
    candidate.assets.map((asset) => [asset.name, asset]),
  );
  const unavailable = requiredAssets.filter(
    (asset) => assetsByName.get(asset).state !== 'uploaded',
  );
  if (unavailable.length > 0) {
    throw new Error(
      `GitHub CLI release ${tag} required assets are not uploaded: ${unavailable.join(', ')}`,
    );
  }

  return {
    tag,
    publishedAt: candidate.publishedAt,
    requiredAssets,
  };
}

export function parseArguments(arguments_) {
  if (
    arguments_.length !== 2 ||
    arguments_[0] !== '--tag' ||
    arguments_[1].length === 0
  ) {
    throw new Error(
      'usage: node scripts/check-cli-release.mjs --tag <vX.Y.Z[-prerelease]>',
    );
  }
  return { tag: arguments_[1] };
}

async function readStandardInput() {
  let input = '';
  process.stdin.setEncoding('utf8');
  for await (const chunk of process.stdin) {
    input += chunk;
  }
  return input;
}

async function main(arguments_) {
  const { tag } = parseArguments(arguments_);
  const source = await readStandardInput();

  let candidate;
  try {
    candidate = JSON.parse(source);
  } catch (error) {
    throw new Error(
      `GitHub CLI release response is not valid JSON: ${error.message}`,
      { cause: error },
    );
  }

  const release = validateCliRelease(candidate, tag);
  process.stdout.write(
    `GitHub CLI release ${release.tag} verified (${release.requiredAssets.length} required assets)\n`,
  );
}

if (process.argv[1] && pathToFileURL(process.argv[1]).href === import.meta.url) {
  main(process.argv.slice(2)).catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
