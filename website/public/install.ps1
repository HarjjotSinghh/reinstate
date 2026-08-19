# Reinstate public bootstrap - pins and verifies one canonical release installer.
# Usage: irm https://reinstate.dev/install.ps1 | iex
$ErrorActionPreference = "Stop"

$Version = "v0.5.0-rc.2"
$PinnedInstallerSha256 = "02c68984964556e7c685a275bde72dc812162e0b898be0f26718a0813efc0dfe"
$InstallerUrl = "https://raw.githubusercontent.com/HarjjotSinghh/reinstate/${Version}/scripts/install.ps1"
$DefaultBase = if ($env:LOCALAPPDATA) { $env:LOCALAPPDATA } else { $env:USERPROFILE }
$InstallDir = if ($env:INSTALL_DIR) {
    $env:INSTALL_DIR
} else {
    Join-Path $DefaultBase "Programs\Reinstate\bin"
}

function Normalize-PathEntry([string]$Entry) {
    if ([string]::IsNullOrWhiteSpace($Entry)) {
        return ""
    }
    $trimmed = $Entry.Trim().Trim('"')
    $expanded = [Environment]::ExpandEnvironmentVariables($trimmed)
    try {
        $normalized = [IO.Path]::GetFullPath($expanded)
    } catch {
        $normalized = $expanded
    }
    return $normalized.TrimEnd("\", "/")
}

function Test-PathContains([string]$PathValue, [string]$Entry) {
    $target = Normalize-PathEntry $Entry
    foreach ($candidate in ($PathValue -split ";")) {
        $normalizedCandidate = Normalize-PathEntry $candidate
        if ([string]::Equals($normalizedCandidate, $target, [StringComparison]::OrdinalIgnoreCase)) {
            return $true
        }
    }
    return $false
}

function Get-Sha256([string]$Path) {
    $stream = [IO.File]::OpenRead($Path)
    try {
        $sha256 = [Security.Cryptography.SHA256]::Create()
        try {
            $hashBytes = $sha256.ComputeHash($stream)
        } finally {
            $sha256.Dispose()
        }
    } finally {
        $stream.Dispose()
    }
    return [BitConverter]::ToString($hashBytes).Replace("-", "").ToLowerInvariant()
}

$Tmp = New-Item -ItemType Directory -Path ([IO.Path]::GetTempPath()) -Name ("reinstate-bootstrap-" + [guid]::NewGuid().ToString())
try {
    $installerPath = Join-Path $Tmp "install.ps1"
    Write-Host "Downloading verified Reinstate installer $Version..."
    Invoke-WebRequest -UseBasicParsing -Uri $InstallerUrl -OutFile $installerPath

    $actualInstallerSha256 = Get-Sha256 $installerPath
    if ($actualInstallerSha256 -ne $PinnedInstallerSha256.ToLowerInvariant()) {
        throw "installer checksum mismatch (expected $PinnedInstallerSha256, actual $actualInstallerSha256)"
    }
    Write-Host "installer checksum ok"

    $hadVersion = Test-Path Env:REINSTATE_VERSION
    $previousVersion = $env:REINSTATE_VERSION
    $hadReleaseBase = Test-Path Env:REINSTATE_RELEASE_BASE_URL
    $previousReleaseBase = $env:REINSTATE_RELEASE_BASE_URL
    $hadSkipVersionCheck = Test-Path Env:REINSTATE_SKIP_VERSION_CHECK
    $previousSkipVersionCheck = $env:REINSTATE_SKIP_VERSION_CHECK
    try {
        $env:REINSTATE_VERSION = $Version
        $env:REINSTATE_RELEASE_BASE_URL = ""
        $env:REINSTATE_SKIP_VERSION_CHECK = "0"
        $installer = [ScriptBlock]::Create([IO.File]::ReadAllText($installerPath))
        & $installer
    } finally {
        if ($hadVersion) {
            $env:REINSTATE_VERSION = $previousVersion
        } else {
            Remove-Item Env:REINSTATE_VERSION -ErrorAction SilentlyContinue
        }
        if ($hadReleaseBase) {
            $env:REINSTATE_RELEASE_BASE_URL = $previousReleaseBase
        } else {
            Remove-Item Env:REINSTATE_RELEASE_BASE_URL -ErrorAction SilentlyContinue
        }
        if ($hadSkipVersionCheck) {
            $env:REINSTATE_SKIP_VERSION_CHECK = $previousSkipVersionCheck
        } else {
            Remove-Item Env:REINSTATE_SKIP_VERSION_CHECK -ErrorAction SilentlyContinue
        }
    }

    $pathScope = if ($env:REINSTATE_BOOTSTRAP_PATH_SCOPE) {
        $env:REINSTATE_BOOTSTRAP_PATH_SCOPE
    } else {
        "User"
    }
    if ($pathScope -notin @("User", "Process")) {
        throw "REINSTATE_BOOTSTRAP_PATH_SCOPE must be User or Process"
    }

    $savedPath = [Environment]::GetEnvironmentVariable("Path", $pathScope)
    if (-not (Test-PathContains $savedPath $InstallDir) -and $env:REINSTATE_SKIP_PATH_UPDATE -ne "1") {
        $newPath = if ([string]::IsNullOrWhiteSpace($savedPath)) {
            $InstallDir
        } else {
            $savedPath.TrimEnd(";") + ";" + $InstallDir
        }
        [Environment]::SetEnvironmentVariable("Path", $newPath, $pathScope)
        if ($pathScope -eq "User") {
            Write-Host "Added $InstallDir to user PATH."
        } else {
            Write-Host "Added $InstallDir to current process PATH."
        }
    }

    if (-not (Test-PathContains $env:PATH $InstallDir)) {
        $env:PATH = if ([string]::IsNullOrWhiteSpace($env:PATH)) {
            $InstallDir
        } else {
            $InstallDir + ";" + $env:PATH
        }
    }

    Write-Host "Next: rein init"
}
finally {
    Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}
