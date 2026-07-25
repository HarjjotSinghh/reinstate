param(
    [string]$DistDir = "dist"
)

$ErrorActionPreference = "Stop"
$DistDir = (Resolve-Path $DistDir).Path
$Checksums = Join-Path $DistDir "checksums.txt"
if (-not (Test-Path $Checksums)) {
    throw "missing checksums.txt"
}

foreach ($line in Get-Content $Checksums) {
    if ($line -notmatch '^([0-9a-fA-F]{64})\s+(.+)$') {
        throw "invalid checksum line: $line"
    }
    $expected = $Matches[1].ToLowerInvariant()
    $artifact = Join-Path $DistDir $Matches[2]
    if (-not (Test-Path $artifact)) {
        throw "missing checksummed artifact: $artifact"
    }
    $actual = (Get-FileHash -Algorithm SHA256 $artifact).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "checksum mismatch: $artifact"
    }
}

$binaryArchives = @(
    Get-ChildItem $DistDir -File |
        Where-Object { $_.Name -match '^reinstate_.+_(darwin|linux)_(amd64|arm64)\.tar\.gz$' -or $_.Name -match '^reinstate_.+_windows_amd64\.zip$' }
)
if ($binaryArchives.Count -ne 5) {
    throw "expected 5 binary archives, found $($binaryArchives.Count)"
}

foreach ($archive in $binaryArchives) {
    if (-not (Test-Path ($archive.FullName + ".sbom.json"))) {
        throw "missing SBOM for $($archive.Name)"
    }
    if ($archive.Extension -eq ".zip") {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        $zip = [System.IO.Compression.ZipFile]::OpenRead($archive.FullName)
        try {
            $entries = @($zip.Entries | ForEach-Object { $_.FullName })
        } finally {
            $zip.Dispose()
        }
    } else {
        $entries = @(& tar -tzf $archive.FullName)
        if ($LASTEXITCODE -ne 0) {
            throw "cannot inspect $($archive.Name)"
        }
    }
    foreach ($required in @("LICENSE", "NOTICE", "README.md", "CHANGELOG.md")) {
        $requiredPattern = '(^|/)' + [regex]::Escape($required) + '$'
        if (-not ($entries | Where-Object { $_ -match $requiredPattern })) {
            throw "$($archive.Name) is missing $required"
        }
    }
    if (-not ($entries | Where-Object { $_ -match '(^|/)reinstate(\.exe)?$' })) {
        throw "$($archive.Name) is missing the Reinstate binary"
    }
}

$sourceArchives = @(Get-ChildItem $DistDir -File -Filter "reinstate_*_source.tar.gz")
if ($sourceArchives.Count -ne 1) {
    throw "expected one source archive, found $($sourceArchives.Count)"
}

Write-Host "release artifacts verified"
