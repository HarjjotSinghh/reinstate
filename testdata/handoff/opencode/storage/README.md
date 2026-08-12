# OpenCode storage-tier handoff fixture

Synthetic OpenCode data root (`Global.Path.data`). Layout mirrors
`~/.local/share/opencode` (Windows: `%USERPROFILE%\.local\share\opencode`):

```
storage/session/<project-id>/<session-id>.json
storage/message/<session-id>/<message-id>.json
storage/part/<message-id>/<part-id>.json
```

Bodies live in parts, not message Info files (MessageV2).
See `docs/research/2026-08-12-phase-4-r1-r2-r3.md`.
