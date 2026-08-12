# OpenCode synthetic storage (macOS)

Mirrors vendor layout under `Global.Path.data/storage` where
`Global.Path.data` is `~/.local/share/opencode` (see
`packages/core/src/global.ts` + `xdg-basedir`).

Evidence: anomalyco/opencode `Storage.write(key)` →
`<data>/storage/<key...>.json`; message keys are
`["message", sessionID, messageID]`.
