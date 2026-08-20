# Emit AGENT-PROBE-V1 via the installed Reinstate binary.
# Output is byte-identical to: rein doctor --agents --json
$ErrorActionPreference = 'Stop'
$bin = Get-Command rein -ErrorAction SilentlyContinue
if (-not $bin) {
    $bin = Get-Command reinstate -ErrorAction SilentlyContinue
}
if (-not $bin) {
    Write-Error 'rein/reinstate not found on PATH'
    exit 1
}
# Build the argument list first. Writing the concatenation inline passed '+'
# and $args to the binary as separate arguments, so the wrapper always exited
# with a usage error and emitted nothing.
$argv = @('doctor', '--agents', '--json') + $args
& $bin.Source @argv
exit $LASTEXITCODE
