# Manual installation — macOS

Replace `<VERSION>` with an exact published tag and `<VERSION_NO_V>` with the
same version without its leading `v`.

```bash
VERSION=<VERSION>
ASSET_VERSION=<VERSION_NO_V>
ARCH=$(uname -m)
[ "$ARCH" = "x86_64" ] && ARCH=amd64
[ "$ARCH" = "arm64" ] && ARCH=arm64
BASE="https://github.com/HarjjotSinghh/reinstate/releases/download/$VERSION"
ASSET="reinstate_${ASSET_VERSION}_darwin_${ARCH}.tar.gz"

curl -fSLO "$BASE/$ASSET"
curl -fSLO "$BASE/checksums.txt"
grep "  $ASSET\$" checksums.txt | shasum -a 256 -c -
tar -xzf "$ASSET"
mkdir -p "$HOME/.local/bin"
install -m 755 reinstate "$HOME/.local/bin/reinstate"
ln -sfn reinstate "$HOME/.local/bin/rein"
"$HOME/.local/bin/rein" version
```

Do not continue after a missing asset/checksum or checksum failure. Add
`$HOME/.local/bin` to `PATH` if necessary, then follow
[verification](verify-installation.md).
