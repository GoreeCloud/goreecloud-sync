# Contributing

GoreeCloud Sync is in early development. Changes should preserve the documented product boundaries rather than treating every desirable transfer feature as interchangeable.

## Development workflow

1. Create a focused branch from the current approved base.
2. Keep changes scoped to one coherent objective.
3. Add or update tests for behavior changes.
4. Run formatting, tests, vetting, and build validation.
5. Update `FEATURES.md` when implementation status changes.
6. Update architecture, protocol, security, or competitive documentation when a change materially affects those areas.
7. Open a pull request and preserve unresolved production-readiness gates in the PR description.

## Local validation

```bash
gofmt -w ./cmd ./internal
go test ./...
go vet ./...
go build ./cmd/goreecloud-sync
```

## Coding principles

- Prefer the smallest dependency set that safely solves the problem.
- Keep authorization checks explicit.
- Keep network reachability separate from access authorization.
- Do not log file contents or secrets.
- Return errors rather than silently ignoring destructive or integrity-relevant failures.
- Preserve backward compatibility after protocol versions become stable.
- Avoid premature abstraction and unnecessary services.
- Keep planned functionality clearly separate from implemented functionality.

## Security-sensitive changes

Changes involving cryptography, identity, enrollment, authorization, share links, relay behavior, path handling, filesystem writes, or deletion propagation require security-focused tests and review before release acceptance.

## Licensing and provenance

Contributions must be compatible with the repository's AGPL-3.0 license and must not introduce code, assets, or dependencies that GoreeCloud does not have the right to redistribute.
