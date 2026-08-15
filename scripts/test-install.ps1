param(
    [string]$DistDir = "dist"
)

$ErrorActionPreference = "Stop"
$RepoDir = Split-Path -Parent $PSScriptRoot

& (Join-Path $PSScriptRoot "verify-release.ps1") -DistDir $DistDir
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

Set-Location $RepoDir
if (-not $env:GOTOOLCHAIN) {
    $env:GOTOOLCHAIN = "go1.25.13"
}
go test ./internal/doctest -run TestInstaller -count=1
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
