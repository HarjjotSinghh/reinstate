import { readFile } from 'node:fs/promises';
import { pathToFileURL } from 'node:url';

export const EXPECTED_VERCEL_PROJECT_LINK = Object.freeze({
  projectName: 'reinstate-web',
  projectId: 'prj_DpNGbqB5dPT1nC3gXm2yUArgOvuJ',
  orgId: 'team_HNHkbCcwzJVaRRWt7XpeY51E',
});

const EXPECTED_KEYS = Object.keys(EXPECTED_VERCEL_PROJECT_LINK).sort();

export function validateVercelProjectLink(candidate) {
  if (
    candidate === null ||
    typeof candidate !== 'object' ||
    Array.isArray(candidate)
  ) {
    throw new Error('Vercel project link must be a JSON object');
  }

  const actualKeys = Object.keys(candidate).sort();
  if (
    actualKeys.length !== EXPECTED_KEYS.length ||
    actualKeys.some((key, index) => key !== EXPECTED_KEYS[index])
  ) {
    throw new Error(
      `Vercel project link must contain exactly: ${EXPECTED_KEYS.join(', ')}`,
    );
  }

  for (const [key, expected] of Object.entries(
    EXPECTED_VERCEL_PROJECT_LINK,
  )) {
    if (candidate[key] !== expected) {
      throw new Error(
        `Vercel project link ${key} must be ${JSON.stringify(expected)}`,
      );
    }
  }

  return candidate;
}

export async function checkVercelProjectLink(
  projectPath = '.vercel/project.json',
) {
  let source;
  try {
    source = await readFile(projectPath, 'utf8');
  } catch (error) {
    throw new Error(
      `Cannot read Vercel project link at ${projectPath}: ${error.message}`,
      { cause: error },
    );
  }

  let candidate;
  try {
    candidate = JSON.parse(source);
  } catch (error) {
    throw new Error(
      `Vercel project link at ${projectPath} is not valid JSON: ${error.message}`,
      { cause: error },
    );
  }

  return validateVercelProjectLink(candidate);
}

async function main(arguments_) {
  if (arguments_.length > 1) {
    throw new Error(
      'usage: node scripts/check-vercel-project-link.mjs [project-json-path]',
    );
  }

  const projectPath = arguments_[0] ?? '.vercel/project.json';
  const project = await checkVercelProjectLink(projectPath);
  process.stdout.write(
    `Vercel project link verified (${project.projectName}, ${project.projectId}, ${project.orgId})\n`,
  );
}

if (process.argv[1] && pathToFileURL(process.argv[1]).href === import.meta.url) {
  main(process.argv.slice(2)).catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
