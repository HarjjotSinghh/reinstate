# Grok Build synthetic sessions (macOS)

Layout confirmed from xAI vendor docs
(`crates/codegen/xai-grok-pager/docs/user-guide/17-sessions.md`) and
`encode_cwd_dirname` in `xai-grok-config/src/paths.rs`:

```
~/.grok/sessions/<url-encoded-cwd>/<session-uuid>/
  summary.json
  updates.jsonl
  chat_history.jsonl
  compaction_checkpoints/
```

Workspace key for `/Users/fixture-user/code/demo` is the URL-encoded
dirname `%2FUsers%2Ffixture-user%2Fcode%2Fdemo`.

This fixture's `chat_history.jsonl` is the **post-`/compact`** active
history. Pre-compaction turns are not left as earlier lines in that file
(`replace_chat_history` atomically rewrites it). They are preserved under
`compaction_requests/` (full request payload). A `CompactionCheckpoint`
marker remains in append-only `updates.jsonl`, and the compacted payload
lives under `compaction_checkpoints/`.
