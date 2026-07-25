## Summary

<!-- What does this PR change and why? -->

## Type of change

- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation
- [ ] Chore / CI / refactor
- [ ] New or updated adapter

## Related issues

<!-- Closes #123 -->

## Test plan

- [ ] `make verify` passes
- [ ] `make build` succeeds
- [ ] Added/updated tests for behavior changes
- [ ] Manual verification steps (describe below)

```text
# steps
```

## Checklist

- [ ] I have read [CONTRIBUTING.md](../CONTRIBUTING.md)
- [ ] Docs updated if user-facing
- [ ] CHANGELOG updated for user-visible behavior
- [ ] Config/schema migration impact documented, or marked not applicable
- [ ] Compatibility impact recorded for each affected OS/agent pair
- [ ] Security impact reviewed (credentials, plaintext, paths, permissions)
- [ ] New fixtures are synthetic and pass `make fixture-scan`
- [ ] No secrets, credentials, or real session transcripts committed
- [ ] Conventional commit title preferred (`feat:`, `fix:`, `docs:`, …)

## Adapter PRs only

- [ ] Synthetic fixtures under `testdata/adapters/<name>/`
- [ ] Credential paths excluded
- [ ] Exact tested agent version/layout recorded
- [ ] Support matrix updated in `docs/adapters.md`, `docs/compatibility.md`, and README
