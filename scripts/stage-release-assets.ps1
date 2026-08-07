#Requires -Version 5.1
<#
.SYNOPSIS
  PowerShell-native parity for scripts/stage-release-assets.sh (no jq required).

.DESCRIPTION
  GoReleaser writes raw binary paths in artifacts.json relative to the repository
  root (for example dist/reinstate_windows_amd64_v1/reinstate.exe), not relative
  to the dist directory itself. This script resolves those paths from the
  process working directory and copies them to top-level checksummed names.
#>
param(
    [string]$DistDir = "dist"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-RepoRelativeFullPath {
    param(
        [Parameter(Mandatory = $true)][string]$PathValue,
        [Parameter(Mandatory = $true)][string]$RepoRoot
    )
    if ([System.IO.Path]::IsPathRooted($PathValue)) {
        return [System.IO.Path]::GetFullPath($PathValue)
    }
    # artifacts.json paths are repo-root-relative (dist/...), matching the shell helper.
    $combined = Join-Path $RepoRoot ($PathValue -replace '/', [System.IO.Path]::DirectorySeparatorChar)
    return [System.IO.Path]::GetFullPath($combined)
}

$repoRoot = (Get-Location).Path
$resolvedDist = (Resolve-Path -LiteralPath $DistDir).Path
$artifactsPath = Join-Path $resolvedDist "artifacts.json"
if (-not (Test-Path -LiteralPath $artifactsPath)) {
    throw "missing artifacts.json in $resolvedDist (run a clean GoReleaser snapshot first)"
}

$artifacts = Get-Content -LiteralPath $artifactsPath -Raw | ConvertFrom-Json
if ($null -eq $artifacts) {
    throw "artifacts.json is empty or invalid"
}

$staged = 0
foreach ($artifact in @($artifacts)) {
    $type = [string]$artifact.type
    $extraId = ""
    if ($null -ne $artifact.extra) {
        # GoReleaser emits Extra.ID; tolerate id for forward compatibility.
        if ($null -ne $artifact.extra.ID) {
            $extraId = [string]$artifact.extra.ID
        } elseif ($null -ne $artifact.extra.id) {
            $extraId = [string]$artifact.extra.id
        }
    }
    if ($type -ne "Binary" -or $extraId -ne "raw") {
        continue
    }

    $sourcePath = [string]$artifact.path
    $assetName = [string]$artifact.name
    if ([string]::IsNullOrWhiteSpace($sourcePath) -or [string]::IsNullOrWhiteSpace($assetName)) {
        throw "raw binary artifact is missing path or name"
    }
    if ($assetName -match '[\\/]') {
        throw "refusing unsafe release asset name: $assetName"
    }

    $fullSource = Get-RepoRelativeFullPath -PathValue $sourcePath -RepoRoot $repoRoot
    $distPrefix = $resolvedDist.TrimEnd('\', '/') + [System.IO.Path]::DirectorySeparatorChar
    if (-not $fullSource.StartsWith($distPrefix, [System.StringComparison]::OrdinalIgnoreCase) -and
        -not ($fullSource.Equals($resolvedDist, [System.StringComparison]::OrdinalIgnoreCase))) {
        # Use ${} so Windows PowerShell 5.1 does not parse $resolvedDist: as a
        # drive-scoped variable (parse error at $resolvedDist:).
        throw "refusing to stage binary outside ${resolvedDist}: ${fullSource} (from ${sourcePath})"
    }
    if (-not (Test-Path -LiteralPath $fullSource)) {
        throw "missing raw binary source: ${fullSource} (from ${sourcePath})"
    }

    $destination = Join-Path $resolvedDist $assetName
    Copy-Item -LiteralPath $fullSource -Destination $destination -Force
    $staged++
}

if ($staged -eq 0) {
    throw "no raw Binary artifacts with extra.ID=raw were staged from $artifactsPath"
}

Write-Host "staged $staged raw release binaries into ${resolvedDist}"
