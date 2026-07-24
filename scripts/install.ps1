# Reinstate Windows installer — downloads release zip and verifies SHA-256.
# Usage: irm https://raw.githubusercontent.com/HarjjotSinghh/reinstate/main/scripts/install.ps1 | iex
$ErrorActionPreference = "Stop"
$Repo = "HarjjotSinghh/reinstate"
$BinName = "reinstate"
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:USERPROFILE ".local\bin" }
$Version = $env:REINSTATE_VERSION

$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { throw "unsupported arch" }
$os = "windows"

if (-not $Version) {
  $rel = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
  $Version = $rel.tag_name
}
if (-not $Version) { throw "No GitHub release found" }

$Asset = "${BinName}_${Version}_${os}_${arch}.zip"
$Base = "https://github.com/$Repo/releases/download/$Version"
$Tmp = New-Item -ItemType Directory -Path ([System.IO.Path]::GetTempPath()) -Name ("reinstate-install-" + [guid]::NewGuid().ToString())

try {
  $zipPath = Join-Path $Tmp $Asset
  $sumPath = Join-Path $Tmp "checksums.txt"
  Invoke-WebRequest -Uri "$Base/$Asset" -OutFile $zipPath
  Invoke-WebRequest -Uri "$Base/checksums.txt" -OutFile $sumPath

  $expected = (Get-Content $sumPath | Where-Object { $_ -match [regex]::Escape($Asset) } | ForEach-Object { ($_ -split '\s+')[0] } | Select-Object -First 1)
  if (-not $expected) { throw "checksum entry missing for $Asset" }
  $actual = (Get-FileHash -Algorithm SHA256 -Path $zipPath).Hash.ToLower()
  if ($expected.ToLower() -ne $actual) { throw "checksum mismatch: expected $expected got $actual" }
  Write-Host "checksum ok"

  Expand-Archive -Path $zipPath -DestinationPath (Join-Path $Tmp "extract") -Force
  $bin = Get-ChildItem -Path (Join-Path $Tmp "extract") -Recurse -Filter "$BinName.exe" | Select-Object -First 1
  if (-not $bin) { throw "binary not found in archive" }

  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
  $dest = Join-Path $InstallDir "$BinName.exe"
  Copy-Item -Force $bin.FullName $dest
  Write-Host "Installed $BinName $Version → $dest"
  & $dest version
}
finally {
  Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}
