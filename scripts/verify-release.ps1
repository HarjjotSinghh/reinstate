#Requires -Version 5.1
param(
    [string]$DistDir = "dist"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Keep the historical entrypoint, but enforce full check-release-artifacts parity.
& (Join-Path $PSScriptRoot "check-release-artifacts.ps1") -DistDir $DistDir
