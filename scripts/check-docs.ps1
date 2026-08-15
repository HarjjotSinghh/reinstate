$ErrorActionPreference = "Stop"

$RepoDir = Split-Path -Parent $PSScriptRoot
Set-Location $RepoDir

if (-not $env:GOTOOLCHAIN) {
    $env:GOTOOLCHAIN = "go1.25.13"
}

go test ./internal/doctest -count=1
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
