# Manual installation — WSL2

WSL2 uses the Linux archive and its own Reinstate home. Do not point it at the
native Windows `%USERPROFILE%\.reinstate` directory.

Replace `<VERSION>` and `<VERSION_NO_V>`, then:

```bash
VERSION=<VERSION>
ASSET_VERSION=<VERSION_NO_V>
ARCH=$(uname -m)
[ "$ARCH" = "x86_64" ] && ARCH=amd64
[ "$ARCH" = "aarch64" ] && ARCH=arm64
BASE="https://github.com/HarjjotSinghh/reinstate/releases/download/$VERSION"
ASSET="reinstate_${ASSET_VERSION}_linux_${ARCH}.tar.gz"

curl -fSLO "$BASE/$ASSET"
curl -fSLO "$BASE/checksums.txt"
grep "  $ASSET\$" checksums.txt | sha256sum -c -
tar -xzf "$ASSET"
mkdir -p "$HOME/.local/bin"
install -m 755 reinstate "$HOME/.local/bin/reinstate"
ln -sfn reinstate "$HOME/.local/bin/rein"
"$HOME/.local/bin/reinstate" version
```

WSL1 is unsupported. Map Windows-mounted projects explicitly, for example:

```bash
rein init --project github.com/acme/app=/mnt/c/Users/me/code/app
```
