#Requires -Version 5.1
<#
.SYNOPSIS
  PowerShell-native parity for scripts/check-release-binary-identity.sh on Windows hosts.
#>
param(
    [Parameter(Mandatory = $true)][string]$DistDir,
    [Parameter(Mandatory = $true)][string]$ExpectedCommit,
    [Parameter(Mandatory = $true)][string]$ExpectedVersion
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ($ExpectedCommit -notmatch '^[0-9a-fA-F]{40}$') {
    throw "expected commit must be a full 40-character git SHA"
}
if ([string]::IsNullOrWhiteSpace($ExpectedVersion)) {
    throw "expected version is required"
}

$resolvedDist = (Resolve-Path -LiteralPath $DistDir).Path
$arch = $env:PROCESSOR_ARCHITECTURE
switch -Regex ($arch) {
    '^(AMD64|X64)$' { $hostArch = "amd64" }
    '^(ARM64)$' { $hostArch = "arm64" }
    default { throw "unsupported release-validation architecture: $arch" }
}

$archiveName = "reinstate_${ExpectedVersion}_windows_${hostArch}.zip"
$archivePath = Join-Path $resolvedDist $archiveName
if (-not (Test-Path -LiteralPath $archivePath)) {
    throw "missing host archive: $archiveName"
}

$identityDir = Join-Path ([System.IO.Path]::GetTempPath()) ("reinstate-identity-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $identityDir | Out-Null
try {
    Expand-Archive -LiteralPath $archivePath -DestinationPath $identityDir -Force
    $binary = Get-ChildItem -LiteralPath $identityDir -Recurse -File |
        Where-Object { $_.Name -eq "reinstate.exe" } |
        Select-Object -First 1
    if ($null -eq $binary) {
        throw "archive is missing reinstate.exe"
    }

    $identityJson = & $binary.FullName version --json
    if ($LASTEXITCODE -ne 0) {
        throw "version --json failed with exit $LASTEXITCODE"
    }
    $identity = $identityJson | ConvertFrom-Json
    if ($identity.commit -ne $ExpectedCommit) {
        throw "release binary commit mismatch: expected $ExpectedCommit, got $($identity.commit)"
    }
    if ($identity.version -ne $ExpectedVersion) {
        throw "release binary version mismatch: expected $ExpectedVersion, got $($identity.version)"
    }
    Write-Host "release binary identity ok: version=$($identity.version) commit=$($identity.commit)"
} finally {
    Remove-Item -LiteralPath $identityDir -Recurse -Force -ErrorAction SilentlyContinue
}
