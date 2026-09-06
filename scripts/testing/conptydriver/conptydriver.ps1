# One-line wrapper: .\scripts\testing\conptydriver\conptydriver.ps1 [flags] -- <command> [args...]
# from anywhere, in PowerShell 5.1 or 7+. Windows only (ConPTY); see
# README.md next to this file.
$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..\..")
Push-Location $repoRoot
try {
    & go run ./scripts/testing/conptydriver @args
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
