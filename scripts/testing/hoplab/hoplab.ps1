# One-line wrapper: .\scripts\testing\hoplab\hoplab.ps1 <command> [flags]
# from anywhere, in PowerShell 5.1 or 7+. See README.md next to this file.
$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..\..")
Push-Location $repoRoot
try {
    & go run ./scripts/testing/hoplab @args
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
