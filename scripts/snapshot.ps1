#Requires -Version 5.1
<#
.SYNOPSIS
  Windows-native GoReleaser snapshot with the same tag env contract as `make snapshot`.

.DESCRIPTION
  Avoids MSYS/make exit-code masking. Fails with clear messages when goreleaser or
  syft is missing. Run from the repository root in native PowerShell:

    powershell -NoProfile -File .\scripts\snapshot.ps1
#>
param(
    [string]$DistDir = "dist"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Assert-Command {
    param([Parameter(Mandatory = $true)][string]$Name)
    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if ($null -eq $cmd) {
        throw "$Name is required on PATH for Windows snapshot acceptance"
    }
    return $cmd.Source
}

$repoRoot = (Get-Location).Path
if (-not (Test-Path -LiteralPath (Join-Path $repoRoot ".goreleaser.yml"))) {
    throw "run snapshot.ps1 from the repository root"
}

$goreleaser = Assert-Command goreleaser
# Syft is invoked by GoReleaser for archive SBOMs; surface a clear error early.
try {
    Assert-Command syft | Out-Null
} catch {
    Write-Host "warning: syft not found on PATH; GoReleaser SBOM steps may fail" -ForegroundColor Yellow
}

$currentTag = (& git describe --tags --match "v[0-9]*" --abbrev=0 2>$null)
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($currentTag)) {
    throw "git describe failed; fetch tags (git fetch --tags) so snapshot can set GORELEASER_CURRENT_TAG"
}
$previousTag = ""
$prev = & git describe --tags --match "v[0-9]*" --abbrev=0 "$currentTag^" 2>$null
if ($LASTEXITCODE -eq 0 -and -not [string]::IsNullOrWhiteSpace($prev)) {
    $previousTag = $prev.Trim()
}

$env:GORELEASER_CURRENT_TAG = $currentTag.Trim()
if ($previousTag) {
    $env:GORELEASER_PREVIOUS_TAG = $previousTag
} else {
    Remove-Item Env:GORELEASER_PREVIOUS_TAG -ErrorAction SilentlyContinue
}

Write-Host "snapshot: goreleaser=$goreleaser current_tag=$($env:GORELEASER_CURRENT_TAG) previous_tag=$previousTag"
& $goreleaser release --snapshot --clean
if ($LASTEXITCODE -ne 0) {
    throw "goreleaser release --snapshot --clean failed with exit $LASTEXITCODE"
}

$artifacts = Join-Path $repoRoot (Join-Path $DistDir "artifacts.json")
if (-not (Test-Path -LiteralPath $artifacts)) {
    throw "snapshot finished without $artifacts"
}
Write-Host "snapshot ok: $artifacts"
