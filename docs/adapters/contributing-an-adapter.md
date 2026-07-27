# Contributing an adapter

New adapters are post–Phase 1 work and require an issue or RFC before code.

An adapter contribution must:

1. detect an exact vendor version and layout;
2. fail closed for unknown versions;
3. discover metadata without printing transcript content;
4. define credential/cache exclusions before export;
5. stream bounded records rather than reading unbounded files;
6. preserve the vendor's native resume identity and path layout;
7. back up and atomically replace existing files;
8. add deterministic macOS, native Windows, and WSL2 fixtures;
9. update `docs/adapters.md`, `docs/compatibility.md`, README, and CHANGELOG.

Use only synthetic fixtures described in
[Testing and fixture policy](../contributing/testing.md). A green parser test
does not establish support: record the exact real vendor version and each
verified device journey separately.

## Configuration adapters (later phase)

Universal configuration uses a related but distinct contract. A configuration
adapter must declare support per capability (MCP, skill, hook/loop, plugin,
marketplace, safe setting), round-trip only understood fields, preserve
unmanaged native config, preview and back up writes, and report unsupported or
lossy mappings. Never infer configuration support from session support.

See [Universal agent configuration](../universal-configuration.md).
