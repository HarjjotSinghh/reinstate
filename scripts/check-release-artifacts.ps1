#Requires -Version 5.1
<#
.SYNOPSIS
  PowerShell-native parity for scripts/check-release-artifacts.sh.

.DESCRIPTION
  Validates a staged release directory without GNU tools (sha256sum, unzip, jq).
  Windows acceptance must treat this script as the required artifact gate when
  the POSIX helper is unavailable.
#>
param(
    [string]$DistDir = "dist"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Assert-MatchCount {
    param(
        [Parameter(Mandatory = $true)][object[]]$Items,
        [Parameter(Mandatory = $true)][int]$Expected,
        [Parameter(Mandatory = $true)][string]$Label
    )
    if ($Items.Count -ne $Expected) {
        throw "$Label expected $Expected, found $($Items.Count)"
    }
}

function Get-RelativeArchiveEntries {
    param([Parameter(Mandatory = $true)][string]$ArchivePath)

    if ($ArchivePath -like "*.zip") {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        $zip = [System.IO.Compression.ZipFile]::OpenRead($ArchivePath)
        try {
            return @($zip.Entries | ForEach-Object { $_.FullName.Replace('\', '/') })
        } finally {
            $zip.Dispose()
        }
    }

    if ($ArchivePath -like "*.tar.gz" -or $ArchivePath -like "*.tgz") {
        $nativeTar = Join-Path $env:SystemRoot "System32\tar.exe"
        if (-not (Test-Path -LiteralPath $nativeTar)) {
            throw "native Windows tar.exe is unavailable: $nativeTar"
        }
        # Do not resolve tar from PATH: an MSYS2 tar treats PowerShell drive
        # paths as remote archive names.
        $entries = @(& $nativeTar -tzf $ArchivePath 2>$null)
        if ($LASTEXITCODE -ne 0) {
            throw "cannot list archive members: $ArchivePath"
        }
        return @($entries | ForEach-Object { "$_".Replace('\', '/') })
    }

    throw "unsupported archive type: $ArchivePath"
}

$resolvedDist = (Resolve-Path -LiteralPath $DistDir).Path
$checksumsPath = Join-Path $resolvedDist "checksums.txt"
if (-not (Test-Path -LiteralPath $checksumsPath)) {
    throw "missing checksums.txt in $resolvedDist"
}

$checksumNames = New-Object "System.Collections.Generic.List[string]"
foreach ($line in Get-Content -LiteralPath $checksumsPath) {
    if ($line -notmatch '^([0-9a-fA-F]{64})\s+(.+)$') {
        throw "invalid checksum line: $line"
    }
    $expected = $Matches[1].ToLowerInvariant()
    $name = $Matches[2].Trim()
    if ($name -match '[\\/]' -or $name -eq "." -or $name -eq "..") {
        throw "refusing unsafe checksum path: $name"
    }
    $artifact = Join-Path $resolvedDist $name
    if (-not (Test-Path -LiteralPath $artifact)) {
        throw "missing checksummed artifact: $name"
    }
    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $artifact).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "checksum mismatch: $name"
    }
    $checksumNames.Add($name) | Out-Null
}

$archives = @(
    Get-ChildItem -LiteralPath $resolvedDist -File |
        Where-Object {
            $_.Name -match '^reinstate_.+_(darwin|linux)_(amd64|arm64)\.tar\.gz$' -or
            $_.Name -match '^reinstate_.+_windows_amd64\.zip$'
        } |
        Sort-Object Name
)
Assert-MatchCount -Items $archives -Expected 5 -Label "binary archives"

$rawBinaries = @(
    $checksumNames |
        Where-Object {
            $_ -match '^reinstate_[^ ]+_(darwin|linux)_(amd64|arm64)$' -or
            $_ -match '^reinstate_[^ ]+_windows_amd64\.exe$'
        } |
        Sort-Object
)
Assert-MatchCount -Items $rawBinaries -Expected 5 -Label "raw binaries"
foreach ($name in $rawBinaries) {
    $path = Join-Path $resolvedDist $name
    if (-not (Test-Path -LiteralPath $path) -or (Get-Item -LiteralPath $path).Length -le 0) {
        throw "raw binary missing or empty: $name"
    }
}

$linuxPackages = @(
    Get-ChildItem -LiteralPath $resolvedDist -File |
        Where-Object {
            $_.Name -match '^reinstate_.+_linux_.+\.(apk|deb|rpm)$' -or
            $_.Name -match '^reinstate_.+_linux_.+\.pkg\.tar\.zst$'
        } |
        Sort-Object Name
)
Assert-MatchCount -Items $linuxPackages -Expected 8 -Label "linux packages"

foreach ($archive in $archives) {
    $sbom = $archive.FullName + ".sbom.json"
    if (-not (Test-Path -LiteralPath $sbom)) {
        throw "missing SBOM for $($archive.Name)"
    }
    $entries = Get-RelativeArchiveEntries -ArchivePath $archive.FullName
    if (-not ($entries | Where-Object { $_ -match '(^|/)reinstate(\.exe)?$' })) {
        throw "$($archive.Name) is missing the Reinstate binary"
    }
    if ($archive.Name -match '_windows_') {
        if (-not ($entries | Where-Object { $_ -match '(^|/)rein\.exe$' })) {
            throw "$($archive.Name) is missing rein.exe"
        }
    }
    foreach ($required in @("LICENSE", "NOTICE", "README.md", "CHANGELOG.md")) {
        $pattern = '(^|/)' + [regex]::Escape($required) + '$'
        if (-not ($entries | Where-Object { $_ -match $pattern })) {
            throw "$($archive.Name) is missing $required"
        }
    }
}

$sourceArchives = @(
    Get-ChildItem -LiteralPath $resolvedDist -File |
        Where-Object { $_.Name -match '^reinstate_.+_source\.tar\.gz$' } |
        Sort-Object Name
)
Assert-MatchCount -Items $sourceArchives -Expected 1 -Label "source archives"
$sourceEntries = Get-RelativeArchiveEntries -ArchivePath $sourceArchives[0].FullName
if (-not ($sourceEntries | Where-Object { $_ -match '^reinstate-[^/]+/go\.mod$' })) {
    throw "source archive is missing reinstate-*/go.mod"
}
if ($sourceEntries | Where-Object { $_ -match '(^|/)(\.git|bin|dist|\.env)(/|$)' }) {
    throw "source archive contains a forbidden generated or secret path"
}

Write-Host "release artifacts verified (PowerShell): $resolvedDist"
