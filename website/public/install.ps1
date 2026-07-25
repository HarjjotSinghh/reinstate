# Reinstate public bootstrap — pins and verifies one canonical release installer.
# Usage: irm https://reinstate.dev/install.ps1 | iex
$ErrorActionPreference = "Stop"

$Version = "v0.1.0-rc.2"
$PinnedInstallerSha256 = "4d6e422f36ef20f4378786b34a75c042223ebff3db13b3a05f7a97e1126d6781"
$Origin = if ($env:REINSTATE_BOOTSTRAP_ORIGIN) {
    $env:REINSTATE_BOOTSTRAP_ORIGIN.TrimEnd("/")
} else {
    "https://raw.githubusercontent.com/HarjjotSinghh/reinstate"
}
$ExpectedInstallerSha256 = if ($env:REINSTATE_BOOTSTRAP_INSTALLER_SHA256) {
    $env:REINSTATE_BOOTSTRAP_INSTALLER_SHA256
} else {
    $PinnedInstallerSha256
}
$InstallerUrl = "$Origin/${Version}/scripts/install.ps1"
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

$Tmp = New-Item -ItemType Directory -Path ([IO.Path]::GetTempPath()) -Name ("reinstate-bootstrap-" + [guid]::NewGuid().ToString())
try {
    $installerPath = Join-Path $Tmp "install.ps1"
    Write-Host "Downloading verified Reinstate installer $Version..."
    Invoke-WebRequest -UseBasicParsing -Uri $InstallerUrl -OutFile $installerPath

    $actualInstallerSha256 = (Get-FileHash -Algorithm SHA256 -Path $installerPath).Hash.ToLowerInvariant()
    if ($actualInstallerSha256 -ne $ExpectedInstallerSha256.ToLowerInvariant()) {
        throw "installer checksum mismatch (expected $ExpectedInstallerSha256, actual $actualInstallerSha256)"
    }
    Write-Host "installer checksum ok"

    $hadVersion = Test-Path Env:REINSTATE_VERSION
    $previousVersion = $env:REINSTATE_VERSION
    try {
        $env:REINSTATE_VERSION = $Version
        $installer = [ScriptBlock]::Create([IO.File]::ReadAllText($installerPath))
        & $installer
    } finally {
        if ($hadVersion) {
            $env:REINSTATE_VERSION = $previousVersion
        } else {
            Remove-Item Env:REINSTATE_VERSION -ErrorAction SilentlyContinue
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
