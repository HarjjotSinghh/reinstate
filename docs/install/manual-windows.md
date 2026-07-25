# Manual installation — native Windows

Use PowerShell. Replace `<VERSION>` with an exact tag and
`<VERSION_NO_V>` with the version without `v`.

```powershell
$Version = "<VERSION>"
$AssetVersion = "<VERSION_NO_V>"
$Base = "https://github.com/HarjjotSinghh/reinstate/releases/download/$Version"
$Asset = "reinstate_${AssetVersion}_windows_amd64.zip"
$Temp = Join-Path $env:TEMP "reinstate-$AssetVersion"
New-Item -ItemType Directory -Force $Temp | Out-Null

Invoke-WebRequest "$Base/$Asset" -OutFile (Join-Path $Temp $Asset)
Invoke-WebRequest "$Base/checksums.txt" -OutFile (Join-Path $Temp "checksums.txt")
$Expected = (Get-Content (Join-Path $Temp "checksums.txt") |
  Where-Object { $_ -match [regex]::Escape($Asset) } |
  ForEach-Object { ($_ -split '\s+')[0] } |
  Select-Object -First 1)
if (-not $Expected) { throw "checksum entry missing" }
$Actual = (Get-FileHash (Join-Path $Temp $Asset) -Algorithm SHA256).Hash
if ($Expected.ToLower() -ne $Actual.ToLower()) { throw "checksum mismatch" }

Expand-Archive (Join-Path $Temp $Asset) (Join-Path $Temp "extract") -Force
$Install = Join-Path $env:LOCALAPPDATA "Programs\Reinstate\bin"
New-Item -ItemType Directory -Force $Install | Out-Null
$Candidate = Join-Path $Install "reinstate.new.exe"
$Destination = Join-Path $Install "reinstate.exe"
Copy-Item (Join-Path $Temp "extract\reinstate.exe") $Candidate -Force
$Reported = (& $Candidate version --json | Out-String | ConvertFrom-Json).version
if ($Reported -ne $AssetVersion) { throw "binary version mismatch" }
if (Test-Path $Destination) {
  $Answer = Read-Host "Replace the existing Reinstate install? [y/N]"
  if ($Answer -notmatch '^(y|yes)$') { Remove-Item $Candidate; throw "cancelled" }
}
Move-Item $Candidate $Destination -Force
Copy-Item $Destination (Join-Path $Install "rein.exe") -Force
& $Destination version
```

Add `%LOCALAPPDATA%\Programs\Reinstate\bin` to the user `PATH` if needed.
Native Windows and WSL are separate Reinstate devices; do not share one
agent-state directory.
