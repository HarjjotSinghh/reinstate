# Reinstate native Windows installer — exact release, SHA-256 verified, no elevation.
# Public: irm https://reinstate.dev/install.ps1 | iex
# Exact-tag audit: $env:REINSTATE_VERSION="vX.Y.Z"; irm https://raw.githubusercontent.com/HarjjotSinghh/reinstate/vX.Y.Z/scripts/install.ps1 | iex
$ErrorActionPreference = "Stop"

$Repo = "HarjjotSinghh/reinstate"
$BinName = "reinstate"
$DefaultBase = if ($env:LOCALAPPDATA) { $env:LOCALAPPDATA } else { $env:USERPROFILE }
$InstallDir = if ($env:INSTALL_DIR) {
    $env:INSTALL_DIR
} else {
    Join-Path $DefaultBase "Programs\Reinstate\bin"
}
$Version = $env:REINSTATE_VERSION

if (-not $Version) {
    throw "REINSTATE_VERSION is required; refusing an unpinned latest release"
}
if ($Version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$') {
    throw "REINSTATE_VERSION must be an exact v-prefixed SemVer"
}
if (-not [Environment]::Is64BitOperatingSystem) {
    throw "unsupported architecture: Reinstate requires 64-bit Windows"
}

$AssetVersion = $Version.Substring(1)
$Asset = "${BinName}_${AssetVersion}_windows_amd64.zip"
$Base = if ($env:REINSTATE_RELEASE_BASE_URL) {
    $env:REINSTATE_RELEASE_BASE_URL
} else {
    "https://github.com/$Repo/releases/download/$Version"
}
$Tmp = New-Item -ItemType Directory -Path ([System.IO.Path]::GetTempPath()) -Name ("reinstate-install-" + [guid]::NewGuid().ToString())

function Get-ReinstateVersion([string]$Path) {
    try {
        $result = (& $Path version --json 2>$null | Out-String | ConvertFrom-Json).version
        return [string]$result
    } catch {
        return ""
    }
}

function Confirm-Replacement([string]$ExistingVersion) {
    if ($env:REINSTATE_CONFIRM_REPLACE -eq "1") {
        return
    }
    if ($Host.UI -and $Host.UI.RawUI) {
        $answer = Read-Host "Replace Reinstate $ExistingVersion with $AssetVersion? [y/N]"
        if ($answer -match '^(y|yes)$') {
            return
        }
    }
    throw "refusing to replace existing Reinstate $ExistingVersion; set REINSTATE_CONFIRM_REPLACE=1 after reviewing the version change"
}

try {
    $zipPath = Join-Path $Tmp $Asset
    $sumPath = Join-Path $Tmp "checksums.txt"
    Invoke-WebRequest -Uri "$Base/$Asset" -OutFile $zipPath
    Invoke-WebRequest -Uri "$Base/checksums.txt" -OutFile $sumPath

    $expected = Get-Content $sumPath |
        Where-Object { $_ -match ('\s' + [regex]::Escape($Asset) + '$') } |
        ForEach-Object { ($_ -split '\s+')[0] } |
        Select-Object -First 1
    if (-not $expected) {
        throw "checksum entry missing for $Asset"
    }
    $actual = (Get-FileHash -Algorithm SHA256 -Path $zipPath).Hash.ToLowerInvariant()
    if ($expected.ToLowerInvariant() -ne $actual) {
        throw "checksum mismatch for $Asset"
    }
    Write-Host "checksum ok"

    $extractDir = Join-Path $Tmp "extract"
    Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force
    $bin = Get-ChildItem -Path $extractDir -Recurse -Filter "$BinName.exe" | Select-Object -First 1
    if (-not $bin) {
        throw "binary not found in archive"
    }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    $dest = Join-Path $InstallDir "$BinName.exe"
    $alias = Join-Path $InstallDir "rein.exe"
    $candidate = Join-Path $InstallDir (".reinstate." + [guid]::NewGuid().ToString() + ".exe")
    Copy-Item $bin.FullName $candidate
    try {
        if ($env:REINSTATE_SKIP_VERSION_CHECK -ne "1") {
            $candidateVersion = Get-ReinstateVersion $candidate
            if ($candidateVersion -ne $AssetVersion) {
                throw "downloaded binary reported version $candidateVersion; expected $AssetVersion"
            }
        }

        if (Test-Path $dest) {
            $existingVersion = Get-ReinstateVersion $dest
            if ($existingVersion -eq $AssetVersion) {
                Copy-Item -Force $dest $alias
                Write-Host "Reinstate $Version is already installed at $dest"
                return
            }
            Confirm-Replacement $(if ($existingVersion) { $existingVersion } else { "unknown" })
        }

        Move-Item -Force $candidate $dest
        Copy-Item -Force $dest $alias
    } finally {
        Remove-Item -Force $candidate -ErrorAction SilentlyContinue
    }

    Write-Host "Installed $BinName $Version -> $dest"
    Write-Host "Installed rein alias -> $alias"
    $pathEntries = $env:PATH -split ';'
    if ($pathEntries -notcontains $InstallDir) {
        Write-Host "Add $InstallDir to your user PATH, then open a new terminal."
    }
}
finally {
    Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}
