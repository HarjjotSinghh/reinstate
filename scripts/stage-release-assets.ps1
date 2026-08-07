#Requires -Version 5.1
<#
.SYNOPSIS
  PowerShell-native parity for scripts/stage-release-assets.sh (no jq required).
#>
param(
    [string]$DistDir = "dist"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$resolvedDist = (Resolve-Path -LiteralPath $DistDir).Path
$artifactsPath = Join-Path $resolvedDist "artifacts.json"
if (-not (Test-Path -LiteralPath $artifactsPath)) {
    throw "missing artifacts.json in $resolvedDist (run a clean GoReleaser snapshot first)"
}

$artifacts = Get-Content -LiteralPath $artifactsPath -Raw | ConvertFrom-Json
if ($null -eq $artifacts) {
    throw "artifacts.json is empty or invalid"
}

foreach ($artifact in @($artifacts)) {
    $type = [string]$artifact.type
    $extraId = ""
    if ($null -ne $artifact.extra -and $null -ne $artifact.extra.ID) {
        $extraId = [string]$artifact.extra.ID
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

    $fullSource = if ([System.IO.Path]::IsPathRooted($sourcePath)) {
        $sourcePath
    } else {
        Join-Path $resolvedDist $sourcePath
    }
    $fullSource = [System.IO.Path]::GetFullPath($fullSource)
    $distPrefix = $resolvedDist.TrimEnd('\', '/') + [System.IO.Path]::DirectorySeparatorChar
    if (-not $fullSource.StartsWith($distPrefix, [System.StringComparison]::OrdinalIgnoreCase) -and
        -not ($fullSource.Equals($resolvedDist, [System.StringComparison]::OrdinalIgnoreCase))) {
        throw "refusing to stage binary outside $resolvedDist: $fullSource"
    }
    if (-not (Test-Path -LiteralPath $fullSource)) {
        throw "missing raw binary source: $fullSource"
    }

    $destination = Join-Path $resolvedDist $assetName
    Copy-Item -LiteralPath $fullSource -Destination $destination -Force
}

Write-Host "staged raw release binaries into $resolvedDist"
