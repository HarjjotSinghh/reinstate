# OpenCode synthetic storage (Windows)

Vendor root (confirmed):

`%USERPROFILE%\.local\share\opencode`

Evidence:

1. Official troubleshooting docs: paste
   `%USERPROFILE%\.local\share\opencode` in Win+R
2. Source `packages/core/src/global.ts`:
   `Path.data = path.join(xdgData, "opencode")`
3. `xdg-basedir` resolves `xdgData` as
   `XDG_DATA_HOME || path.join(homedir, ".local", "share")`
   on **all** platforms, including Windows — not `%LOCALAPPDATA%`

This fixture tree is the relative layout under that root's `storage/`.
