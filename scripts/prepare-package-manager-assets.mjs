#!/usr/bin/env node

import { chmod, copyFile, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import path from "node:path";
import process from "node:process";

const REPOSITORY = "https://github.com/HarjjotSinghh/reinstate";
const RELEASE_BASE = `${REPOSITORY}/releases/download`;
const DESCRIPTION = "Continue supported coding-agent sessions across configured devices.";

function parseArgs(argv) {
  const values = {};
  for (let index = 0; index < argv.length; index += 2) {
    const key = argv[index];
    const value = argv[index + 1];
    if (!key?.startsWith("--") || !value) {
      throw new Error("usage: prepare-package-manager-assets.mjs --version X.Y.Z --dist DIR --output DIR [--release-date YYYY-MM-DD]");
    }
    values[key.slice(2)] = value;
  }
  if (!/^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$/.test(values.version ?? "")) {
    throw new Error(`invalid release version: ${values.version ?? "<missing>"}`);
  }
  if (!values.dist || !values.output) {
    throw new Error("--dist and --output are required");
  }
  values["release-date"] ??= new Date().toISOString().slice(0, 10);
  if (!/^[0-9]{4}-[0-9]{2}-[0-9]{2}$/.test(values["release-date"])) {
    throw new Error(`invalid release date: ${values["release-date"]}`);
  }
  return values;
}

async function writeText(file, body, mode) {
  await mkdir(path.dirname(file), { recursive: true });
  await writeFile(file, body.endsWith("\n") ? body : `${body}\n`, "utf8");
  if (mode) await chmod(file, mode);
}

async function writeJSON(file, value) {
  await writeText(file, `${JSON.stringify(value, null, 2)}\n`);
}

function parseChecksums(body) {
  const checksums = new Map();
  for (const line of body.split(/\r?\n/)) {
    if (!line.trim()) continue;
    const match = line.match(/^([0-9a-f]{64})\s+\*?(.+)$/);
    if (!match) throw new Error(`invalid checksum line: ${line}`);
    checksums.set(match[2], match[1]);
  }
  return checksums;
}

function requiredChecksum(checksums, name) {
  const value = checksums.get(name);
  if (!value) throw new Error(`checksums.txt does not contain ${name}`);
  return value;
}

function npmLauncher() {
  return `#!/usr/bin/env node
"use strict";

const path = require("node:path");
const { spawnSync } = require("node:child_process");

const targets = {
  "darwin-x64": "@reinstate/cli-darwin-x64",
  "darwin-arm64": "@reinstate/cli-darwin-arm64",
  "linux-x64": "@reinstate/cli-linux-x64",
  "linux-arm64": "@reinstate/cli-linux-arm64",
  "win32-x64": "@reinstate/cli-win32-x64"
};
const key = \`\${process.platform}-\${process.arch}\`;
const packageName = targets[key];
if (!packageName) {
  console.error(\`Reinstate does not publish an npm binary for \${key}.\`);
  process.exit(1);
}

let manifest;
try {
  manifest = require.resolve(\`\${packageName}/package.json\`);
} catch {
  console.error(\`The optional package \${packageName} is missing. Reinstall @reinstate/cli without --omit=optional.\`);
  process.exit(1);
}

const executable = path.join(path.dirname(manifest), "bin", process.platform === "win32" ? "reinstate.exe" : "reinstate");
const result = spawnSync(executable, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
process.exit(result.status ?? 1);
`;
}

function jsrModule(version, targets) {
  return `/** Verified native launcher metadata for Reinstate ${version}. */
export const VERSION = ${JSON.stringify(version)};
export const RELEASE_TAG = ${JSON.stringify(`v${version}`)};

export const TARGETS = ${JSON.stringify(targets, null, 2)} as const;

export type ReinstateTarget = keyof typeof TARGETS;

/** Return the native release target for the current Deno host. */
export function currentTarget(): ReinstateTarget {
  const arch = Deno.build.arch === "x86_64" ? "amd64" : Deno.build.arch;
  const key = \`\${Deno.build.os}-\${arch}\`;
  if (!(key in TARGETS)) {
    throw new Error(\`Reinstate ${version} does not publish a binary for \${key}\`);
  }
  return key as ReinstateTarget;
}

function cacheRoot(): string {
  const explicit = Deno.env.get("REINSTATE_JSR_CACHE");
  if (explicit) return explicit;
  if (Deno.build.os === "windows") {
    const local = Deno.env.get("LOCALAPPDATA");
    if (!local) throw new Error("LOCALAPPDATA is unavailable; set REINSTATE_JSR_CACHE");
    return \`\${local}/Reinstate/jsr\`;
  }
  const home = Deno.env.get("HOME");
  const base = Deno.env.get("XDG_CACHE_HOME") ?? (home ? \`\${home}/.cache\` : undefined);
  if (!base) throw new Error("HOME is unavailable; set REINSTATE_JSR_CACHE");
  return \`\${base}/reinstate/jsr\`;
}

async function sha256(bytes: Uint8Array): Promise<string> {
  // Copy into an ArrayBuffer-backed view so this remains compatible with the
  // stricter BufferSource type used by current Deno/TypeScript releases.
  const digest = await crypto.subtle.digest("SHA-256", new Uint8Array(bytes));
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
}

/** Download, verify, and cache the immutable native binary for this package version. */
export async function ensureBinary(): Promise<string> {
  const target = currentTarget();
  const metadata = TARGETS[target];
  const separator = "/";
  const directory = [cacheRoot(), VERSION, target].join(separator);
  const binary = [directory, Deno.build.os === "windows" ? "reinstate.exe" : "reinstate"].join(separator);

  try {
    const existing = await Deno.readFile(binary);
    if (await sha256(existing) === metadata.sha256) return binary;
  } catch (error) {
    if (!(error instanceof Deno.errors.NotFound)) throw error;
  }

  const response = await fetch(metadata.url, { redirect: "follow" });
  if (!response.ok) throw new Error(\`download failed (HTTP \${response.status}): \${metadata.url}\`);
  const bytes = new Uint8Array(await response.arrayBuffer());
  const actual = await sha256(bytes);
  if (actual !== metadata.sha256) {
    throw new Error(\`checksum mismatch for \${metadata.asset}: got \${actual}, want \${metadata.sha256}\`);
  }

  await Deno.mkdir(directory, { recursive: true });
  const temporary = \`\${binary}.tmp-\${crypto.randomUUID()}\`;
  await Deno.writeFile(temporary, bytes, { createNew: true, mode: 0o755 });
  if (Deno.build.os === "windows") {
    try {
      await Deno.remove(binary);
    } catch (error) {
      if (!(error instanceof Deno.errors.NotFound)) throw error;
    }
  }
  await Deno.rename(temporary, binary);
  if (Deno.build.os !== "windows") await Deno.chmod(binary, 0o755);
  return binary;
}
`;
}

function jsrCLI() {
  return `#!/usr/bin/env -S deno run
import { ensureBinary } from "./mod.ts";

const executable = await ensureBinary();
const child = new Deno.Command(executable, {
  args: Deno.args,
  stdin: "inherit",
  stdout: "inherit",
  stderr: "inherit"
});
const status = await child.spawn().status;
Deno.exit(status.code);
`;
}

function homebrewFormula(version, checksums) {
  const archive = (os, arch, extension = "tar.gz") => `reinstate_${version}_${os}_${arch}.${extension}`;
  const stanza = (os, arch) => {
    const name = archive(os, arch);
    return `      url "${RELEASE_BASE}/v${version}/${name}"
      sha256 "${requiredChecksum(checksums, name)}"`;
  };
  return `class Reinstate < Formula
  desc "${DESCRIPTION}"
  homepage "https://reinstate.dev"
  version "${version}"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.arm?
${stanza("darwin", "arm64")}
    else
${stanza("darwin", "amd64")}
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
${stanza("linux", "arm64")}
    else
${stanza("linux", "amd64")}
    end
  end

  def install
    bin.install "reinstate"
    bin.install_symlink "reinstate" => "rein"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/rein version")
  end
end
`;
}

function chocolateyInstall(version, archive, checksum) {
  return `$ErrorActionPreference = 'Stop'
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$packageArgs = @{
  PackageName    = $env:ChocolateyPackageName
  Url64bit       = '${RELEASE_BASE}/v${version}/${archive}'
  UnzipLocation  = $toolsDir
  Checksum64     = '${checksum}'
  ChecksumType64 = 'sha256'
}
Install-ChocolateyZipPackage @packageArgs
New-Item -ItemType File -Path (Join-Path $toolsDir 'rein.exe.ignore') -Force | Out-Null
New-Item -ItemType File -Path (Join-Path $toolsDir 'reinstate.exe.ignore') -Force | Out-Null
Install-BinFile -Name 'reinstate' -Path (Join-Path $toolsDir 'reinstate.exe')
Install-BinFile -Name 'rein' -Path (Join-Path $toolsDir 'rein.exe')
`;
}

function chocolateyNuspec(version) {
  return `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd">
  <metadata>
    <id>reinstate</id>
    <version>${version}</version>
    <title>Reinstate</title>
    <authors>Harjot Singh Rana</authors>
    <owners>Harjot Singh Rana</owners>
    <projectUrl>https://reinstate.dev</projectUrl>
    <projectSourceUrl>${REPOSITORY}</projectSourceUrl>
    <packageSourceUrl>${REPOSITORY}/blob/v${version}/scripts/prepare-package-manager-assets.mjs</packageSourceUrl>
    <docsUrl>https://reinstate.dev/docs</docsUrl>
    <bugTrackerUrl>${REPOSITORY}/issues</bugTrackerUrl>
    <licenseUrl>${REPOSITORY}/blob/v${version}/LICENSE</licenseUrl>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <description>${DESCRIPTION} Reinstate uses end-to-end encryption and storage you control.</description>
    <summary>${DESCRIPTION}</summary>
    <releaseNotes>${REPOSITORY}/releases/tag/v${version}</releaseNotes>
    <tags>reinstate cli coding-agent codex claude sessions sync encryption</tags>
  </metadata>
  <files>
    <file src="tools\\**" target="tools" />
  </files>
</package>
`;
}

function wingetFiles(version, archive, checksum, releaseDate) {
  const identifier = "HarjotSinghRana.Reinstate";
  return {
    [`${identifier}.yaml`]: `PackageIdentifier: ${identifier}
PackageVersion: ${version}
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.10.0
`,
    [`${identifier}.installer.yaml`]: `PackageIdentifier: ${identifier}
PackageVersion: ${version}
InstallerLocale: en-US
InstallerType: zip
NestedInstallerType: portable
NestedInstallerFiles:
  - RelativeFilePath: rein.exe
    PortableCommandAlias: rein
  - RelativeFilePath: reinstate.exe
    PortableCommandAlias: reinstate
Architecture: x64
InstallerUrl: ${RELEASE_BASE}/v${version}/${archive}
InstallerSha256: ${checksum.toUpperCase()}
UpgradeBehavior: install
Commands:
  - rein
  - reinstate
ReleaseDate: ${releaseDate}
ManifestType: installer
ManifestVersion: 1.10.0
`,
    [`${identifier}.locale.en-US.yaml`]: `PackageIdentifier: ${identifier}
PackageVersion: ${version}
PackageLocale: en-US
Publisher: Harjot Singh Rana
PublisherUrl: https://harjot.co
PublisherSupportUrl: ${REPOSITORY}/issues
PackageName: Reinstate
PackageUrl: https://reinstate.dev
License: Apache-2.0
LicenseUrl: ${REPOSITORY}/blob/v${version}/LICENSE
ShortDescription: ${DESCRIPTION}
Description: Reinstate securely syncs supported coding-agent sessions across configured devices using end-to-end encryption and storage you control.
Tags:
  - cli
  - claude
  - coding-agent
  - codex
  - encryption
  - session
  - sync
ReleaseNotesUrl: ${REPOSITORY}/releases/tag/v${version}
ManifestType: defaultLocale
ManifestVersion: 1.10.0
`
  };
}

function aurFiles(version, checksums, licenseHash) {
  const amd64 = `reinstate_${version}_linux_amd64`;
  const arm64 = `reinstate_${version}_linux_arm64`;
  const pkgbuild = `# Maintainer: Harjot Singh Rana <harjot at harjot dot co>
pkgname=reinstate-bin
pkgver=${version}
pkgrel=1
pkgdesc='${DESCRIPTION}'
arch=('x86_64' 'aarch64')
url='https://reinstate.dev'
license=('Apache-2.0')
provides=('reinstate')
conflicts=('reinstate')
source=("LICENSE-$pkgver::${REPOSITORY}/raw/v$pkgver/LICENSE")
source_x86_64=("reinstate-$pkgver-x86_64::${RELEASE_BASE}/v$pkgver/${amd64}")
source_aarch64=("reinstate-$pkgver-aarch64::${RELEASE_BASE}/v$pkgver/${arm64}")
sha256sums=('${licenseHash}')
sha256sums_x86_64=('${requiredChecksum(checksums, amd64)}')
sha256sums_aarch64=('${requiredChecksum(checksums, arm64)}')

package() {
  install -Dm755 "$srcdir/reinstate-$pkgver-$CARCH" "$pkgdir/usr/bin/reinstate"
  ln -s reinstate "$pkgdir/usr/bin/rein"
  install -Dm644 "$srcdir/LICENSE-$pkgver" "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}
`;
  const srcinfo = `pkgbase = reinstate-bin
\tpkgdesc = ${DESCRIPTION}
\tpkgver = ${version}
\tpkgrel = 1
\turl = https://reinstate.dev
\tarch = x86_64
\tarch = aarch64
\tlicense = Apache-2.0
\tprovides = reinstate
\tconflicts = reinstate
\tsource = LICENSE-${version}::${REPOSITORY}/raw/v${version}/LICENSE
\tsha256sums = ${licenseHash}
\tsource_x86_64 = reinstate-${version}-x86_64::${RELEASE_BASE}/v${version}/${amd64}
\tsha256sums_x86_64 = ${requiredChecksum(checksums, amd64)}
\tsource_aarch64 = reinstate-${version}-aarch64::${RELEASE_BASE}/v${version}/${arm64}
\tsha256sums_aarch64 = ${requiredChecksum(checksums, arm64)}

pkgname = reinstate-bin
`;
  return { pkgbuild, srcinfo };
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const version = options.version;
  const dist = path.resolve(options.dist);
  const output = path.resolve(options.output);
  const cwd = path.resolve(process.cwd());
  if (output === cwd || output === path.parse(output).root) {
    throw new Error(`refusing unsafe output directory: ${output}`);
  }

  const checksums = parseChecksums(await readFile(path.join(dist, "checksums.txt"), "utf8"));
  const license = await readFile(path.join(cwd, "LICENSE"));
  const licenseHash = createHash("sha256").update(license).digest("hex");
  await rm(output, { recursive: true, force: true });
  await mkdir(output, { recursive: true });

  const platforms = [
    { npm: "darwin-x64", os: "darwin", cpu: "x64", goos: "darwin", goarch: "amd64", extension: "" },
    { npm: "darwin-arm64", os: "darwin", cpu: "arm64", goos: "darwin", goarch: "arm64", extension: "" },
    { npm: "linux-x64", os: "linux", cpu: "x64", goos: "linux", goarch: "amd64", extension: "" },
    { npm: "linux-arm64", os: "linux", cpu: "arm64", goos: "linux", goarch: "arm64", extension: "" },
    { npm: "win32-x64", os: "win32", cpu: "x64", goos: "windows", goarch: "amd64", extension: ".exe" }
  ];
  const optionalDependencies = {};
  const jsrTargets = {};
  for (const target of platforms) {
    const packageName = `@reinstate/cli-${target.npm}`;
    const asset = `reinstate_${version}_${target.goos}_${target.goarch}${target.extension}`;
    const checksum = requiredChecksum(checksums, asset);
    const packageDir = path.join(output, "npm", `cli-${target.npm}`);
    const binaryName = `reinstate${target.extension}`;
    await writeJSON(path.join(packageDir, "package.json"), {
      name: packageName,
      version,
      description: `Native Reinstate binary for ${target.os}/${target.cpu}`,
      license: "Apache-2.0",
      repository: { type: "git", url: `git+${REPOSITORY}.git` },
      homepage: "https://reinstate.dev",
      os: [target.os],
      cpu: [target.cpu],
      files: ["bin"]
    });
    await mkdir(path.join(packageDir, "bin"), { recursive: true });
    await copyFile(path.join(dist, asset), path.join(packageDir, "bin", binaryName));
    if (!target.extension) await chmod(path.join(packageDir, "bin", binaryName), 0o755);
    await copyFile(path.join(cwd, "LICENSE"), path.join(packageDir, "LICENSE"));
    await writeText(path.join(packageDir, "README.md"), `# ${packageName}\n\nPlatform package for [Reinstate](${REPOSITORY}). Install \`@reinstate/cli\` instead.\n`);
    optionalDependencies[packageName] = version;
    jsrTargets[`${target.goos}-${target.goarch}`] = {
      asset,
      sha256: checksum,
      url: `${RELEASE_BASE}/v${version}/${asset}`
    };
  }

  const npmRoot = path.join(output, "npm", "cli");
  await writeJSON(path.join(npmRoot, "package.json"), {
    name: "@reinstate/cli",
    version,
    description: DESCRIPTION,
    license: "Apache-2.0",
    author: "Harjot Singh Rana",
    repository: { type: "git", url: `git+${REPOSITORY}.git` },
    homepage: "https://reinstate.dev",
    bugs: `${REPOSITORY}/issues`,
    keywords: ["cli", "coding-agent", "claude", "codex", "sessions", "sync", "encryption"],
    engines: { node: ">=18" },
    bin: { rein: "bin/rein.js", reinstate: "bin/rein.js" },
    files: ["bin"],
    optionalDependencies,
    publishConfig: { access: "public" }
  });
  await writeText(path.join(npmRoot, "bin", "rein.js"), npmLauncher(), 0o755);
  await copyFile(path.join(cwd, "LICENSE"), path.join(npmRoot, "LICENSE"));
  await writeText(path.join(npmRoot, "README.md"), `# @reinstate/cli\n\nInstall Reinstate with \`npm install --global @reinstate/cli\`, then run \`rein version\`.\n\nThis package selects an embedded, platform-specific binary. It does not download executable code in a lifecycle script.\n`);

  const jsrDir = path.join(output, "jsr");
  await writeJSON(path.join(jsrDir, "jsr.json"), {
    $schema: "https://jsr.io/schema/config-file.v1.json",
    name: "@reinstate/cli",
    version,
    exports: { ".": "./mod.ts", "./cli": "./cli.ts" },
    publish: { include: ["LICENSE", "README.md", "cli.ts", "mod.ts"] }
  });
  await writeText(path.join(jsrDir, "mod.ts"), jsrModule(version, jsrTargets));
  await writeText(path.join(jsrDir, "cli.ts"), jsrCLI(), 0o755);
  await copyFile(path.join(cwd, "LICENSE"), path.join(jsrDir, "LICENSE"));
  await writeText(path.join(jsrDir, "README.md"), `# @reinstate/cli\n\nA checksum-verifying Deno/JSR launcher for Reinstate ${version}.\n\n\`\`\`sh\ndeno install --global --allow-env --allow-read --allow-write --allow-net=github.com --allow-run --name rein jsr:@reinstate/cli/cli\n\`\`\`\n`);

  await writeText(path.join(output, "homebrew", "Formula", "reinstate.rb"), homebrewFormula(version, checksums));

  const windowsArchive = `reinstate_${version}_windows_amd64.zip`;
  const windowsChecksum = requiredChecksum(checksums, windowsArchive);
  await writeJSON(path.join(output, "scoop", "reinstate.json"), {
    version,
    description: DESCRIPTION,
    homepage: "https://reinstate.dev",
    license: "Apache-2.0",
    architecture: {
      "64bit": {
        url: `${RELEASE_BASE}/v${version}/${windowsArchive}`,
        hash: windowsChecksum
      }
    },
    bin: [["reinstate.exe", "rein"], ["reinstate.exe", "reinstate"]],
    checkver: { github: REPOSITORY },
    autoupdate: {
      architecture: { "64bit": { url: `${REPOSITORY}/releases/download/v$version/reinstate_$version_windows_amd64.zip` } }
    }
  });

  const chocolatey = path.join(output, "chocolatey");
  await writeText(path.join(chocolatey, "reinstate.nuspec"), chocolateyNuspec(version));
  await writeText(path.join(chocolatey, "tools", "chocolateyinstall.ps1"), chocolateyInstall(version, windowsArchive, windowsChecksum));
  await writeText(path.join(chocolatey, "tools", "chocolateyuninstall.ps1"), "Uninstall-BinFile -Name 'rein'\nUninstall-BinFile -Name 'reinstate'\n");

  const winget = wingetFiles(version, windowsArchive, windowsChecksum, options["release-date"]);
  for (const [name, body] of Object.entries(winget)) {
    await writeText(path.join(output, "winget", name), body);
  }

  const aur = aurFiles(version, checksums, licenseHash);
  await writeText(path.join(output, "aur", "PKGBUILD"), aur.pkgbuild);
  await writeText(path.join(output, "aur", ".SRCINFO"), aur.srcinfo);

  await writeJSON(path.join(output, "manifest.json"), {
    version,
    releaseDate: options["release-date"],
    channels: ["npm", "jsr", "homebrew", "scoop", "chocolatey", "winget", "aur"]
  });
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : error);
  process.exit(1);
});
